#include <algorithm>
#include <cerrno>
#include <cstdint>
#include <chrono>
#include <fstream>
#include <iostream>
#include <map>
#include <string>
#include <system_error>
#include <vector>

#include <omp.h>
#include <nlohmann/json.hpp>
#include <clara/clara.hpp>
#include <mio/mio.hpp>

#include <seismic-cloud/seismic-cloud.hpp>

using json = nlohmann::json;

namespace {

struct config {
    bool help = false;

    std::string bin;
    std::string manifest;
    std::string surface;
    std::string input_dir = "./";
    bool timing = false;

    clara::Parser cli() {
        using namespace clara;

        return ExeName( bin )
            | Arg( manifest, "manifest" )
                 ( "Manifest" )
            | Opt( timing )
                 ( "Writing timing report" )
                 ["--time"]["-t"]
            | Opt( input_dir, "Input directory" )
                 ["--input-dir"]["-i"]
            | Help( this->help )
        ;
    }
};

void throw_errno() {
    auto errc = static_cast< std::errc >( errno );
    throw std::system_error( std::make_error_code( errc ) );
}

std::map< sc::point, std::vector< int > > bin( sc::dimension fragment_size,
                                               sc::dimension cube_size,
                                               const std::vector< sc::point >& xs ) {
    std::map< sc::point, std::vector< int > > ret;
    for (const auto& p : xs) {
        sc::point root = sc::global_to_root( p, fragment_size );
        sc::point local = sc::global_to_local( p, fragment_size );
        int pos = sc::point_to_offset( local, fragment_size );

        auto itr = ret.find(root);
        if (itr == ret.end()) {
          itr = ret.emplace(root, std::vector<int>{}).first;
        }
        itr->second.push_back(pos);
    }

    for (auto& kv : ret)
        std::sort( kv.second.begin(), kv.second.end() );

    return ret;
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

    json manifest;
    std::ifstream( cfg.input_dir + "/" + cfg.manifest ) >> manifest;

    sc::dimension fragment_size {
        manifest["fragment-xs"].get< int >(),
        manifest["fragment-ys"].get< int >(),
        manifest["fragment-zs"].get< int >(),
    };

    sc::dimension cube_size {
        manifest["cube-xs"].get< int >(),
        manifest["cube-ys"].get< int >(),
        manifest["cube-zs"].get< int >(),
    };

    auto start_time = std::chrono::system_clock::now();

    json meta;
    std::cin >> meta;
    std::cout << meta;
    int size = meta["size"];

    std::vector< sc::point > surface( size );

    auto points = std::vector< char >(size * sizeof(std::int32_t) * 3);
    std::cin.read(points.data(), points.size());

    [&surface] (char* ptr) {
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

    std::cout.sync_with_stdio(false);
    auto surface_time = std::chrono::system_clock::now();

    const auto bins = bin( fragment_size, cube_size, surface );
    auto bin_time = std::chrono::system_clock::now();

    decltype (bins.begin()) itr;
    #pragma omp parallel
    #pragma omp single nowait
    {
    for (itr = bins.begin(); itr != bins.end(); ++itr) {
        #pragma omp task firstprivate(itr)
        {
        const auto& key = itr->first;
        const auto& val = itr->second;
        const std::string path = manifest["basename"].get< std::string >()
                               + "-" + std::to_string( key.x )
                               + "-" + std::to_string( key.y )
                               + "-" + std::to_string( key.z )
                               + ".f32"
                               ;
        mio::mmap_source file( cfg.input_dir + "/" + path );

        const char* ptr = static_cast< const char* >( file.data() );

        for (const auto& off : val) {
            float f;
            std::memcpy( &f, ptr + off * 4, 4 );

            std::uint64_t global_offset =
                sc::local_to_global( off, fragment_size, cube_size, key );

            #pragma omp critical
            {
            std::cout.write((char*)&global_offset, sizeof(std::uint64_t));
            std::cout.write((char*)&f, sizeof(float));
            }
        }
        }
    }

    } // omp

    auto end_time = std::chrono::system_clock::now();

    if (cfg.timing) {
        using namespace std::chrono;
        auto surf =  duration_cast< milliseconds >(surface_time - start_time);
        auto bin =   duration_cast< milliseconds >(bin_time - surface_time);
        auto read =  duration_cast< milliseconds >(end_time - bin_time);
        auto total = duration_cast< milliseconds >(end_time - start_time);

        std::ofstream out( "./time", std::ofstream::app );

        out << "Fragment size: "
            << "x: "   << fragment_size.x
            << ", y: " << fragment_size.y
            << ", z: " << fragment_size.z << "\n"

            << "Parsing surface: "    << surf.count()  << "ms\n"
            << "Binning surface: "    << bin.count()   << "ms\n"
            << "Reading surface: "    << read.count()  << "ms\n"
            << "Total elapsed time: " << total.count() << "ms\n\n"
        ;
    }
}
