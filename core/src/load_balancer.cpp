#include <spdlog/spdlog.h>
#include <zmq.hpp>
#include <zmq_addon.hpp>

#include <oneseismic/load_balancer.hpp>

namespace one {

void load_balancer(
        zmq::socket_t& frontend,
        zmq::socket_t& backend,
        zmq::socket_t& control,
        std::time_t request_timeout
) {
    /*
     * Message broker that will distribute tasks to workers as they become
     * available.
     *
     * The advantage of this approach over PUSH-PULL is that jobs are sent to
     * idle workers, whereas PUSH-PULL distributes round robin. A disadvantage
     * is that the queue becomes a single point of failure, and a single point
     * for all messages to pass through, while PUSH-PULL is N-to-N.
     */

    while (true) {
        zmq::pollitem_t items[] = {
            { static_cast< void* >(frontend),  0, ZMQ_POLLIN, 0 },
            { static_cast< void* >(control), 0, ZMQ_POLLIN, 0 },
        };

        int poll = zmq::poll(items, 2, request_timeout);

        /*
         * Send an empty message to workers on timeout event. This will trigger
         * a new request from the workers.
         */
        if (poll == 0) {
            zmq::message_t msg;
            while (backend.recv(msg, zmq::recv_flags::dontwait)) {
                backend.send(zmq::message_t(), zmq::send_flags::none);
            }
        }

        /*
         *
         */
        if (items[0].revents & ZMQ_POLLIN) {
            auto task = zmq::multipart_t(frontend);
            zmq::message_t msg;
            backend.recv(msg, zmq::recv_flags::none);
            task.send(backend);
        }

        if (items[1].revents & ZMQ_POLLIN) {
            break;
        }
    }
}

}