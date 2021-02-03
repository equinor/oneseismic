#ifndef ONESEISMIC_TASKS_HPP
#define ONESEISMIC_TASKS_HPP

#include <memory>

#include <zmq.hpp>

namespace one {

class transfer;

class manifest_task {
public:
    void run(
            zmq::socket_t& input,
            zmq::socket_t& output,
            zmq::socket_t& fail)
        noexcept (false);

    void connect_working_storage(const std::string&);

    int  max_task_size() const noexcept (true);
    void max_task_size(int)    noexcept (false);

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

}

#endif // ONESEISMIC_TASKS_HPP
