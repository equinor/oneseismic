#ifndef ONESEISMIC_MESSAGES_HPP
#define ONESEISMIC_MESSAGES_HPP

#include <array>
#include <optional>
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
    std::vector< volumedesc >                             vol;
    std::vector< attributedesc >                          attr;
    std::vector< std::vector< int > >                     line_numbers;
    std::vector< std::string >                            line_labels;
    std::optional< std::vector< std::vector< double > > > utm_to_lineno;
};

/*
 * The *query messages are parsing utilities for the input messages built from
 * the graphql queries. They help build a corresponding *task which is fed to
 * the workers, and should contain all the information the workers need to do
 * their job.
 */
struct basic_query {
    std::string                 pid;
    std::string                 token;
    std::string                 url_query;
    std::string                 guid;
    manifestdoc                 manifest;
    std::string                 storage_endpoint;
    std::string                 function;
    std::vector< std::string >  attributes;

    const std::vector< int >& shape() const noexcept (false) {
        /*
         * When support is in place, users (and oneseismic itself, really) can
         * hint at what shape would be better for a particular query, but at
         * the end of the day it has to query a fragment set that's available.
         * The optimal cell size very much depends on the type of query, so
         * specific *_query objects might want to implement specific logic for
         * the shape() function. Now, it is probably sufficient to pick the
         * first (and usually only) shape.
         *
         * The use of at() is deliberate to detect badly-formatted manifests.
         */
        try {
            return this->manifest.vol.at(0).shapes.at(0);
        } catch (std::out_of_range&) {
            throw bad_document("Missing data or shape field");
        }
    }
};

struct basic_task {
    basic_task() = default;
    explicit basic_task(const basic_query& q) :
        pid              (q.pid),
        token            (q.token),
        url_query        (q.url_query),
        guid             (q.guid),
        prefix           (q.manifest.vol.at(0).prefix),
        ext              (q.manifest.vol.at(0).ext),
        storage_endpoint (q.storage_endpoint),
        shape            (q.shape()),
        function         (q.function),
        attribute        ("data")
    {
        this->shape_cube.reserve(q.manifest.line_numbers.size());
        for (const auto& d : q.manifest.line_numbers)
            this->shape_cube.push_back(d.size());
    }

    basic_task(const basic_query& q, const attributedesc& attr) :
        pid              (q.pid),
        token            (q.token),
        url_query        (q.url_query),
        guid             (q.guid),
        prefix           (attr.prefix),
        ext              (attr.ext),
        storage_endpoint (q.storage_endpoint),
        shape            (attr.shapes.at(0)),
        function         (q.function),
        attribute        (attr.type)
    {
        this->shape_cube.reserve(q.manifest.line_numbers.size());
        for (const auto& d : q.manifest.line_numbers)
            this->shape_cube.push_back(d.size());
        this->shape_cube.back() = 1;
    }

    std::string        pid;
    std::string        token;
    std::string        url_query;
    std::string        guid;
    std::string        storage_endpoint;
    std::string        prefix;
    std::string        ext;
    std::vector< int > shape;
    std::vector< int > shape_cube;
    std::string        function;
    std::string        attribute;
};

/*
 * The process header, which should be output by the scheduler/planner. It
 * describes the number of tasks the process has been split into and advices
 * the client on how to parse the response.
 *
 * The index is context sensitive, and the content depends on the structure
 * queried - for example, for a slice(dim: 0) it is the crossline numbers and
 * time/depth interval.
 *
 * The index is laid out linearly, fortran style, and the first ndims items are
 * the dimensions. Conceptually it works like this:
 *
 * {
 *  ndims: 2
 *  index: [3 5 [n1 n2 n3] [m1 m2 m3 m4 m5]]
 * }
 *
 * While it makes the structure slightly less intuitive, it makes parsing and
 * serializing a lot simpler in many (otherwise clumsy) cases.
 *
 */

enum class functionid {
    slice   = 1,
    curtain = 2,
};

struct process_header : MsgPackable< process_header > {
    std::string                         pid;
    functionid                          function;
    int                                 nbundles;
    int                                 ndims;
    std::vector< int >                  index;
    std::vector< std::string >          labels;
    std::vector< std::string >          attributes;
    std::vector< int >                  shapes;
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

    slice_task(const slice_query& q, const attributedesc& attr) :
        basic_task(q, attr),
        dim(q.dim)
    {}

    int dim;
    int idx;
    std::vector< std::array< int, 3 > > ids;
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
    std::string attr;
    std::vector< tile > tiles;

    std::string pack()   const noexcept (false);
    void unpack(const char* fst, const char* lst) noexcept (false);
};

struct single {
    /* id is a 3-tuple (i,j,k) that gives the fragment-ID */
    std::array< int, 3 > id;

    /*
     * The offset is the index of this fragment in the lexicographically sorted
     * set of fragments that make up a query. It is very useful for efficient
     * extraction, and is just a mechanism for carrying the ordering of
     * sub-tasks across boundaries, but carries no other semantics.
     */
    int offset;
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

struct curtain_bundle {
    /*
     * This message describes correspond to traces all pulled from a single
     * fragment [1], with a parallel-array layout.
     *
     * The curtain is an arbitrary collection of traces [(x1,y1), (x2,y2),
     * ...], but every bundle of data holds only small pieces of the trace. An
     * index is used to map segments onto the final result, but to reduce
     * overhead it is compacted significantly.
     *
     * The index is composed of major and minor, and a simple algorithm to
     * expand them; the majors serve as "keyframes", which correspond to
     * (i,j,_) trace blocks in the lexicographically sorted request. The minors
     * are "block-local" offsets. Put slightly differently - the major gives
     * the fragment, the minors the trace segment *in* the fragment.
     *
     * Both the major and minor are laid out in [fst,lst) pairs; the length of
     * either array is 2*size [2]. This is slightly redundant, but makes the
     * structure much easier to use.
     *
     * The output is a 2-dimensional array with output-shape. In numpy syntax,
     * extraction then becomes:
     *
     *    out[maj[i]:maj[i+1], min[i]:min[i+1]] = ...
     *
     * i.e. the major slices the 1st axis, the minor slices the 2nd axis.
     *
     * The parallel array layout is used because it fast and easy to pack and
     * parse, and because it reduces the need for a dynamic and labelled
     * structure. This is a detail that partially comes from using a statically
     * typed language, but it makes for nice messages regardless.
     *
     * The zlength is the height of the output in the z dimension; this is
     * trace-length for data, and usually 1 for attributes. It's embedded in
     * the message to make decoding easier, and to handle more shapes.
     *
     * [1] conceptually, although multiple fragments may be merged
     * [2] this might get changed to multiple shorter arrays
     */
    std::string attr;
    int size;
    int zlength;
    std::vector< int > major;
    std::vector< int > minor;
    std::vector< float > values;

    std::string pack() const noexcept (false);
    void unpack(const char* fst, const char* lst) noexcept (false);
};

namespace detail {

std::pair< int, int > utm_to_cartesian(
    const std::vector< int >& inlines,
    const std::vector< int >& crosslines,
    const std::vector< std::vector< double > >& utm_to_lino,
    float x,
    float y
) noexcept (false);

}

}

#endif //ONESEISMIC_MESSAGES_HPP
