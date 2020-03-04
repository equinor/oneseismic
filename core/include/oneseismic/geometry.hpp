#ifndef ONESEISMIC_GEOMETRY_HPP
#define ONESEISMIC_GEOMETRY_HPP

#include <array>
#include <cassert>
#include <iostream>
#include <string>
#include <vector>

namespace one {

template < typename Base, std::size_t Dims >
class basic_tuple : private std::array< std::size_t, Dims > {
    using base_type = std::array< std::size_t, Dims >;
    static_assert(Dims > 0, "0-tuples are non-sensical, is this a bug?");

    /*
     * Dimensionalities and coordinates are all structurally identical, but
     * semantically different. It makes perfect sense for them all to be
     * different types, but it's quite tedious and difficult to maintain
     * multiple identical implementations.
     *
     * Which means it's time to bring out the mixins with CRTP.
     */

public:

    static constexpr const auto dimensions = Dims;

    basic_tuple() noexcept (true) { this->fill(0); }
    basic_tuple(const base_type& t) noexcept (true) : base_type(t) {}

    template < typename... Vs >
    basic_tuple(std::size_t v, Vs... vs) noexcept (true) :
    basic_tuple(base_type{ v, static_cast< std::size_t >(vs) ... }) {
        /*
         * A super-duper hack to support brace-initialization, emplace back and
         * similar.
         *
         * Really, this boils down to something along the lines of:
         *
         *  basic_tuple(std::size_t x, std::size_t y, std::size_t z) :
         *      std::array<>({ x, y, z })
         *  {}
         *
         * but for N dimensions. The first argument is fixed to size_t (and not
         * variadic), otherwise arbitrary overloads are easily picked up on,
         * and a compile error quickly follows.
         *
         * It delegates to the array constructor, in case more behaviour should
         * be added to it.
         */
        static_assert(
            sizeof...(vs) + 1 == Dims,
            "constructor must have exactly Dims arguments"
        );
    }

    /* inherit methods from std::array */
    using base_type::operator [];
    using base_type::begin;
    using base_type::end;
    using base_type::rbegin;
    using base_type::rend;
    using base_type::size;

    std::string string() const;

    /*
     * Comparisons, but only within same type - no conversion!
     */
    friend
    bool operator != (const Base& left, const Base& right) noexcept (true) {
        const auto& lhs = static_cast< const base_type& >(left);
        const auto& rhs = static_cast< const base_type& >(right);
        return lhs != rhs;
    }

    friend
    bool operator == (const Base& left, const Base& right) noexcept (true) {
        const auto& lhs = static_cast< const base_type& >(left);
        const auto& rhs = static_cast< const base_type& >(right);
        return lhs == rhs;
    }

    friend
    bool operator < (const Base& left, const Base& right) noexcept (true) {
        const auto& lhs = static_cast< const base_type& >(left);
        const auto& rhs = static_cast< const base_type& >(right);
        return lhs < rhs;
    }

    friend
    bool operator <= (const Base& left, const Base& right) noexcept (true) {
        const auto& lhs = static_cast< const base_type& >(left);
        const auto& rhs = static_cast< const base_type& >(right);
        return lhs <= rhs;
    }

    friend
    bool operator > (const Base& left, const Base& right) noexcept (true) {
        const auto& lhs = static_cast< const base_type& >(left);
        const auto& rhs = static_cast< const base_type& >(right);
        return lhs > rhs;
    }

    friend
    bool operator >= (const Base& left, const Base& right) noexcept (true) {
        const auto& lhs = static_cast< const base_type& >(left);
        const auto& rhs = static_cast< const base_type& >(right);
        return lhs > rhs;
    }

    friend std::ostream&
    operator << (std::ostream& o, const Base& p) {
        static_assert(
            Dims > 1,
            "ostream << is only implemented for Dims > 1, "
            "fix it by writing a better join");

        /*
         * C++ :------------)
         *
         * '(' + ', '.join(*this) + ')'
         */
        o << '(';
        for (auto x = p.begin(); x != p.end() - 1; ++x)
            o << *x << ", ";
        return o << *(p.end() - 1) << ')';
    }
};

template < std::size_t ND >
struct dimension {
    template < typename T >
    explicit dimension(T x) noexcept (false) : v(std::size_t(x)) {
        if (x >= ND) {
            throw std::invalid_argument(
                  "invalid dimension: expected d (= "
                + std::to_string(x)
                + ") < ND (= "
                + std::to_string(ND)
                + ")"
            );
        }

        if (x < 0) {
            throw std::invalid_argument(
                  "invalid dimension: expected d (= "
                + std::to_string(x)
                + ") >= 0"
            );
        }

    }

    operator std::size_t () const noexcept (true) { return this->v; }
    std::size_t v;
};

/*
 * The slice_layout is a structure inspired by the slice (name is accidental!)
 * object in python [1]. slice_layout describes how slices are laid out in the
 * cube, and the related positions in an isolated fragment.
 *
 * All values are in number-of-units, so if the data is in 4-byte float, things
 * must be multiplied with sizeof(float) (= 4) to get the correct byte offset.
 *
 * By using the slice layout, a fragment can be fetched from storage, and the
 * slice in question can be extracted with a single loop.
 *
 * TODO: Example & illustration
 *
 * [1] The python slice object is a start-stop-step tuple, which when given to
 * range, operator[] etc yields the stride and initial offset to access only
 * certain elements of the list.
 */
struct slice_layout {
    /*
     * Number of read ops to perform:
     *
     * for (int i = 0; i < iterations; ++i)
     *     read(chunk_size)
     */
    int iterations;

    /*
     * Size of the chunk (in elements) to read at every read op.
     */
    int chunk_size;

    /*
     * initial_skip is the number of values to skip to get to the start of the
     * data. Note that this is must be multiplied with the index of the line in
     * question, for example.
     *
     * initial_skip is always applied to the superstride side of the
     * transformation.
     */
    int initial_skip;

    /*
     * The distance between a point and its lateral neighbour, which
     * corresponds to the *height* of the structure. Advance the write position
     * with this for every iteration.
     *
     * The super in superstride refers to it being a part of the larger
     * structure, and is refers to strides in the *cube*. When used in a
     * "flattened" cube, i.e. with a dimension set to 1, it is still a cube,
     * and a larger system.
     *
     * TODO: illustration
     */
    int superstride;

    /*
     * The distance between a point and lateral neighbour in an isolated
     * fragment, i.e. not part of a larger system.
     */
    int substride;
};

/*
 * Points and dimensions
 * =====================
 * All the examples in this section will deal with the more natural 3
 * dimensional case, but they are generalisable to N dimensions. An N-element
 * tuple of integers can represent most aspects of this system, such as
 * points/coordinates and the shape of volumes.
 *
 * These N-tuples share representation, but are quite different in terms of
 * semantics. They are made distinct types so that mixing them up is a
 * violation, i.e. you cannot pass coordinates meant for fragments to a
 * function that expects to know the size of a cube.
 *
 * The names are made acronyms, with the following pattern:
 *    C - Cube
 *    F - Fragment
 *    P - Point
 *    S - Shape
 *    ND - Number-of-dimensions
 *
 * where
 *  - C refers to a full survey volume
 *  - F refers to the fragments a C is partitioned into
 *  - P is a point/coordinate
 *  - S is the shape, shape of a C or F, and an upper bound of P
 *  - All have ND number of elements
 */

/*
 * CP - cube point
 *
 * The 0-based coordinates in the cube, i.e. the full survey volume. For a
 * volume, all cube points are unique, i.e. if two cube points are the same,
 * they refer to the same sample in the cube.
 *
 * It holds that CP[i] < CS[i]
 */
template < std::size_t ND >
struct CP : public basic_tuple< CP< ND >, ND > {
    using base_type = basic_tuple< CP, ND >;
    using base_type::base_type;
};

/*
 * FP - fragment point
 *
 * Similar to CP, but it refers to a point in a fragment. For a fragment, all
 * fragment points are unique, but a fragment point is *not* sufficient to
 * uniquely identify a sample value in a cube.
 *
 * It holds that FP[i] < FS[i]
 */
template < std::size_t ND >
struct FP : public basic_tuple< FP< ND >, ND > {
    using base_type = basic_tuple< FP, ND >;
    using base_type::base_type;
};

/*
 * FID - fragment ID
 *
 * The fragment ID is a unique identifier for a fragment, and corresponds to
 * the point in a structure where each fragment is considered a sample. Two
 * fragments are next to each other (share a face) in the cube if their IDs
 * differ by abs(1) in one direction, i.e. (1, 2, 2) is a neighbour of (1, 2,
 * 1), but not a neighbour of (2, 2, 1) .
 */
template < std::size_t ND >
struct FID : public basic_tuple< FID< ND >, ND > {
    using base_type = basic_tuple< FID, ND >;
    using base_type::base_type;
};

/*
 * CS - cube shape
 *
 * The shape of a cube/volume.
 */
template< std::size_t ND >
struct CS : public basic_tuple< CS< ND >, ND > {
    using base_type = basic_tuple< CS, ND >;
    using base_type::base_type;

    std::size_t to_offset(CP< ND >)  const noexcept (true);
    std::size_t to_offset(FID< ND >) const noexcept (true);
    std::size_t slice_samples(dimension< ND >) const noexcept (true);
};

/*
 * FS - fragment shape
 *
 * The shape of a fragment.
 */
template< std::size_t ND >
struct FS : public basic_tuple< FS< ND >, ND > {
    using base_type = basic_tuple< FS, ND >;
    using base_type::base_type;

    std::size_t to_offset(FP< ND >) const noexcept (true);
    std::size_t slice_samples(dimension< ND >) const noexcept (true);
    slice_layout slice_stride(dimension< ND >) const noexcept (false);
};

/*
 * gvt - global volume translation
 *
 * Map between different reference systems. This is the source of truth for
 * addresses, fragment IDs, and geometric information. The problem of mapping
 * between these systems pop up all the time:
 *
 * - How big is the source total cube?
 * - How big is the padded total cube?
 * - How many fragments aret here?
 * - How big are the samples?
 * - Where do fragment values map into an extracted slice?
 *
 * gvt is the central component to get this questions answered. It is
 * lightweight and cheap to copy, and to be considered immutable as it's
 * tightly connected to its cube (for a specific fragmentation).
 *
 * --
 *
 * Consider the flat cube with coordinates x, y:
 *
 *     0,0   0,3   0,5   0,7
 *      +-----+-----+-----+
 *      |  1  |  2  |  3  |
 *      |     |     |     |
 *  2,0 +-----+-----+-----+ 6,7
 *      |  4  |  5  |  6  |
 *      |     |     |     |
 *  4,0 +-----+-----+-----+ 6,7
 *      |  7  |  8  |  9  |
 *      |     |     |     |
 *  6,0 +-----+-----+-----+ 6,7
 *      |  A  |  B  |  C  |
 *      |     |     |     |
 *      +-----+-----+-----+
 *     8,0   8,3   8,5   8,7
 *
 * This consists of 12 smaller fragments 1..C, which can internally be indexed
 * m, n:
 *
 * Fragment 5
 * ----------
 *     0,0 0,1 0,2
 *      +---+---+
 *      | 1 | 2 |
 *  1,0 +---+---+ 1,2
 *      | 3 | 4 |
 *      +---+---+
 *     2,0 2,1 2,2
 *
 *
 * The global coordinate 3,4 would map to fragment 5, coordinates 1,1. Each
 * fragment is named and identified by its position in the grid of *fragments*,
 * i.e. top-left fragment is (0,0), next to the right is (0,1) etc.
 */
template < std::size_t ND >
class gvt {
    public:
        using Dimension = dimension< ND >;
        /*
         * The cube dimension is the source, un-padded cube dimension, which
         * *must* be rectangular. If the source survey tapers like this:
         *        ____
         *       /    \
         *      |      |
         *      |      |
         *      |      |
         *      |      |
         *      +------+
         *
         * the correct cube dimensions are still:
         *
         *      +------+
         *      |      |
         *      |      |
         *      |      |
         *      |      |
         *      |      |
         *      +------+
         *
         * i.e. the cube must *internally* filled with data, so the dimensions
         * are rectangular. It will therefore always hold that:
         *      cubedim[i] <= count(fragdim[i]) * size(fragdim[i])
         */
        gvt(CS< ND >, FS< ND >) noexcept (true);

        /*
         * map global x, y, z -> m, n, k in the fragment. This is quite useful
         * when extracting arbitrary surfaces, where this function gives the
         * m,n,k in the fragment returned by frag_id with the same parameter.
         */
        FP< ND > to_local(CP< ND >) const noexcept (true);
        /*
         * get the ID of the fragment that contains the global coordinate.
         *
         * See: to_local
         */
        FID< ND > frag_id(CP< ND >) const noexcept (true);

        /*
         * Get the fragment-IDs for a slice through the cube. Please note that
         * this operates on fragment grid resolution, so the pin refers to the
         * *fragment*, not the line.
         */
        std::vector< FID< ND > >
        slice(Dimension dim, std::size_t n) noexcept (false);

        /*
         * The slice layout for putting a single fragment into a cube (usually
         * compacted with the slice-direction = 1)
         */
        slice_layout slice_stride(Dimension, FID< ND >) const noexcept (true);

        /*
         * The number of fragments in a direction.
         */
        std::size_t fragment_count(Dimension) const noexcept (false);

        const CS< ND >& cube_shape()     const noexcept (true);
        const FS< ND >& fragment_shape() const noexcept (true);

        /*
         * Map a fragment and coordinate-in-fragment to the global coordinate.
         * It holds that:
         *      x, y, z == to_global(frag_id(x, y, z), to_local(x, y, z))
         */
        CP< ND > to_global(FID< ND >, FP< ND >) const noexcept (true);

        /*
         * The number of (x,y,z) triples, or points, in the cube.
         */
        std::size_t global_size() const noexcept (true);

        /*
         * Number of samples padded in direction d.
         */
        int padding(FID< ND > id, Dimension d) const noexcept (true);

    private:
        CS< ND > global_dims;
        FS< ND > fragment_dims;
};

}

#endif //ONESEISMIC_GEOMETRY_HPP
