#ifndef ONESEISMIC_TEST_UTILS_HPP
#define ONESEISMIC_TEST_UTILS_HPP

#include <string>

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

/*
 * Make a reasonably unique PID by just concatenating the filename+lineno,
 * which still makes a valid redis key. Key collisions are important to avoid
 * in the unit tests to avoid both false positives and false negatives, as it
 * reduces the need to wipe the database between tests.
 *
 * oneseismic_expandstr() expands __LINE__ to an integer, then re-interprets it
 * as a string (char*)
 */
#define oneseismic_stringify(x) #x
#define oneseismic_expandstr(x) oneseismic_stringify(x)
#define makepid() std::string(__FILE__ ":" oneseismic_expandstr(__LINE__))

#endif //ONESEISMIC_TEST_UTILS_HPP
