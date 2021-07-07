#ifndef ONESEISMIC_MESSAGES_HPP
#define ONESEISMIC_MESSAGES_HPP

#include <stdexcept>
#include <string>
#include <vector>

namespace one {

template < typename T >
struct Packable {
    std::string pack() const noexcept (false);
    void unpack(const char* fst, const char* lst) noexcept (false);
};

template < typename T >
struct MsgPackable {
    std::string pack() const noexcept (false);
    void unpack(const char* fst, const char* lst) noexcept (false);
};

class bad_message : public std::runtime_error {
    using std::runtime_error::runtime_error;
};

class bad_document : public std::runtime_error {
    using std::runtime_error::runtime_error;
};

class bad_value : public std::runtime_error {
    using std::runtime_error::runtime_error;
};

struct not_found : public std::out_of_range {
    using std::out_of_range::out_of_range;
};

struct volumedesc {
    std::string prefix; /* e.g. src/, attributes/ */
    std::string ext;    /* file-extension */
    std::vector< std::vector< int > > shapes;
};

struct attributedesc {
    std::string prefix; /* e.g. src/, attributes/ */
    std::string ext;    /* file-extension */
    std::string type;   /* e.g. cdp, utm */
    std::string layout; /* e.g. tiled */
    std::vector< std::string > labels;
    std::vector< std::vector< int > > shapes;
};

struct manifestdoc {
    std::vector< volumedesc >           vol;
    std::vector< attributedesc >        attr;
    std::vector< std::vector< int > >   line_numbers;
    std::vector< std::string >          line_labels;
};

/*
 * The *query messages are parsing utilities for the input messages built from
 * the graphql queries. They help build a corresponding *task which is fed to
 * the workers, and should contain all the information the workers need to do
 * their job.
 */
struct basic_query {
    std::string        pid;
    std::string        token;
    std::string        guid;
    manifestdoc        manifest;
    std::string        storage_endpoint;
    std::vector< int > shape;
    std::string        function;
};

struct basic_task {
    basic_task() = default;
    explicit basic_task(const basic_query& q) :
        pid              (q.pid),
        token            (q.token),
        guid             (q.guid),
        storage_endpoint (q.storage_endpoint),
        shape            (q.shape),
        function         (q.function)
    {
        this->shape_cube.reserve(q.manifest.line_numbers.size());
        for (const auto& d : q.manifest.line_numbers)
            this->shape_cube.push_back(d.size());
    }

    std::string        pid;
    std::string        token;
    std::string        guid;
    std::string        storage_endpoint;
    std::vector< int > shape;
    std::vector< int > shape_cube;
    std::string        function;
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
struct process_header : Packable< process_header > {
    std::string        pid;
    int                ntasks;
    std::vector< int > shape;
    std::vector< std::vector< int > > index;
};

struct slice_query : public basic_query, Packable< slice_query > {
    int dim;
    int idx;
};

struct curtain_query : public basic_query, Packable< curtain_query > {
    std::vector< int > dim0s;
    std::vector< int > dim1s;
};

/*
 */
struct slice_task : public basic_task, Packable< slice_task > {
    slice_task() = default;
    explicit slice_task(const slice_query& q) :
        basic_task(q),
        dim(q.dim)
    {}

    int dim;
    int idx;
    std::vector< std::vector< int > > ids;
};

struct tile {
    int iterations;
    int chunk_size;
    int initial_skip;
    int superstride;
    int substride;
    std::vector< float > v;
};

struct slice_tiles : public MsgPackable< slice_tiles > {
    /*
     * The shape of the slice itself
     */
    std::vector< int > shape;
    std::vector< tile > tiles;
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

struct curtain_task : public basic_task, Packable< curtain_task > {
    using basic_task::basic_task;
    std::vector< single > ids;
};

struct trace {
    std::vector< int > coordinates;
    std::vector< float > v;
};

struct curtain_traces : public MsgPackable< curtain_traces > {
    std::vector< trace > traces;
};

}

#endif //ONESEISMIC_MESSAGES_HPP
