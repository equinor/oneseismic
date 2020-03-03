#ifndef ONESEISMIC_WORKER_HPP
#define ONESEISMIC_WORKER_HPP

#include <zmq.hpp>

namespace one {

class transfer;

void workloop(transfer&, zmq::socket_t& in, zmq::socket_t& out);

}

#endif //ONESEISMIC_WORKER_HPP
