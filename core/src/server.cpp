#include <algorithm>
#include <cstdint>
#include <sstream>
#include <string>

#include <clara/clara.hpp>
#include <grpc/grpc.h>
#include <grpcpp/server.h>
#include <grpcpp/server_builder.h>
#include <grpcpp/server_context.h>

#include <seismic-cloud/seismic-cloud.hpp>
#include <seismic-cloud/fetch.hpp>
#include <seismic-cloud/url.hpp>

#include "core.grpc.pb.h"

namespace {

class Server : public oneseismic::Core::Service {
public:
    Server(int maxconn, const std::string& k, const std::string& sa) :
        cm(maxconn),
        key(k),
        storage_account(sa)
    {}

    grpc::Status Slice(grpc::ServerContext* ctx,
                       const oneseismic::SliceRequest* req,
                       oneseismic::SliceReply* writer);

private:
    sc::fetch curl_context;
    sc::multifetch cm;
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
            throw std::invalid_argument("wrong dimension (switch lineno)");

    }

    auto itr = std::find(begin, end, lineno);

    if (itr == end) {
        throw std::invalid_argument(
            "No such lineno " + std::to_string(lineno)
        );
    }

    return std::distance(begin, itr);
}

grpc::Status Server::Slice(
        grpc::ServerContext* ctx,
        const oneseismic::SliceRequest* req,
        oneseismic::SliceReply* writer) {

    const auto dim = sc::dimension< 3 >{ req->dim() };
    const auto lineno = indexof(req->geometry(), dim.v, req->lineno());
    std::cout << "Line index: " << lineno << "\n";

    const auto fragment_dimensions = sc::frag_dimension< 3 > {
        req->dim0(),
        req->dim1(),
        req->dim2(),
    };

    std::cout << "Fragment dimensions: " << fragment_dimensions << "\n";
    const auto cube_dimensions = sc::cube_dimension< 3 > {
        req->geometry().dim0().size(),
        req->geometry().dim1().size(),
        req->geometry().dim2().size()
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

    auto job = std::async([this] { this->cm.run(); });

    writer->mutable_v()->Resize(cube_dimensions.slice_samples(dim), 0);

    const auto src_stride = fragment_dimensions.slice_stride(dim);
    auto* out = reinterpret_cast< std::uint8_t* >(writer->mutable_v()->begin());
    auto slice_dimensions = cube_dimensions;
    slice_dimensions[dim.v] = 1;
    const auto corner = sc::frag_point< 3 >{ 0, 0, 0 };

    for (std::size_t i = 0; i < fragment_ids.size(); ++i) {
        const auto& id = fragment_ids[i];
        const auto& frag = jobs.at(i).get();

        const auto start = lineno * src_stride.start;
        auto pos = start;
        /*
         * Currently broken, as it does not account for padding in the
         * fragments. However, since the positions read are always smaller than
         * the padded fragment, it should never be out-of-bounds, only wrong.
         */
        auto dst_id = id;
        dst_id[dim.v] = 0;

        auto cornerpoint = gvt.to_global(id, corner);
        cornerpoint[dim.v] = 0;
        const auto offset = slice_dimensions.to_offset(cornerpoint);
        std::cout << offset << " => " << cornerpoint << "\n";

        std::vector< std::uint8_t > tmp(src_stride.readsize);
        for (auto i = 0; i < src_stride.readcount; ++i) {
            std::copy_n(frag.begin() + pos, src_stride.readsize, tmp.begin());
            pos += src_stride.stride;
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

    Server service(max_connections, key, storage_account);

    grpc::ServerBuilder builder;
    builder.AddListeningPort(address, grpc::InsecureServerCredentials());
    builder.RegisterService(&service);
    std::unique_ptr<grpc::Server> server(builder.BuildAndStart());
    server->Wait();
}
