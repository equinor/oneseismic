#include <vector>
#include <zmq.hpp>

#ifndef ONESEISMIC_LOAD_BALANCER_HPP
#define ONESEISMIC_LOAD_BALANCER_HPP

namespace one {

void load_balancer(
    zmq::socket_t& frontend,
    zmq::socket_t& backend,
    zmq::socket_t& control,
    std::time_t request_timeout
);

}

#endif //ONESEISMIC_LOAD_BALANCER_HPP