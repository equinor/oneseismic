#include <string>
#include <thread>

#include <catch/catch.hpp>
#include <microhttpd.h>
#include <zmq.hpp>

#include <oneseismic/transfer.hpp>
#include <oneseismic/tasks.hpp>

#include "mhttpd.hpp"
#include "core.pb.h"

using namespace Catch::Matchers;

namespace {

int simple_manifest(
        void*,
        struct MHD_Connection* connection,
        const char*,
        const char*,
        const char*,
        const char*,
        size_t*,
        void**) {

    std::string manifest = R"(
    {
        "guid": "0d235a7138104e00c421e63f5e3261bf2dc3254b",
        "dimensions": [
            [1, 2, 3],
            [2, 4, 6],
            [0, 4, 8, 12, 16, 20, 24, 28, 32, 36]
        ]
    }
    )";

    auto* response = MHD_create_response_from_buffer(
            manifest.size(),
            (void*)manifest.data(),
            MHD_RESPMEM_MUST_COPY
    );

    auto ret = MHD_queue_response(connection, MHD_HTTP_OK, response);
    MHD_destroy_response(response);
    return ret;
}

}

std::string make_slice_request() {
    oneseismic::api_request req;
    req.set_root("root");
    req.set_guid("0d235a7138104e00c421e63f5e3261bf2dc3254b");

    auto* shape = req.mutable_shape();
    shape->set_dim0(2);
    shape->set_dim1(2);
    shape->set_dim2(2);

    auto* slice = req.mutable_slice();
    slice->set_dim(1);
    slice->set_lineno(4);

    std::string msg;
    req.SerializeToString(&msg);
    return msg;
}

TEST_CASE("Manifest-message is pushed through") {
    zmq::context_t ctx;
    zmq::socket_t sock(ctx, ZMQ_REQ);
    sock.bind("inproc://queue");

    std::thread task([&ctx] {
        mhttpd httpd(simple_manifest);
        loopback_cfg storage(httpd.port());
        one::transfer xfer(1, storage);

        zmq::socket_t sock(ctx, ZMQ_REP);
        sock.connect("inproc://queue");
        one::manifest_task mt;
        mt.run(xfer, sock, sock);
    });

    const auto apireq = make_slice_request();
    zmq::message_t apimsg(apireq.data(), apireq.size());
    sock.send(apimsg, zmq::send_flags::none);

    zmq::message_t msg;
    sock.recv(msg, zmq::recv_flags::none);
    oneseismic::fetch_request req;
    task.join();
    const auto ok = req.ParseFromArray(msg.data(), msg.size());
    REQUIRE(ok);

    CHECK(req.root() == "root");
    CHECK(req.shape().dim0() == 2);
    CHECK(req.shape().dim1() == 2);
    CHECK(req.shape().dim2() == 2);

    REQUIRE(req.has_slice());
    CHECK(req.slice().dim() == 1);
    CHECK(req.slice().idx() == 1);
}
