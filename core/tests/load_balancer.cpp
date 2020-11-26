#include <catch/catch.hpp>
#include <thread>
#include <zmq.hpp>
#include <zmq_addon.hpp>

#include <oneseismic/load_balancer.hpp>

namespace {

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
    zmq::socket_t worker1(ctx, ZMQ_REQ);
    zmq::socket_t worker2(ctx, ZMQ_REQ);
    zmq::socket_t controller(ctx, ZMQ_PUSH);

    zmq::socket_t frontend(ctx, ZMQ_PULL);
    zmq::socket_t backend(ctx, ZMQ_REP);
    zmq::socket_t control(ctx, ZMQ_PULL);

    frontend.bind("inproc://frontend");
    backend.bind("inproc://backend");
    control.bind("inproc://control");

    std::thread t([&]() {
        one::load_balancer(frontend, backend, control, -1);
    });

    worker1.connect("inproc://backend");
    worker2.connect("inproc://backend");
    client.connect("inproc://frontend");
    controller.connect("inproc://control");

    SECTION("Workers can fetch tasks that are queued before task-fetch") {
        task().send(client);
        task().send(client);

        worker1.send(zmq::message_t(), zmq::send_flags::none);
        auto result = zmq::multipart_t(worker1);
        CHECK(result[0].to_string() == "part1");
        CHECK(result[1].to_string() == "part2");

        worker2.send(zmq::message_t(), zmq::send_flags::none);
        result = zmq::multipart_t(worker2);
        CHECK(result[0].to_string() == "part1");
        CHECK(result[1].to_string() == "part2");

        controller.send(zmq::message_t(""), zmq::send_flags::none);
        t.join();
    }

    SECTION("Jobs arriving on queue are passed to waiting workers") {
        worker1.send(zmq::message_t(), zmq::send_flags::none);
        worker2.send(zmq::message_t(), zmq::send_flags::none);

        task().send(client);
        auto result = zmq::multipart_t(worker1);
        CHECK(result[0].to_string() == "part1");
        CHECK(result[1].to_string() == "part2");

        task().send(client);
        result = zmq::multipart_t(worker2);
        CHECK(result[0].to_string() == "part1");
        CHECK(result[1].to_string() == "part2");

        controller.send(zmq::message_t(""), zmq::send_flags::none);
        t.join();
    }
}