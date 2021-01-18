#include <algorithm>
#include <ctime>
#include <vector>
#include <spdlog/spdlog.h>
#include <zmq.hpp>
#include <zmq_addon.hpp>

#include <oneseismic/load_balancer.hpp>

namespace {

using worker = one::detail::load_balancer::worker;

void update_worker(
    std::vector< worker >& workers,
    const std::string& identity,
    const std::time_t expiry
) noexcept (false) {
    auto w = std::find_if(
        std::begin(workers),
        std::end(workers),
        [&identity] (auto item) { return item.identity == identity; }
    );

    if (w == workers.end())
        workers.push_back(worker{identity, expiry});
    else
        w->expiry = expiry;
}

void purge_workers(
    std::vector< worker >& workers,
    std::time_t current_time
) noexcept (false) {
    auto expired = [current_time](auto& worker) {
        return worker.expiry < current_time;
    };

    workers.erase(
        std::remove_if(workers.begin(), workers.end(), expired),
        workers.end()
    );
}

}

namespace one {

namespace detail { namespace load_balancer {

void run(
    zmq::socket_t& frontend,
    zmq::socket_t& backend,
    std::vector< worker >& available_workers,
    int ready_ttl,
    std::time_t current_time
) noexcept (false) {

    zmq::pollitem_t items[] = {
            {static_cast< void* >(backend),  0, ZMQ_POLLIN, 0},
            {static_cast< void* >(frontend), 0, ZMQ_POLLIN, 0}
    };

    /*
     * Poll frontend only if there are available workers
     */
    if (!available_workers.empty())
        zmq::poll(items, 2, ready_ttl);
    else
        zmq::poll(items, 1, ready_ttl);

    // Handle worker control messages from backend
    if (items[0].revents & ZMQ_POLLIN) {
        auto msg = zmq::multipart_t(backend);
        auto expiry = current_time + ready_ttl;

        auto identity = msg[0].to_string();
        auto message  = msg[1].to_string();

        if (message == "READY")
            update_worker(available_workers, identity, expiry);
        else
            spdlog::error("Invalid message from worker: {}", message);
    }

    /*
     * Push message from frontend to available worker.
     */
    if (items[1].revents & ZMQ_POLLIN) {
        auto task = zmq::multipart_t(frontend);
        auto identity = available_workers[0].identity;
        available_workers.erase(available_workers.begin());
        task.pushstr(identity);
        task.send(backend);
    }

    purge_workers(available_workers, current_time);
}

}}

[[noreturn]] void load_balancer(
    zmq::socket_t& frontend,
    zmq::socket_t& backend,
    int ready_ttl
) {
    /*
     * Message broker that will distribute tasks to workers as they become
     * available.
     *
     * This solution is based on the Robust Reliable Queuing (Paranoid Pirate
     * Pattern) [1], with some key differences:
     *
     *     1. Replies from workers are not routed back to the client. Since the
     *        workers don't send a reply when the job is done, a READY message
     *        is sent instead, indicating that the worker is available to
     *        receive new tasks.
     *     2. Tasks arrive on a PULL socket.
     *
     * The advantage of this approach over PUSH-PULL is that jobs are sent to
     * idle workers, whereas PUSH-PULL distributes round robin. A disadvantage
     * is that the queue becomes a single point of failure, and a single point
     * for all messages to pass through, while PUSH-PULL is N-to-N.
     *
     * [1] https://zguide.zeromq.org/docs/chapter4/
     */

    std::vector< worker > available_workers;

    while (true) {
        detail::load_balancer::run(
            frontend,
            backend,
            available_workers,
            ready_ttl,
            std::time(nullptr)
        );
    }
}

}