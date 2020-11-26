#include <catch/catch.hpp>
#include <ctime>
#include <zmq.hpp>
#include <zmq_addon.hpp>

#include <oneseismic/load_balancer.hpp>

static constexpr int HEARTBEAT_INTERVAL = 10;

namespace {

using worker = one::detail::load_balancer::worker;

zmq::multipart_t task() {
    zmq::multipart_t task;
    task.addstr("part1");
    task.addstr("part2");
    return task;
}

}

TEST_CASE("Messages are pushed through to available workers") {
    zmq::context_t ctx;
    zmq::socket_t client(ctx, ZMQ_PUSH);
    zmq::socket_t worker1(ctx, ZMQ_DEALER);
    zmq::socket_t worker2(ctx, ZMQ_DEALER);
    zmq::socket_t queue_frontend(ctx, ZMQ_PULL);
    zmq::socket_t queue_backend(ctx, ZMQ_ROUTER);

    queue_frontend.bind("inproc://queue_frontend");
    queue_backend.bind("inproc://queue_backend");
    worker1.connect("inproc://queue_backend");
    worker2.connect("inproc://queue_backend");
    client.connect("inproc://queue_frontend");

    std::vector< worker > available_workers;

    auto load_balancer_run = [&] (
            int ready_ttl,
            std::time_t time
    ) {
        one::detail::load_balancer::run(
                queue_frontend,
                queue_backend,
                available_workers,
                ready_ttl,
                time
        );
    };

    auto ready = []() {
        return zmq::message_t(std::string("READY"));
    };

    SECTION("Job is passed to available worker") {
        int ready_ttl = 1000;
        std::time_t time = 1000;

        worker1.send(ready(), zmq::send_flags::none);
        load_balancer_run(ready_ttl, time);
        task().send(client);
        load_balancer_run(ready_ttl, time);

        auto result = zmq::multipart_t(worker1);
        CHECK(result[0].to_string() == "part1");
        CHECK(result[1].to_string() == "part2");
    }

    SECTION("Worker is removed from pool if it has not sent a new READY "
            "message within ready_ttl") {
        int ready_ttl = 1000;
        std::time_t time = 1000;

        worker1.send(ready(), zmq::send_flags::none);
        load_balancer_run(ready_ttl, time);
        load_balancer_run(
            ready_ttl,
            time + ready_ttl + 1
        );

        CHECK(available_workers.empty());
    }

    SECTION("Worker is *not* removed from pool when new READY is received "
            "within ready_ttl interval") {
        int ready_ttl = 1000;
        std::time_t time = 1000;

        worker1.send(ready(), zmq::send_flags::none);
        worker2.send(ready(), zmq::send_flags::none);
        load_balancer_run(ready_ttl, time);
        load_balancer_run(ready_ttl, time);
        worker1.send(ready(), zmq::send_flags::none);
        load_balancer_run(
            ready_ttl,
            time + ready_ttl
        );
        load_balancer_run(
            ready_ttl,
            time + ready_ttl + 1
        );

        CHECK(available_workers.size() == 1);
    }

}