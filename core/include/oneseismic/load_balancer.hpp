#include <vector>

#ifndef ONESEISMIC_LOAD_BALANCER_HPP
#define ONESEISMIC_LOAD_BALANCER_HPP

namespace one {

[[noreturn]] void load_balancer(
    const std::string& frontend,
    const std::string& backend,
    int heartbeat_interval = 1000,
    int heartbeat_liveness = 3);

namespace detail { namespace load_balancer {

struct worker {
    std::string identity;
    std::time_t expiry;
};

void run(
    zmq::socket_t& frontend,
    zmq::socket_t& backend,
    zmq::multipart_t& task,
    std::vector< worker >& available_workers,
    int heartbeat_interval,
    int heartbeat_liveness,
    std::time_t current_time);

void configure_sockets(zmq::socket_t&, zmq::socket_t&);

}}

}

#endif //ONESEISMIC_LOAD_BALANCER_HPP