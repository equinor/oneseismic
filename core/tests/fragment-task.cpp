#include <string>

#include <catch/catch.hpp>
#include <fmt/format.h>
#include <microhttpd.h>
#include <zmq.hpp>
#include <zmq_addon.hpp>

#include <oneseismic/messages.hpp>
#include <oneseismic/tasks.hpp>
#include <oneseismic/transfer.hpp>

#include "mhttpd.hpp"
#include "utility.hpp"

using namespace Catch::Matchers;

namespace {

/*
 * A 2x2x2 fragment where each byte is encoded by its index
 */
const auto index_2x2x2 = std::vector< std::uint8_t > {
    0x00, 0x01, 0x02, 0x03,
    0x04, 0x05, 0x06, 0x07,
    0x08, 0x09, 0xA0, 0xA1,
    0xA2, 0xA3, 0xA4, 0xA5,
    0xA6, 0xA7, 0xA8, 0xA9,
    0xB0, 0xB1, 0xB2, 0xB3,
    0xB4, 0xB5, 0xB6, 0xB7,
};


int fragment_response(
        void*,
        struct MHD_Connection* connection,
        const char*,
        const char*,
        const char*,
        const char*,
        size_t*,
        void**) {


    auto* response = MHD_create_response_from_buffer(
            index_2x2x2.size(),
            (void*)index_2x2x2.data(),
            MHD_RESPMEM_MUST_COPY
    );

    auto ret = MHD_queue_response(connection, MHD_HTTP_OK, response);
    MHD_destroy_response(response);
    return ret;
}

void sendmsg(zmq::socket_t& sock, const std::string& body) {
    /*
     * One-off message with placeholer value for pid.
     *
     * Having this is a test helper achieves two things:
     * 1. Removes some always-repeated noise from the tests
     * 2. A single point of reference (and updating) for the protocol/format of
     * messages sent to the framgents phase
     *
     * Please note that this is only intended for single-part tasks, with the
     * hard-coding of 0/1. Multi-part jobs should probably be explicitly tested
     * as such, or use a different sendmulti() function.
     */
    zmq::multipart_t msg;
    msg.addstr("pid");
    msg.addstr("0/1");
    msg.addstr(body);
    msg.send(sock);
}

}

std::string make_slice_request(int dim, int idx, int k = 0) {
    one::slice_fetch req;
    req.guid = "0d235a7138104e00c421e63f5e3261bf2dc3254b";
    req.shape = { 2, 2, 2 };
    req.cube_shape = { 8, 8, 8 };
    req.dim = dim;
    req.lineno = idx;
    req.ids = {
        { 0, 0, k },
    };
    return req.pack();
}

TEST_CASE(
        "Fragment is sliced and pushed through to the right queue",
        "[slice]")
{
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

    mhttpd httpd(fragment_response);
    const auto apireq = make_slice_request(0, 0);

    SECTION("Successful calls are pushed to destination") {
        loopback_cfg storage(httpd.port());
        one::transfer xfer(1, storage);

        sendmsg(caller_req, apireq);
        one::fragment_task ft;
        ft.run(xfer, worker_req, worker_rep, worker_fail);

        zmq::multipart_t response(caller_rep);
        CHECK(response[0].to_string() == "pid");
        CHECK(response[1].to_string() == "0/1");
        const auto& msg = response[3];

        one::slice_tiles tiles;
        tiles.unpack(
            static_cast< const char* >(msg.data()),
            static_cast< const char* >(msg.data()) + msg.size()
        );

        std::vector< float > expected(4);
        std::memcpy(expected.data(), index_2x2x2.data(), 4 * sizeof(float));

        REQUIRE(tiles.tiles.size() == 1);
        const auto& tile = tiles.tiles.front();
        CHECK(tile.iterations    == 2);
        CHECK(tile.chunk_size    == 2);
        CHECK(tile.initial_skip  == 0);
        CHECK(tile.superstride   == 8);
        CHECK(tile.substride     == 2);
        CHECK_THAT(tile.v, Equals(expected));
    }

    SECTION("Multi-part job triggers multiple fetches") {
        struct record_request : public loopback_cfg {
            using loopback_cfg::loopback_cfg;

            std::string url(
                    const one::batch& b,
                    const std::string& id) const override {
                this->requested.push_back(id);
                return this->loopback_cfg::url(b, id);
            }

            mutable std::vector< std::string > requested;
        } storage_cfg(httpd.port());

        const auto requests = std::vector< std::string > {
            "0-0-0",
            "0-0-1",
            "0-0-2",
        };

        for (std::size_t i = 0; i < requests.size(); ++i) {
            zmq::multipart_t request;
            request.addstr("pid");
            request.addstr(fmt::format("{}/{}", i, requests.size()));
            request.addstr(make_slice_request(0, 0, i));
            request.send(caller_req);
        }

        one::transfer xfer(1, storage_cfg);
        one::fragment_task ft;
        for (const auto& x : requests) {
            ft.run(xfer, worker_req, worker_rep, worker_fail);
        }

        std::sort(storage_cfg.requested.begin(), storage_cfg.requested.end());
        CHECK_THAT(storage_cfg.requested, Equals(requests));
    }

    SECTION("not-found messages are pushed on failure") {

        struct fragment_404 : public loopback_cfg {
            using loopback_cfg::loopback_cfg;

            action onstatus(
                    const one::buffer&,
                    const one::batch&,
                    const std::string&,
                    long) override {
                throw one::notfound("no reason");
            }
        } storage_cfg(httpd.port());

        sendmsg(caller_req, apireq);
        one::transfer xfer(1, storage_cfg);
        one::fragment_task ft;
        ft.run(xfer, worker_req, worker_rep, worker_fail);

        zmq::multipart_t fail;
        const auto received = fail.recv(
                caller_fail,
                static_cast< int >(zmq::recv_flags::dontwait)
        );
        CHECK(received);
        CHECK(fail.size() == 2);
        CHECK(fail[0].to_string() == "pid");
        CHECK(fail[1].to_string() == "fragment-not-found");

        CHECK(not received_message(caller_rep));
    }

    SECTION("not-authorized messages are pushed on failure") {
        struct fragment_403 : public loopback_cfg {
            using loopback_cfg::loopback_cfg;

            action onstatus(
                    const one::buffer&,
                    const one::batch&,
                    const std::string&,
                    long) override {
                throw one::unauthorized("no reason");
            }
        } storage_cfg(httpd.port());

        sendmsg(caller_req, apireq);
        one::transfer xfer(1, storage_cfg);
        one::fragment_task ft;
        ft.run(xfer, worker_req, worker_rep, worker_fail);

        zmq::multipart_t fail;
        const auto received = fail.recv(
                caller_fail,
                static_cast< int >(zmq::recv_flags::dontwait)
        );
        CHECK(received);
        CHECK(fail.size() == 2);
        CHECK(fail[0].to_string() == "pid");
        CHECK(fail[1].to_string() == "fragment-not-authorized");

        CHECK(not received_message(caller_rep));
    }

    SECTION("Multiple sequential requests are all successful") {
        loopback_cfg storage(httpd.port());
        one::transfer xfer(1, storage);
        one::fragment_task ft;

        for (int i = 0; i < 3; ++i) {
            sendmsg(caller_req, apireq);
            ft.run(xfer, worker_req, worker_rep, worker_fail);

            zmq::multipart_t response(caller_rep);
            REQUIRE(response.size() == 4);
            CHECK(response[0].to_string() == "pid");
            CHECK(response[1].to_string() == "0/1");
            const auto& msg = response[3];

            one::slice_tiles tiles;
            tiles.unpack(
                static_cast< const char* >(msg.data()),
                static_cast< const char* >(msg.data()) + msg.size()
            );

            std::vector< float > expected(4);
            std::memcpy(expected.data(), index_2x2x2.data(), 4 * sizeof(float));

            REQUIRE(tiles.tiles.size() == 1);
            const auto& tile = tiles.tiles.front();
            CHECK(tile.iterations    == 2);
            CHECK(tile.chunk_size    == 2);
            CHECK(tile.initial_skip  == 0);
            CHECK(tile.superstride   == 8);
            CHECK(tile.substride     == 2);
            CHECK_THAT(tile.v, Equals(expected));
        }
    }
}
