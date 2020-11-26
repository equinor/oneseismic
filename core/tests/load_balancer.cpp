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

    one::detail::load_balancer::configure_sockets(queue_backend, queue_backend);
    queue_frontend.bind("inproc://queue_frontend");
    queue_backend.bind("inproc://queue_backend");
    worker1.connect("inproc://queue_backend");
    worker2.connect("inproc://queue_backend");
    client.connect("inproc://queue_frontend");

    std::vector< worker > available_workers;
    zmq::multipart_t t;

    auto load_balancer_run = [&] (
            int hearthbeat_interval,
            int heartbeat_liveness,
            std::time_t time
    ) {
        one::detail::load_balancer::run(
                queue_frontend,
                queue_backend,
                t,
                available_workers,
                hearthbeat_interval,
                heartbeat_liveness,
                time
        );
    };

    SECTION("Job is passed to available worker") {
        int heartbeat_interval = 1000;
        int heartbeat_liveness = 3;
        std::time_t time = 1000;

        worker1.send(zmq::message_t("READY"), zmq::send_flags::none);
        load_balancer_run(heartbeat_interval, heartbeat_liveness, time);
        task().send(client);
        load_balancer_run(heartbeat_interval, heartbeat_liveness, time);

        auto result = zmq::multipart_t(worker1);
        CHECK(result[0].to_string() == "part1");
        CHECK(result[1].to_string() == "part2");
    }

    SECTION("Job is resent to another worker if it fails to send") {
        int heartbeat_interval = 1000;
        int heartbeat_liveness = 3;
        std::time_t time = 1000;

        worker1.send(zmq::message_t("READY"), zmq::send_flags::none);
        load_balancer_run(heartbeat_interval, heartbeat_liveness, time);
        worker1.disconnect("inproc://queue_backend");
        task().send(client);
        load_balancer_run(heartbeat_interval, heartbeat_liveness, time);
        worker2.send(zmq::message_t("READY"), zmq::send_flags::none);
        load_balancer_run(heartbeat_interval, heartbeat_liveness, time);
        load_balancer_run(heartbeat_interval, heartbeat_liveness, time);

        auto result = zmq::multipart_t(worker2);
        CHECK(result[0].to_string() == "part1");
        CHECK(result[1].to_string() == "part2");
    }

    SECTION("Worker is removed from pool on HEARTBEAT timeout") {
        int heartbeat_interval = 1000;
        int heartbeat_liveness = 3;
        std::time_t time = 1000;

        worker1.send(zmq::message_t("READY"), zmq::send_flags::none);
        load_balancer_run(heartbeat_interval, heartbeat_liveness, time);
        load_balancer_run(
            heartbeat_interval,
            heartbeat_liveness,
            time + heartbeat_liveness * heartbeat_interval + 1
        );

        CHECK(available_workers.empty());
    }

    SECTION("Worker is *not* removed from pool when HEARTBEAT is received "
            "within timeout interval") {
        int heartbeat_interval = 1000;
        int heartbeat_liveness = 3;
        std::time_t time = 1000;

        worker1.send(zmq::message_t("READY"), zmq::send_flags::none);
        worker2.send(zmq::message_t("READY"), zmq::send_flags::none);
        load_balancer_run(heartbeat_interval, heartbeat_liveness, time);
        load_balancer_run(heartbeat_interval, heartbeat_liveness, time);
        worker1.send(zmq::message_t("HEARTBEAT"), zmq::send_flags::none);
        load_balancer_run(
            heartbeat_interval,
            heartbeat_liveness,
            time + heartbeat_interval
        );
        load_balancer_run(
            heartbeat_interval,
            heartbeat_liveness,
            time + heartbeat_liveness * heartbeat_interval + 1
        );

        CHECK(available_workers.size() == 1);
    }

}