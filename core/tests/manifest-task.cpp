#include <ciso646>
#include <string>

#include <catch/catch.hpp>
#include <microhttpd.h>
#include <zmq.hpp>
#include <zmq_addon.hpp>

#include <oneseismic/transfer.hpp>
#include <oneseismic/tasks.hpp>

#include "mhttpd.hpp"
#include "utility.hpp"
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

    auto* fragment_shape = req.mutable_shape();
    fragment_shape->set_dim0(2);
    fragment_shape->set_dim1(2);
    fragment_shape->set_dim2(2);

    auto* slice = req.mutable_slice();
    slice->set_dim(1);
    slice->set_lineno(4);

    std::string msg;
    req.SerializeToString(&msg);
    return msg;
}

TEST_CASE("Manifest messages are pushed to the right queue") {
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
    const auto req = make_slice_request();
    zmq::message_t reqmsg(req.data(), req.size());

    SECTION("Successful calls are pushed to destination") {
        loopback_cfg storage(httpd.port());
        one::transfer xfer(1, storage);
        one::manifest_task mt;

        // Attach message envelope
        caller_req.send(zmq::str_buffer("ENVELOPE"), zmq::send_flags::sndmore);

        caller_req.send(reqmsg, zmq::send_flags::none);
        mt.run(xfer, worker_req, worker_rep, worker_fail);

        // Envelope should be passed throug without
        // interfering with the manifest task
        zmq::message_t envelope;
        caller_rep.recv(envelope, zmq::recv_flags::none);
        CHECK(envelope.to_string() == "ENVELOPE");

        zmq::message_t repmsg;
        const auto rep_recv = caller_rep.recv(
                repmsg,
                zmq::recv_flags::dontwait
        );
        REQUIRE(rep_recv);

        oneseismic::fetch_request rep;
        const auto ok = rep.ParseFromArray(repmsg.data(), repmsg.size());
        REQUIRE(ok);

        CHECK(rep.root() == "root");
        CHECK(rep.fragment_shape().dim0() == 2);
        CHECK(rep.fragment_shape().dim1() == 2);
        CHECK(rep.fragment_shape().dim2() == 2);

        REQUIRE(rep.has_slice());
        CHECK(rep.slice().dim() == 1);
        CHECK(rep.slice().idx() == 1);
    }

    SECTION("not-found messages are pushed on failure") {

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

        zmq::multipart_t request;
        request.addstr("job-id");
        request.add(std::move(reqmsg));
        request.send(caller_req);

        one::transfer xfer(1, storage_cfg);
        one::manifest_task mt;
        mt.run(xfer, worker_req, worker_rep, worker_fail);

        zmq::multipart_t fail;
        const auto received = fail.recv(
                caller_fail,
                static_cast< int >(zmq::recv_flags::dontwait)
        );
        CHECK(received);
        CHECK(fail.front().to_string() == "job-id");
        CHECK(fail.back().to_string() == "manifest-not-found");

        CHECK(not received_message(caller_rep));
    }
}
