#ifndef SEISMIC_CLOUD_HPP
#define SEISMIC_CLOUD_HPP

#include <array>
#include <cassert>
#include <iostream>
#include <string>
#include <vector>

namespace sc {

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

    basic_tuple() = default;
    basic_tuple(const base_type& t) : base_type(t) {}

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

/*
 * TODO: these types deserve better naming and vocabulary
 */
template < std::size_t Dims >
struct cube_point : public basic_tuple< cube_point< Dims >, Dims > {
    using base_type = basic_tuple< cube_point, Dims >;
    using base_type::base_type;
};

template < std::size_t Dims >
struct frag_point : public basic_tuple< frag_point< Dims >, Dims > {
    using base_type = basic_tuple< frag_point, Dims >;
    using base_type::base_type;
};

template < std::size_t Dims >
struct fragment_id : public basic_tuple< fragment_id< Dims >, Dims > {
    using base_type = basic_tuple< fragment_id, Dims >;
    using base_type::base_type;
};

template< std::size_t Dims >
struct cube_dimension : public basic_tuple< cube_dimension< Dims >, Dims > {
    using base_type = basic_tuple< cube_dimension, Dims >;
    using base_type::base_type;

    std::size_t to_offset(cube_point< Dims > p)  const noexcept (true);
    std::size_t to_offset(fragment_id< Dims > p) const noexcept (true);
};

template< std::size_t Dims >
struct frag_dimension : public basic_tuple< frag_dimension< Dims >, Dims > {
    using base_type = basic_tuple< frag_dimension, Dims >;
    using base_type::base_type;

    std::size_t to_offset(frag_point< Dims > p) const noexcept (true);
};

template < std::size_t Dims >
struct dimension {
    dimension(std::size_t x) noexcept (false) : v(x) {
        if (x >= Dims)
            throw std::invalid_argument("invalid dimension");
    }

    std::size_t v;
};

/*
 * Map between different reference systems
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
 *
 * Names
 * -----
 *
 *  global, cube:
 *      The names global and cube always refer to the full survey, and is
 *      independent on how the system is fragmented
 *  local, frag:
 *      The names local and frag always refer to the individual fragments
 *      (subcubes) and their dimensions
 *  frag_id, anchor:
 *      IDs for fragments, which is also the coordinate of the fragment in the
 *      (coarsened) grid of fragments.
 */
template < std::size_t Dims >
class cubecoords {
    public:
        using Cube_dimension    = cube_dimension< Dims >;
        using Frag_dimension    = frag_dimension< Dims >;
        using Cube_point        = cube_point< Dims >;
        using Fragment_id       = fragment_id< Dims >;
        using Frag_point        = frag_point< Dims >;
        using Dimension         = dimension< Dims >;

        cubecoords(Cube_dimension, Frag_dimension) noexcept (true);

        /*
         * map global x, y, z -> m, n, k in the fragment
         */
        Frag_point to_local(Cube_point) const noexcept (true);
        /*
         * get the ID of the fragment that contains the global coordinate
         */
        Fragment_id frag_id(Cube_point) const noexcept (true);

        /*
         * get the fragment-IDs for a slice through the cube. Please note that
         * this operates on fragment grid resolution, so the pin refers to the
         * *fragment*, not the line.
         */
        std::vector< Fragment_id >
        slice(Dimension dim, std::size_t pin)
        noexcept (false);

        std::size_t size(Dimension) const noexcept (false);

        /*
         * map a fragment and coordinate-in-fragment to the global coordinate
         */
        Cube_point to_global(Fragment_id, Frag_point) const noexcept (true);

        /*
         * get the point of an offset, assuming the cube is flattened to a
         * large array, zs first.
         */
        //Cube_point from_offset(std::size_t) const noexcept (true);

        /*
         * the number of (x,y,z) triples, or points, in the cube
         */
        std::size_t global_size() const noexcept (true);

    private:
        Cube_dimension global_dims;
        Frag_dimension fragment_dims;
};

}

#endif //SEISMIC_CLOUD_HPP
