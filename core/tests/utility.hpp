#ifndef ONESEISMIC_TEST_UTILS_HPP
#define ONESEISMIC_TEST_UTILS_HPP

#include <catch/catch.hpp>
#include <zmq.hpp>

namespace {

bool received_message(zmq::socket_t& socket) {
    zmq::pollitem_t queue[] = {
        { static_cast< void* >(socket), 0, ZMQ_POLLIN, 0 },
    };

    const auto polled = zmq::poll(queue, 1, 0);

    if (polled == -1)
        FAIL("Polling the socket failed");

    return polled == 1;
}

}

#endif //ONESEISMIC_TEST_UTILS_HPP
