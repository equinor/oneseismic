#include <ciso646>
#include <string>

#include <catch/catch.hpp>
#include <fmt/format.h>
#include <nlohmann/json.hpp>
#include <zmq_addon.hpp>
#include <zmq.hpp>

#include <oneseismic/messages.hpp>
#include <oneseismic/tasks.hpp>

#include "config.hpp"
#include "utility.hpp"

using namespace Catch::Matchers;

namespace {

void sendmsg(zmq::socket_t& sock, const std::string& body, const std::string& pid) {
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
    msg.addstr(pid);
    msg.addstr(body);
    msg.send(sock);
}

std::string make_slice_request(const std::string& manifest) {
    one::slice_task task;
    task.guid = "0d235a7138104e00c421e63f5e3261bf2dc3254b";
    task.manifest = manifest;
    task.storage_endpoint = "storage";
    task.shape  = { 2, 2, 2 };
    task.dim    = 1;
    task.lineno = 4;
    return task.pack();
}

std::string make_slice_request() {
    const std::string manifest = R"(
    {
        "guid": "0d235a7138104e00c421e63f5e3261bf2dc3254b",
        "dimensions": [
            [1, 2, 3],
            [2, 4, 6],
            [0, 4, 8, 12, 16, 20, 24, 28, 32, 36]
        ]
    }
    )";

    return make_slice_request(manifest);
}

}

TEST_CASE(
        "Manifest messages are pushed to the right queue",
        "[.][integration]") {
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

    const auto reqmsg = make_slice_request();

    SECTION("Successful calls are pushed to destination") {
        one::manifest_task mt;
        mt.connect_working_storage(redisaddr());

        const auto pid = makepid();
        sendmsg(caller_req, reqmsg, pid);
        mt.run(worker_req, worker_rep, worker_fail);

        zmq::multipart_t response(caller_rep);
        REQUIRE(response.size() == 3);
        CHECK(response[0].to_string() == pid);
        CHECK(response[1].to_string() == "0/1");
        const auto& msg = response[2];

        one::slice_fetch task;
        task.unpack(
                static_cast< const char* >(msg.data()),
                static_cast< const char* >(msg.data()) + msg.size()
        );

        CHECK(task.storage_endpoint == "storage");
        CHECK(task.dim    == 1);
        CHECK(task.lineno == 1);
        CHECK_THAT(task.shape, Equals(std::vector< int >{ 2, 2, 2 }));
    }

    SECTION("badly-formatted manifest pushes failure") {
        const auto pid = makepid();
        /*
         *  TODO
         *  There are all kinds of bad manifests that should be covered. This
         *  document misses the closing }, as parsing now only catches json
         *  parse errors, not missing fields or bad values.
         */
        const auto bad_manifest = R"(
        {
            "guid": "0d235a7138104e00c421e63f5e3261bf2dc3254b",
            "dimensions": [
                [1, 2, 3],
                [2, 4, 6],
                [0, 4, 8, 12, 16, 20, 24, 28, 32, 36]
            ]
        )";
        auto reqmsg = make_slice_request(bad_manifest);
        sendmsg(caller_req, reqmsg, pid);

        one::manifest_task mt;
        mt.run(worker_req, worker_rep, worker_fail);

        zmq::multipart_t fail;
        const auto received = fail.recv(
                caller_fail,
                static_cast< int >(zmq::recv_flags::dontwait)
        );
        CHECK(received);
        CHECK(fail.size() == 2);
        CHECK(fail[0].to_string() == pid);
        CHECK(fail[1].to_string() == "json-parse-error");
        CHECK(not received_message(caller_rep));
    }

    SECTION("Setting task size changes # of IDs") {
        one::manifest_task mt;
        mt.connect_working_storage(redisaddr());
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

        sendmsg(caller_req, reqmsg, makepid());
        mt.run(worker_req, worker_rep, worker_fail);

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
            const auto& msg = response[2];

            one::slice_fetch task;
            task.unpack(
                static_cast< const char* >(msg.data()),
                static_cast< const char* >(msg.data()) + msg.size()
            );

            for (const auto& id : task.ids) {
                received.push_back(fmt::format("{}", fmt::join(id, "-")));
            }
        }

        std::sort(received.begin(), received.end());
        CHECK_THAT(received, Equals(expected));
    }
}

TEST_CASE(
        "No tasks are queued when header put fails",
        "[.][integration][!shouldfail]") {
    /*
     * This test is marked as shouldfail, as it is not possible to test
     * currently. This tests if redis SET fails, which is not possible [1]
     * to emulate right now. If this is fixed then the test should be
     * updated to change tag.
     *
     * When this test passes again, it can become a section in the "Manifest
     * messages are pushed to the right queue" test case again.
     *
     * [1] maybe possible, but certainly not feasible, to emulate
     */
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

    const auto reqmsg = make_slice_request();

    SECTION("No tasks are queued when header put fails") {
        const auto pid = makepid();
        sendmsg(caller_req, reqmsg, pid);
        one::manifest_task mt;
        mt.connect_working_storage(redisaddr());
        mt.run(worker_req, worker_rep, worker_fail);

        zmq::multipart_t fail;
        const auto received = fail.recv(
                caller_fail,
                static_cast< int >(zmq::recv_flags::dontwait)
        );
        REQUIRE(received);
        REQUIRE(fail.size() == 2);
        CHECK(fail[0].to_string() == pid);
        CHECK(fail[1].to_string() == "header-put-not-authorized");
        CHECK(not received_message(caller_rep));
    }
}
