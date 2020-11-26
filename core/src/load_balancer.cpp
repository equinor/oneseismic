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

    /*
     * A HEARTBEAT from a worker not in the pool is clearly a sign that
     * something is wrong. This could be due to dropped READY message, network
     * instabilities or the queue being restarted. However, adding the worker to
     * the pool is probably the best we can do under the circumstances.
     */
    if (w == workers.end()) {
        spdlog::error(
            "Unexpected HEARTBEAT received from worker not in pool. This could "
            "be due to a dropped READY message, a network issue or the queue "
            "being restarted. The worker has been added back to the pool."
        );
        workers.push_back(worker{identity, expiry});
    }

    w->expiry = expiry;
}

void purge_workers(
    std::vector< worker >& workers,
    std::time_t current_time
) noexcept (false) {
    for (auto worker = workers.begin(); worker < workers.end(); ++worker) {
        if (worker->expiry < current_time)
            workers.erase(worker);
    }
}

bool send_to_worker(
    std::vector< worker >& workers,
    zmq::multipart_t task,
    zmq::socket_t& backend
) noexcept (false) {
    /*
     * Send task to a worker and remove worker from pool. Return false if send
     * fails with EAGAIN or EHOSTUNREACH, true if send is successful.
     *
     */
    auto identity = workers[0].identity;
    workers.erase(workers.begin());
    task.pushstr(identity);
    try {
        return task.send(backend);
    } catch (zmq::error_t& e) {
        if (e.num() == EHOSTUNREACH)
            return false;
        throw e;
    }
}

}

namespace one {

namespace detail { namespace load_balancer {

void run(
    zmq::socket_t& frontend,
    zmq::socket_t& backend,
    zmq::multipart_t& task,
    std::vector< worker >& available_workers,
    int heartbeat_interval,
    int heartbeat_liveness,
    std::time_t current_time
) noexcept (false) {

    zmq::pollitem_t items[] = {
            {static_cast< void* >(backend),  0, ZMQ_POLLIN, 0},
            {static_cast< void* >(frontend), 0, ZMQ_POLLIN, 0}
    };

    /*
     * Poll frontend only if there are available workers and the previous task
     * was successfully sent to a worker.
     */
    if (!available_workers.empty() and task.empty())
        zmq::poll(items, 2, heartbeat_interval);
    else
        zmq::poll(items, 1, heartbeat_interval);

    // Handle worker control messages from backend
    if (items[0].revents & ZMQ_POLLIN) {
        auto msg = zmq::multipart_t(backend);
        auto expiry = current_time + heartbeat_interval * heartbeat_liveness;

        auto identity = msg[0].to_string();
        auto message  = msg[1].to_string();

        if (std::strcmp(message.c_str(), "READY") == 0)
            available_workers.push_back(worker{identity, expiry});
        else if (std::strcmp(message.c_str(), "HEARTBEAT") == 0)
            update_worker(available_workers, identity, expiry);
        else
            spdlog::error("Invalid message from worker: {}", message);
    }

    /*
     * Push message from frontend to available worker. A copy of the message
     * is kept around in case the send fails.
     */
    if (items[1].revents & ZMQ_POLLIN) {
        task = zmq::multipart_t(frontend);
        if (send_to_worker(available_workers, task.clone(), backend))
            task.clear();
    }

    /*
     * If the message fails to send we try to send it to other workers in
     * the pool until the pool is exhausted, in which case the message is
     * kept around so we can try again when more workers are added to the
     * the pool.
     */
    while (!task.empty() and !available_workers.empty()) {
        if (send_to_worker(available_workers, task.clone(), backend))
            task.clear();
    }

    purge_workers(available_workers, current_time);
}

void configure_sockets(zmq::socket_t& frontend, zmq::socket_t& backend) {
    backend.setsockopt(ZMQ_ROUTER_MANDATORY, 1);
}

}}

[[noreturn]] void load_balancer(
    const std::string& front,
    const std::string& back,
    int heartbeat_interval,
    int heartbeat_liveness
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
     *     3. Messages will not be dropped if they fail to be sent to a worker.
     *        This has the cost of making a copy of each incoming message and
     *        slightly more convoluted control flow, but should be worth it due
     *        to the high cost of dropped messages (entire job must be
     *        rescheduled).
     *
     * TODO: If resilience in the form of failed jobs being rescheduled or
     *       similar, the need for the retry mechanism should be reevaluated.
     *
     * The advantage of this approach over PUSH-PULL is that jobs are sent to
     * idle workers, whereas PUSH-PULL distributes round robin. A disadvantage
     * is that the queue becomes a single point of failure, and a single point
     * for all messages to pass through, while PUSH-PULL is N-to-N.
     *
     * [1] https://zguide.zeromq.org/docs/chapter4/
     */
    zmq::context_t ctx;
    zmq::socket_t frontend(ctx, ZMQ_PULL);
    zmq::socket_t backend(ctx, ZMQ_ROUTER);
    detail::load_balancer::configure_sockets(frontend, backend);

    frontend.bind(front);
    backend.bind(back);

    zmq::multipart_t task;
    std::vector< worker > available_workers;

    while (true) {
        detail::load_balancer::run(
            frontend,
            backend,
            task,
            available_workers,
            heartbeat_liveness,
            heartbeat_interval,
            std::time(nullptr)
        );
    }
}

}