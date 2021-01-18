#include <cstdlib>
#include <iostream>
#include <string>
#include <thread>

#include <clara/clara.hpp>
#include <fmt/format.h>
#include <zmq.hpp>

#include <oneseismic/transfer.hpp>
#include <oneseismic/tasks.hpp>
#include <oneseismic/load_balancer.hpp>

namespace {

void start_load_balancer(
        zmq::context_t &ctx,
        zmq::socket_t &queue_front,
        zmq::socket_t &queue_back,
        int ready_ttl
) {
    std::string queue("inproc://queue");
    queue_front.bind(queue);

    std::thread([&ctx, &queue_back, ready_ttl, queue]() {
        zmq::socket_t queue_source(ctx, ZMQ_PULL);
        queue_source.connect(queue);
        one::load_balancer(queue_source, queue_back, ready_ttl);
    }).detach();
}

}

int main(int argc, char** argv) {
    std::string source_address;
    std::string sink_address = "tcp://*:68142";
    std::string control_address;
    std::string fail_address;
    std::string redis_address;
    bool help = false;
    int ntransfers = 4;
    int task_size = 10;
    int ready_ttl = 3000;

    auto cli
        = clara::Help(help)
        | clara::Opt(sink_address, "sink")
            ["--sink"]
            (fmt::format("Sink address, default = {}", sink_address))
        | clara::Opt(source_address, "source")
            ["--source"]
            (fmt::format("source address"))
        | clara::Opt(control_address, "control")
            ["--control"]
            (fmt::format("control address, currently unused"))
        | clara::Opt(fail_address, "fail")
            ["--fail"]
            (fmt::format("failure address"))
        | clara::Opt(redis_address, "redis")
            ["--redis"]
            (fmt::format("working storage (redis) address"))
        | clara::Opt(ntransfers, "transfers")
            ["-j"]["--transfers"]
            (fmt::format("Concurrent blob connections, default = {}", ntransfers))
        | clara::Opt(task_size, "task size")
            ["-t"]["--task-size"]
            (fmt::format("Max task size (# of fragments), default = {}", task_size))
        | clara::Opt(ready_ttl, "ready_ttl")
            ["--ready-ttl"]
            (fmt::format("Sets the timeout for worker READY messages"))
    ;

    auto result = cli.parse(clara::Args(argc, argv));

    if (!result) {
        fmt::print(stderr, "{}\n", result.errorMessage());
        std::exit(EXIT_FAILURE);
    }

    if (help) {
        std::cout << cli << "\n";
        std::exit(EXIT_SUCCESS);
    }

    zmq::context_t ctx;
    zmq::socket_t source(ctx, ZMQ_PULL);
    zmq::socket_t queue_front(ctx, ZMQ_PUSH);
    zmq::socket_t queue_back(ctx, ZMQ_ROUTER);
    zmq::socket_t control(ctx, ZMQ_SUB);
    zmq::socket_t fail(ctx, ZMQ_PUSH);
    control.setsockopt(ZMQ_SUBSCRIBE, "ctrl:kill", 0);

    try {
        source.connect(source_address);
    } catch (...) {
        std::cerr << "Invalid source address\n";
        std::exit(EXIT_FAILURE);
    }
    try {
        queue_back.bind(sink_address);
    } catch (...) {
        std::cerr << "Invalid sink address\n";
        std::exit(EXIT_FAILURE);
    }
    try {
        fail.connect(fail_address);
    } catch (...) {
        std::cerr << "Invalid failure address\n";
        std::exit(EXIT_FAILURE);
    }

    start_load_balancer(ctx, queue_front, queue_back, ready_ttl);

    one::manifest_task task;
    task.connect_working_storage(redis_address);
    try {
        task.max_task_size(task_size);
    } catch (const std::exception& e) {
        std::cerr << e.what() << "\n";
        std::exit(EXIT_FAILURE);
    }

    zmq::pollitem_t items[] = {
        { static_cast< void* >(source),  0, ZMQ_POLLIN, 0 },
        { static_cast< void* >(control), 0, ZMQ_POLLIN, 0 },
    };

    while (true) {
        zmq::poll(items, 2, -1);

        if (items[0].revents & ZMQ_POLLIN) {
            task.run(source, queue_front, fail);
        }

        if (items[1].revents & ZMQ_POLLIN) {
            break;
        }
    }
}
