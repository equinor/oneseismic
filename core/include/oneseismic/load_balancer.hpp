#include <vector>

#ifndef ONESEISMIC_LOAD_BALANCER_HPP
#define ONESEISMIC_LOAD_BALANCER_HPP

namespace one {

[[noreturn]] void load_balancer(
    zmq::socket_t& frontend,
    zmq::socket_t& backend,
    int ready_ttl);

namespace detail { namespace load_balancer {

struct worker {
    std::string identity;
    std::time_t expiry;
};

void run(
    zmq::socket_t& frontend,
    zmq::socket_t& backend,
    std::vector< worker >& available_workers,
    int ready_ttl,
    std::time_t current_time);

}}

}

#endif //ONESEISMIC_LOAD_BALANCER_HPP