#include <fstream>

#include <clara/clara.hpp>
#include <mio/mio.hpp>
#if defined(HAVE_OPENMP)
    #include <omp.h>
#endif

#include <grpc/grpc.h>
#include <grpcpp/server.h>
#include <grpcpp/server_builder.h>
#include <grpcpp/server_context.h>

#include "proto/core.grpc.pb.h"
#include "proto/core.pb.h"

#include <seismic-cloud/seismic-cloud.hpp>

using grpc::ServerBuilder;
using grpc::Server;
using grpc::ServerContext;
using grpc::ServerWriter;
using grpc::Status;
using seismic_core::Core;
using seismic_core::ShatterLinkRequest;
using seismic_core::SurfaceRequest;
using seismic_core::ShatterReply;
using seismic_core::SurfaceReply;
using seismic_core::SurfaceValue;

namespace {
struct config {
    bool help = false;
    std::string bin;

    std::string input_dir = "./";

    clara::Parser cli() {
        using namespace clara;

        return ExeName( bin )
            | Opt( input_dir, "Input directory" )
                 ["--input-dir"]["-i"]
            | Help( this->help )
        ;
    }
};

std::vector< sc::point > read_surface( const std::string& filename ) {

    std::ifstream f( filename );

    f.seekg (0, f.end);
    std::size_t fsize = f.tellg();
    f.seekg (0, f.beg);

    std::vector< char > points( fsize );
    f.read( points.data(), points.size() );

    std::size_t size = points.size() / ( sizeof(std::int32_t) * 3 );

    std::vector< sc::point > surface( size );

    [&surface] (const char* ptr) {
        for (auto& p : surface) {
            std::int32_t x, y, z;
            std::memcpy(&x, ptr, sizeof(x));
            ptr += sizeof(x);
            std::memcpy(&y, ptr, sizeof(y));
            ptr += sizeof(y);
            std::memcpy(&z, ptr, sizeof(z));
            ptr += sizeof(z);

            p.x = x;
            p.y = y;
            p.z = z;
        }
    }(points.data());

    return surface;
}

class CoreImpl final : public Core::Service {
public:
    explicit CoreImpl( std::string input_dir ) : input_dir( input_dir + "/" ){};

    Status ShatterLink( ServerContext* context,
                        const ShatterLinkRequest* link,
                        ShatterReply* reply ) override {
        return Status::OK;
    }

    Status StitchSurface( ServerContext* context,
                          const SurfaceRequest* sfc,
                          SurfaceReply* reply ) override {

        sc::dimension cube_size;
        cube_size.x = sfc->cubexs();
        cube_size.y = sfc->cubeys();
        cube_size.z = sfc->cubezs();

        sc::dimension fragment_size;
        fragment_size.x = sfc->fragmentxs();
        fragment_size.y = sfc->fragmentys();
        fragment_size.z = sfc->fragmentzs();

        const auto surface = read_surface( this->input_dir + sfc->surface() );

        // TODO: error detection (and recovery?) when loading surface
        //if( <surface reading fails> ) return Status::FAILED_PRECONDITION;

        auto out = reply->mutable_surface();
        out->Reserve( surface.size() );

        const auto bins = sc::bin(fragment_size, cube_size, surface);

        #pragma omp parallel for
        for (std::size_t i = 0; i < bins.keys.size(); ++i) {
            const auto bin = bins.at(i);
            const auto& key = bin.key;
            const std::string path = sfc->basename()
                                   + "-" + std::to_string( key.x )
                                   + "-" + std::to_string( key.y )
                                   + "-" + std::to_string( key.z )
                                   + ".f32"
                                   ;
            mio::mmap_source file( this->input_dir + path );

            const auto bin_size = std::distance(bin.begin(), bin.end());

            std::vector< std::pair< std::int64_t, float> > output;
            output.reserve(bin_size);

            const char* in = static_cast< const char* >(file.data());

            for (const auto off : bin) {
                const std::uint64_t global_offset =
                    sc::local_to_global(off, fragment_size, cube_size, key);

                float value;
                std::memcpy(&value, in + off * 4, sizeof(float));

                output.push_back( std::make_pair(global_offset, value) );
            }

            #pragma omp critical
            {
            for( auto v : output ) {
                auto e = out->Add();
                e->set_i(v.first);
                e->set_v(v.second);
            }
            }
        } // omp

        return Status::OK;
    }
private:
    std::string input_dir;
};

void RunServer( const std::string& input_dir ) {
    std::string server_address("localhost:5005");
    CoreImpl service( input_dir );

    ServerBuilder builder;
    builder.AddListeningPort(server_address, grpc::InsecureServerCredentials());
    builder.RegisterService(&service);
    std::unique_ptr<Server> server(builder.BuildAndStart());
    std::cout << "Server listening on " << server_address << std::endl;
    server->Wait();
}
}

int main( int args, char** argv ) {
    config cfg;
    auto cli = cfg.cli();

    auto result = cli.parse( clara::Args( args, argv ) );

    if (cfg.help) {
        std::cerr << cli << "\n";
        std::exit( EXIT_SUCCESS );
    }

    if (!result) {
        std::cerr << result.errorMessage() << '\n';
        std::cerr << "usage: " << cli << '\n';
        std::exit( EXIT_FAILURE );
    }

    RunServer( cfg.input_dir );

    return 0;
}
