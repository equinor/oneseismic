#ifndef ONESEISMIC_TASKS_HPP
#define ONESEISMIC_TASKS_HPP

#include <memory>

#include <zmq.hpp>

namespace one {

class transfer;

class manifest_task {
public:
    void run(
            one::transfer& xfer,
            zmq::socket_t& input,
            zmq::socket_t& output,
            zmq::socket_t& fail)
        noexcept (false);

    manifest_task();
    ~manifest_task();

private:
    /*
     * a herb-style compilation firewall for storage
     * https://herbsutter.com/gotw/_100/
     */
    class impl;
    std::unique_ptr< impl > p;
};

class fragment_task {
public:
    void run(
            one::transfer& xfer,
            zmq::socket_t& input,
            zmq::socket_t& output)
        noexcept (false);

    fragment_task();
    ~fragment_task();

private:
    class impl;
    std::unique_ptr< impl > p;
};

}

#endif // ONESEISMIC_TASKS_HPP
