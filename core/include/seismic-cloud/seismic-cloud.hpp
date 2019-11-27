#ifndef SEISMIC_CLOUD_HPP
#define SEISMIC_CLOUD_HPP

#include <iostream>
#include <vector>

namespace sc {

template < typename T >
struct triple_comparison {
    bool operator < (const T& rhs) const noexcept (true) {
        const auto& self = *static_cast< const T* >(this);
        if (self.x < rhs.x) return true;
        if (self.x > rhs.x) return false;
        if (self.y < rhs.y) return true;
        if (self.y > rhs.y) return false;
        if (self.z < rhs.z) return true;
        if (self.z > rhs.z) return false;
        return false;
    }

    bool operator > (const T& rhs) const noexcept (true) {
        const auto& self = *static_cast< const T* >(this);
        if (self.x > rhs.x) return true;
        if (self.x < rhs.x) return false;
        if (self.y > rhs.y) return true;
        if (self.y < rhs.y) return false;
        if (self.z > rhs.z) return true;
        if (self.z < rhs.z) return false;
        return false;
    }

    bool operator <= (const T& rhs) const noexcept (true) {
        const auto& self = *static_cast< const T* >(this);
        return self < rhs or self == rhs;
    }

    bool operator >= (const T& rhs) const noexcept (true) {
        const auto& self = *static_cast< const T* >(this);
        return self > rhs or self == rhs;
    }

    bool operator == (const T& rhs) const noexcept (true) {
        const auto& self = *static_cast< const T* >(this);
        return self.x == rhs.x
           and self.y == rhs.y
           and self.z == rhs.z
        ;
    }

    bool operator != (const T& rhs) const noexcept (true) {
        const auto& self = *static_cast< const T* >(this);
        return self.x != rhs.x
            or self.y != rhs.y
            or self.z != rhs.z
        ;
    }
};

template < typename Point >
struct basic_point : triple_comparison< Point > {
    std::size_t x;
    std::size_t y;
    std::size_t z;

    basic_point() noexcept (true) = default;
    basic_point(std::size_t x, std::size_t y, std::size_t z) noexcept (true) :
        x(x), y(y), z(z) {}

    friend std::ostream&
    operator << (std::ostream& o, const basic_point< Point >& p) {
        return o << "(" << p.x << ", " << p.y << ", " << p.z << ")";
    }
};

struct cube_point : basic_point< cube_point > {
    using basic_point::basic_point;
    constexpr static const char* name = "cube_point";
};

struct frag_point : basic_point< frag_point > {
    using basic_point::basic_point;
    constexpr static const char* name = "frag_point";
};

struct fragment_id : basic_point< fragment_id > {
    using basic_point::basic_point;
    constexpr static const char* name = "fragment_id";

    cube_point operator + (frag_point) const noexcept (true);
};

template < typename Dim >
struct basic_dim : triple_comparison< Dim > {
    std::size_t x;
    std::size_t y;
    std::size_t z;

    basic_dim() = default;
    basic_dim(std::size_t x, std::size_t y, std::size_t z) noexcept (true) :
        x(x), y(y), z(z) {}
};

struct cube_dim : basic_dim< cube_dim > {
    using basic_dim::basic_dim;
    std::size_t to_offset(cube_point p) const noexcept (true);
    std::size_t to_offset(fragment_id p) const noexcept (true);
};

struct frag_dim : basic_dim< cube_dim > {
    using basic_dim::basic_dim;
    std::size_t to_offset(frag_point p) const noexcept (true);
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
class cubecoords {
    public:
        cubecoords(cube_dim, frag_dim) noexcept (true);

        /*
         * map global x, y, z -> m, n, k in the fragment
         */
        frag_point to_local(cube_point) const noexcept (true);
        /*
         * get the ID of the fragment that contains the global coordinate
         */
        fragment_id frag_id(cube_point) const noexcept (true);

        /*
         * map a fragment and coordinate-in-fragment to the global coordinate
         */
        cube_point to_global(fragment_id, frag_point) const noexcept (true);

        /*
         * get the point of an offset, assuming the cube is flattened to a
         * large array, zs first.
         */
        cube_point from_offset(std::size_t) const noexcept (true);

        /*
         * the number of (x,y,z) triples, or points, in the cube
         */
        std::size_t global_size() const noexcept (true);

    private:
        cube_dim global_dims;
        frag_dim fragment_dims;
};

}

#endif //SEISMIC_CLOUD_HPP
