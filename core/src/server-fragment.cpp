#include <cstdlib>
#include <iostream>
#include <string>

#include <clara/clara.hpp>
#include <fmt/format.h>
#include <zmq.hpp>

#include <oneseismic/azure.hpp>
#include <oneseismic/transfer.hpp>
#include <oneseismic/tasks.hpp>

namespace one {

class az_manifest : public az {
public:
    using az::az;

    std::string url(
            const one::batch& batch,
            const std::string&) const override {

        return fmt::format(
            "https://{}.blob.core.windows.net/{}/manifest.json",
            batch.root,
            batch.guid
        );

    }

    std::string canonicalized_resource(
            const std::string& root,
            const std::string& guid,
            const std::string& fragment_shape,
            const std::string& fragment_id)
    const noexcept (false) override {
        return fmt::format(
                "/{}/{}/manifest.json",
                root,
                guid
        );
    }

    action onstatus(
            const buffer& b,
            const batch&,
            const std::string& fragment_id,
            long status_code) override {

        // https://docs.microsoft.com/en-us/rest/api/storageservices/blob-service-error-codes

        if (status_code == 200) {
            this->ok = true;
            return action::done;
        }

        if (status_code == 404) {
            this->ok = false;
            this->response.assign(b.begin(), b.end());
            return action::done;
        }

        if (status_code == 500) {
            this->ok = false;
            this->response.assign(b.begin(), b.end());
            return action::done;
        }

        throw aborted(fmt::format("az-manifest: unhandled status code {}", status_code));
    }

    bool ok;
    std::string response;
};

}

int main(int argc, char** argv) {
    std::string source_address;
    std::string sink_address;
    std::string control_address;
    std::string fail_address;
    std::string acc;
    std::string key;
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
        | clara::Opt(acc, "storage account")
            ["-a"]["--account"]
            ("Storage account")
        | clara::Opt(key, "key")
            ["-k"]["--key"]
            ("Pre-shared key")
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

    if (acc.empty()) {
        std::cerr << "Need storage account\n" << cli << "\n";
        std::exit(EXIT_FAILURE);
    }

    if (key.empty()) {
        std::cerr << "Need pre-shared key\n" << cli << "\n";
        std::exit(EXIT_FAILURE);
    }

    zmq::context_t ctx;
    zmq::socket_t source(ctx, ZMQ_PULL);
    zmq::socket_t sink(ctx, ZMQ_PUSH);
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

    one::az_manifest az(acc, key);
    one::transfer xfer(ntransfers, az);
    one::fragment_task task;

    zmq::pollitem_t items[] = {
        { static_cast< void* >(source),  0, ZMQ_POLLIN, 0 },
        { static_cast< void* >(control), 0, ZMQ_POLLIN, 0 },
    };

    while (true) {
        zmq::poll(items, 2, -1);

        if (items[0].revents & ZMQ_POLLIN) {
            task.run(xfer, source, sink);
        }

        if (items[1].revents & ZMQ_POLLIN) {
            break;
        }
    }
}
