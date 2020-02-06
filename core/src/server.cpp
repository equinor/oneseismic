#include <algorithm>
#include <cstdint>
#include <iostream>
#include <string>

#include <clara/clara.hpp>
#include <grpc/grpc.h>
#include <grpc++/server.h>
#include <grpc++/server_builder.h>
#include <grpc++/server_context.h>

#include <seismic-cloud/geometry.hpp>
#include <seismic-cloud/fetch.hpp>
#include <seismic-cloud/url.hpp>

#include "core.grpc.pb.h"

namespace {

class Server : public oneseismic::Core::Service {
public:
    Server(int maxconn, const std::string& k, const std::string& sa) :
        maxconns(maxconn),
        key(k),
        storage_account(sa)
    {}

    grpc::Status Slice(grpc::ServerContext* ctx,
                       const oneseismic::SliceRequest* req,
                       oneseismic::SliceReply* writer);

private:
    int maxconns;
    std::string key;
    std::string storage_account;
};

template < typename Geometry >
std::int64_t indexof(const Geometry& g, std::size_t dim, std::int64_t lineno) {
    decltype(g.dim0().begin()) begin, end;
    switch (dim) {
        case 0:
            begin = g.dim0().begin();
            end   = g.dim0().end();
            break;
        case 1:
            begin = g.dim1().begin();
            end   = g.dim1().end();
            break;
        case 2:
            begin = g.dim2().begin();
            end   = g.dim2().end();
            break;

        default:
            throw std::invalid_argument(
                "expected dimension [0, 1, 2], was " + std::to_string(dim)
            );

    }

    const auto itr = std::find(begin, end, lineno);

    if (itr == end) {
        throw std::invalid_argument(
            "No such lineno " + std::to_string(lineno)
        );
    }

    return std::distance(begin, itr);
}

void assert_dim_positive(int x, int dim) {
    if (x >= 0) return;
    throw std::runtime_error(
        "expected fragment size in dim" + std::to_string(dim)
        + " >= 0, was " + std::to_string(x)
    );
}

std::vector< float >
slice_fragment(const std::vector< std::uint8_t >& fragment,
               sc::slice_layout lay,
               std::size_t pin) {
    auto outcome = std::vector< float >();
    outcome.resize(lay.iterations * lay.chunk_size);
    auto* dst = reinterpret_cast< std::uint8_t* >(outcome.data());
    auto* src = fragment.data() + lay.initial_skip * pin * sizeof(float);

    for (auto i = 0; i < lay.iterations; ++i) {
        std::memcpy(dst, src, lay.chunk_size * sizeof(float));
        dst += lay.substride * sizeof(float);
        src += lay.superstride * sizeof(float);
    }
    return outcome;
}

grpc::Status Server::Slice(
        grpc::ServerContext* ctx,
        const oneseismic::SliceRequest* req,
        oneseismic::SliceReply* writer) {
    try {
        sc::multifetch cm { this->maxconns };
        const auto dim = sc::dimension< 3 >(req->dim());
        const auto lineno = indexof(req->geometry(), dim.v, req->lineno());

        assert_dim_positive(req->dim0(), 0);
        assert_dim_positive(req->dim1(), 1);
        assert_dim_positive(req->dim2(), 2);
        const auto fragment_dimensions = sc::FS< 3 > {
            std::size_t(req->dim0()),
            std::size_t(req->dim1()),
            std::size_t(req->dim2()),
        };
        const auto cube_dimensions = sc::CS< 3 > {
            std::size_t(req->geometry().dim0().size()),
            std::size_t(req->geometry().dim1().size()),
            std::size_t(req->geometry().dim2().size())
        };
        auto gvt = sc::gvt< 3 >(cube_dimensions, fragment_dimensions);

        auto generator = sc::azure_request_generator(
            this->storage_account,
            req->geometry().guid(),
            "src",
            fragment_dimensions.string()
        );
        generator.timestamp();
        generator.shared_key(key);

        const auto fragment_ids = gvt.slice(dim, lineno);
        auto headers = generator.headers();
        auto jobs = std::vector< decltype(cm.enqueue("", headers)) >();
        for (const auto& id : fragment_ids) {
            const auto& frag = id.string();
            const auto url = generator.url(frag);
            const auto auth = generator.authorization(frag);

            headers.push_back(auth);
            jobs.push_back(cm.enqueue(url, headers));
            headers.pop_back();
        }

        auto job = std::async([this, &cm] { cm.run(); });

        /*
        * The shape of the output slice, i.e. a flattened cube. Simply set the
        * slice-dimension to 1
        */
        auto slice_gvt = [&] {
            auto cube = cube_dimensions;
            cube[dim.v] = 1;
            auto frag = fragment_dimensions;
            frag[dim.v] = 1;
            return sc::gvt< 3 >(cube, frag);
        }();

        writer->mutable_v()->Resize(slice_gvt.cube_shape().slice_samples(dim), 0);

        const auto src_layout = fragment_dimensions.slice_stride(dim);
        const auto pin = lineno % fragment_dimensions[dim];

        auto* out = writer->mutable_v()->begin();
        auto slice_dimensions = cube_dimensions;
        slice_dimensions[dim.v] = 1;
        const auto corner = sc::FP< 3 >{ 0, 0, 0 };

        for (std::size_t i = 0; i < fragment_ids.size(); ++i) {
            const auto& id = fragment_ids[i];
            const auto& frag = jobs.at(i).get();
            const auto tmp = slice_fragment(frag, src_layout, pin);

            auto dst_id = id;
            dst_id[dim.v] = 0;
            auto layout = slice_gvt.slice_stride(dim, dst_id);
            auto src = tmp.begin();
            auto dst = out + layout.initial_skip;
            for (auto i = 0; i < layout.iterations; ++i) {
                std::copy_n(src, layout.chunk_size, dst);
                src += layout.substride;
                dst += layout.superstride;
            }
        }

        switch (dim.v) {
            case 0:
                writer->set_dim0(cube_dimensions[1]);
                writer->set_dim1(cube_dimensions[2]);
                break;
            case 1:
                writer->set_dim0(cube_dimensions[0]);
                writer->set_dim1(cube_dimensions[2]);
                break;
            case 2:
                writer->set_dim0(cube_dimensions[0]);
                writer->set_dim1(cube_dimensions[1]);
                break;
            default:
                throw std::invalid_argument("wrong dimension (switch)");
        }

        return grpc::Status::OK;
    } catch (std::exception &e) {
        std::string msg = std::string("Slice failed: ") + e.what();
        std::cerr << msg << std::endl;
        return grpc::Status(grpc::StatusCode::CANCELLED, msg);
    }
}

}

int main(int argc, char** argv) {
    std::string address = "localhost:50051";
    std::string storage_account;
    std::string key;
    int max_connections = 4;

    // key = env["AZURE_ACCESS_KEY"]

    auto cli
        = clara::Opt(address, "bind")
            ["-b"]["--bind"]
            ("Address to bind, default = " + address)
        | clara::Opt(key, "key")
            ["-k"]["--key"]
            ("Access key")
        | clara::Opt(storage_account, "storage account")
            ["-A"]["--account"]["--storage-account"]
            ("Storage account for the blobs")
        | clara::Opt(max_connections, "maximum connections")
            ["-m"]["--max-connections"]["--maximum-connections"]
            ("Maximum curl connections, default = "
            + std::to_string(max_connections))
    ;

    auto result = cli.parse(clara::Args(argc, argv));
    if (!result) {
        std::cerr << "Error: " << result.errorMessage() << "\n";
        std::exit(EXIT_FAILURE);
    }


    if (key.empty())
        throw std::runtime_error("No Azure access key set");

    if (storage_account.empty())
        throw std::runtime_error("No storage account set");

    sc::fetch curl_context;
    Server service(max_connections, key, storage_account);

    grpc::ServerBuilder builder;
    builder.AddListeningPort(address, grpc::InsecureServerCredentials());
    builder.RegisterService(&service);
    std::unique_ptr<grpc::Server> server(builder.BuildAndStart());
    server->Wait();
}
