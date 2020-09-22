#ifndef ONESEISMIC_MESSAGES_HPP
#define ONESEISMIC_MESSAGES_HPP

#include <stdexcept>
#include <string>
#include <vector>

namespace one {

class bad_message : public std::runtime_error {
    using std::runtime_error::runtime_error;
};


/*
 * The basic message, and the fields that *all* tasks share. The only reason
 * for inheritance to even play here is just to make the implementation a lot
 * shorter.
 *
 * The load() function is *not* in exception safe, and will modify
 * member-by-member in-place. That means the moment load() is called, the
 * instance should be considered unspecified in case an exception is thrown.
 *
 * load takes a range of bytes, rather than being explicit here on the encoding
 * of the message. This should make all other code more robust should when the
 * message body itself changes (different protocols, added fields etc), as only
 * the load() internals must be changed.
 *
 * This is the message that's sent by the api/ when scheduling work
 */
struct common_task {
    std::string        pid;
    std::string        token;
    std::string        guid;
    std::string        storage_endpoint;
    std::vector< int > shape;
    std::string        function;

    std::string pack() const noexcept(false);
    void unpack(const char* fst, const char* lst) noexcept (false);
};

struct slice_task : public common_task {
    slice_task() = default;
    explicit slice_task(const common_task& t) : common_task(t) {}

    int dim;
    int lineno;

    std::string pack() const noexcept(false);
    void unpack(const char* fst, const char* lst) noexcept (false);
};

/*
 */
struct slice_fetch : public slice_task {
    slice_fetch() = default;
    explicit slice_fetch(const slice_task& t) : slice_task(t) {}

    std::vector< int > cube_shape;
    std::vector< std::vector< int > > ids;

    std::string pack() const noexcept (false);
    void unpack(const char* fst, const char* lst) noexcept (false);
};

struct tile {
    int iterations;
    int chunk_size;
    int initial_skip;
    int superstride;
    int substride;
    std::vector< float > v;
};

struct slice_tiles {
    /*
     * The shape of the slice itself
     */
    std::vector< int > shape;
    std::vector< tile > tiles;

    std::string pack() const noexcept(false);
    void unpack(const char* fst, const char* lst) noexcept (false);
};

}

#endif //ONESEISMIC_MESSAGES_HPP
