#include <ciso646>
#include <string>

#include <catch/catch.hpp>
#include <microhttpd.h>
#include <fmt/format.h>
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

void sendmsg(zmq::socket_t& sock, const std::string& body) {
    /*
     * One-off message with placeholer values for address and pid.
     *
     * Having this is a test helper achieves two things:
     * 1. Removes some always-repeated noise from the tests
     * 2. A single point of reference (and updating) for the protocol/format of
     *    messages sent to the manifest phase
     *
     * All in all it should make tests leaner, and easier to maintain.
     */
    zmq::multipart_t msg;
    msg.addstr("addr");
    msg.addstr("pid");
    msg.addstr(body);
    msg.send(sock);
}

}

std::string make_slice_request() {
    oneseismic::api_request req;
    req.set_root("root");
    req.set_guid("0d235a7138104e00c421e63f5e3261bf2dc3254b");
    req.set_storage_endpoint("storage");

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
    const auto reqmsg = make_slice_request();

    SECTION("Successful calls are pushed to destination") {
        loopback_cfg storage(httpd.port());
        one::transfer xfer(1, storage);
        one::manifest_task mt;

        sendmsg(caller_req, reqmsg);
        mt.run(xfer, worker_req, worker_rep, worker_fail);

        zmq::multipart_t response(caller_rep);
        REQUIRE(response.size() == 4);
        CHECK(response[0].to_string() == "addr");
        CHECK(response[1].to_string() == "pid");
        CHECK(response[2].to_string() == "0/1");
        const auto& msg = response[3];

        oneseismic::fetch_request rep;
        const auto ok = rep.ParseFromArray(msg.data(), msg.size());
        REQUIRE(ok);

        CHECK(rep.root() == "root");
        CHECK(rep.storage_endpoint() == "storage");
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

        sendmsg(caller_req, reqmsg);

        one::transfer xfer(1, storage_cfg);
        one::manifest_task mt;
        mt.run(xfer, worker_req, worker_rep, worker_fail);

        zmq::multipart_t fail;
        const auto received = fail.recv(
                caller_fail,
                static_cast< int >(zmq::recv_flags::dontwait)
        );
        CHECK(received);
        CHECK(fail.size() == 2);
        CHECK(fail[0].to_string() == "pid");
        CHECK(fail[1].to_string() == "manifest-not-found");

        CHECK(not received_message(caller_rep));
    }

    SECTION("not-authorized messages are pushed on failure") {

        struct storage_403 : public loopback_cfg {
            using loopback_cfg::loopback_cfg;

            action onstatus(
                    const one::buffer&,
                    const one::batch&,
                    const std::string&,
                    long) override {
                throw one::unauthorized("no reason");
            }
        } storage_cfg(httpd.port());

        sendmsg(caller_req, reqmsg);
        one::transfer xfer(1, storage_cfg);
        one::manifest_task mt;
        mt.run(xfer, worker_req, worker_rep, worker_fail);

        zmq::multipart_t fail;
        const auto received = fail.recv(
                caller_fail,
                static_cast< int >(zmq::recv_flags::dontwait)
        );
        CHECK(received);
        CHECK(fail.size() == 2);
        CHECK(fail[0].to_string() == "pid");
        CHECK(fail[1].to_string() == "manifest-not-authorized");

        CHECK(not received_message(caller_rep));
    }

    SECTION("Setting task size changes # of IDs") {
        loopback_cfg storage(httpd.port());
        one::transfer xfer(1, storage);

        one::manifest_task mt;
        int size, tasks;
        /*
         * The total result is 10 fragments. This table encodes the task size
         * and the expected jobs to get from the size. The main goal is to make
         * the test not hang in case of bugs
         */
        std::tie(size, tasks) = GENERATE(table< int, int >({
                {1, 10},
                {2, 5},
                {3, 4},
        }));
        mt.max_task_size(size);

        sendmsg(caller_req, reqmsg);
        mt.run(xfer, worker_req, worker_rep, worker_fail);

        const auto expected = std::vector< std::string > {
            "0-0-0",
            "0-0-1",
            "0-0-2",
            "0-0-3",
            "0-0-4",
            "1-0-0",
            "1-0-1",
            "1-0-2",
            "1-0-3",
            "1-0-4",
        };

        auto received = std::vector< std::string >();
        for (int i = 0; i < tasks; ++i) {
            if (not received_message(caller_rep)) {
                FAIL("Not enough messages received, did not get #" << i + 1);
            }

            zmq::multipart_t response(caller_rep);
            const auto& msg = response[3];

            oneseismic::fetch_request rep;
            const auto ok = rep.ParseFromString(msg.to_string());
            REQUIRE(ok);

            for (const auto& id : rep.ids()) {
                received.push_back(fmt::format(
                    "{}-{}-{}",
                    id.dim0(),
                    id.dim1(),
                    id.dim2()
                ));
            }
        }

        std::sort(received.begin(), received.end());
        CHECK_THAT(received, Equals(expected));
    }
}
