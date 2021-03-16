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
    std::string        manifest;
    std::string        storage_endpoint;
    std::vector< int > shape;
    std::vector< int > shape_cube;
    std::string        function;

    std::string pack() const noexcept(false);
    void unpack(const char* fst, const char* lst) noexcept (false);
};

/*
 * The process header, which should be output by the scheduler/planner. It
 * describes the number of tasks the process has been split into and advices
 * the client on how to parse the response.
 *
 * The process header is written from a point of awareness of the shape of the
 * survey, so the shape tuple is the shape of the response *with padding*.
 *
 * The contents and order of the shape and index depend on the request type and
 * parameters.
 */
struct process_header {
    std::string        pid;
    int                ntasks;
    std::vector< int > shape;
    std::vector< std::vector< int > > index;

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

struct curtain_task : public common_task {
    curtain_task() = default;
    explicit curtain_task(const common_task& t) : common_task(t) {}

    std::vector< int > dim0s;
    std::vector< int > dim1s;

    std::string pack() const noexcept(false);
    void unpack(const char* fst, const char* lst) noexcept (false);
};

/*
 */
struct slice_fetch : public slice_task {
    slice_fetch() = default;
    explicit slice_fetch(const slice_task& t) : slice_task(t) {}

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

struct single {
    /* id is a 3-tuple (i,j,k) that gives the fragment-ID */
    std::vector< int > id;
    /*
     * coordinates is a 2-tuple (i', j') that gives the x/y position of the
     * trace. This is already a "local" coordinate a is 0-based.
     */
    std::vector< std::array< int, 2 > > coordinates;
};

struct curtain_fetch : public curtain_task {
    curtain_fetch() = default;
    explicit curtain_fetch(const curtain_task& t) : curtain_task(t) {}

    std::vector< single > ids;

    std::string pack() const noexcept(false);
    void unpack(const char* fst, const char* lst) noexcept (false);
};

struct trace {
    std::vector< int > coordinates;
    std::vector< float > v;
};

struct curtain_traces {
    std::vector< trace > traces;

    std::string pack() const noexcept(false);
    void unpack(const char* fst, const char* lst) noexcept (false);
};

}

#endif //ONESEISMIC_MESSAGES_HPP
