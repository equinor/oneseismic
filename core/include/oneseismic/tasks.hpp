#ifndef ONESEISMIC_TASKS_HPP
#define ONESEISMIC_TASKS_HPP

#include <zmq.hpp>

namespace one {

class transfer;

class manifest_task {
public:
    void run(
            one::transfer& xfer,
            zmq::socket_t& input,
            zmq::socket_t& output)
        noexcept (false);
};

}

#endif // ONESEISMIC_TASKS_HPP
