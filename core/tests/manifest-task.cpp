#include <ciso646>
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
        zmq::socket_t fail(ctx, ZMQ_PUSH);
        sock.connect("inproc://queue");
        one::manifest_task mt;
        mt.run(xfer, sock, sock, fail);
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

TEST_CASE("Manifest-not-found puts message on failure queue") {
    zmq::context_t ctx;

    zmq::socket_t caller_req( ctx, ZMQ_PUSH);
    zmq::socket_t caller_rep( ctx, ZMQ_PULL);
    zmq::socket_t caller_fail(ctx, ZMQ_PULL);

    zmq::socket_t worker_req( ctx, ZMQ_PULL);
    zmq::socket_t worker_rep( ctx, ZMQ_PUSH);
    zmq::socket_t worker_fail(ctx, ZMQ_PUSH);

    caller_req.bind( "inproc://req");
    caller_rep.bind( "inproc://rep");
    caller_fail.bind("inproc://fail");
    worker_req.connect( "inproc://req");
    worker_rep.connect( "inproc://rep");
    worker_fail.connect("inproc://fail");

    mhttpd httpd(simple_manifest);
    struct storage_sans_manifest : public loopback_cfg {
        using loopback_cfg::loopback_cfg;

        action onstatus(
                const one::buffer&,
                const one::batch&,
                const std::string&,
                long) override {
            throw one::notfound("no reason");
        }
    } storage_cfg(httpd.port());

    const auto apireq = make_slice_request();
    zmq::message_t apimsg(apireq.data(), apireq.size());
    caller_req.send(apimsg, zmq::send_flags::none);

    one::transfer xfer(1, storage_cfg);
    one::manifest_task mt;
    mt.run(xfer, worker_req, worker_rep, worker_fail);

    zmq::message_t fail;
    const auto received = caller_fail.recv(fail, zmq::recv_flags::dontwait);
    CHECK(received);

    zmq::message_t result;
    const auto res_recv = caller_rep.recv(result, zmq::recv_flags::dontwait);
    CHECK(not res_recv);
}
