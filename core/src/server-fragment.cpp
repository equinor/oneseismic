#include <cstdlib>
#include <iostream>
#include <string>

#include <clara/clara.hpp>
#include <fmt/format.h>
#include <zmq.hpp>

#include <oneseismic/azure.hpp>
#include <oneseismic/transfer.hpp>
#include <oneseismic/tasks.hpp>

int main(int argc, char** argv) {
    std::string source_address;
    std::string sink_address;
    std::string control_address;
    std::string fail_address;
    bool help = false;
    int ntransfers = 4;

    auto cli
        = clara::Help(help)
        | clara::Opt(sink_address, "sink")
            ["--sink"]
            ("sink (session manager) address")
        | clara::Opt(source_address, "source")
            ["--source"]
            ("source (manifest) address")
        | clara::Opt(control_address, "control")
            ["--control"]
            (fmt::format("control address, currently unused"))
        | clara::Opt(fail_address, "fail")
            ["--fail"]
            (fmt::format("failure address, currently unused"))
        | clara::Opt(ntransfers, "transfers")
            ["-j"]["--transfers"]
            (fmt::format("Concurrent blob connections, default = {}", ntransfers))
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
    zmq::socket_t sink(ctx, ZMQ_ROUTER);
    sink.setsockopt(ZMQ_ROUTER_MANDATORY, 1);
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
        sink.connect(sink_address);
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

    one::az az("");
    one::transfer xfer(ntransfers, az);

    zmq::pollitem_t items[] = {
        { static_cast< void* >(source),  0, ZMQ_POLLIN, 0 },
        { static_cast< void* >(control), 0, ZMQ_POLLIN, 0 },
    };

    while (true) {
        zmq::poll(items, 2, -1);

        if (items[0].revents & ZMQ_POLLIN) {
            one::fragment_task task;
            task.run(xfer, source, sink, fail);
        }

        if (items[1].revents & ZMQ_POLLIN) {
            break;
        }
    }
}
