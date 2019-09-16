#ifndef SEISMIC_CLOUD_HPP
#define SEISMIC_CLOUD_HPP

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

struct point : triple_comparison< point > {
    std::size_t x;
    std::size_t y;
    std::size_t z;

    point() = default;
    point(std::size_t x, std::size_t y, std::size_t z) noexcept (true) :
        x(x), y(y), z(z) {}
};

template < typename Point >
struct basic_point : triple_comparison< Point > {
    std::size_t x;
    std::size_t y;
    std::size_t z;

    basic_point() noexcept (true) = default;
    basic_point(std::size_t x, std::size_t y, std::size_t z) noexcept (true) :
        x(x), y(y), z(z) {}
};

struct cube_point : basic_point< cube_point > {
    using basic_point::basic_point;
    constexpr static const char* name = "cube_point";
};

struct frag_point : basic_point< frag_point > {
    using basic_point::basic_point;
    constexpr static const char* name = "frag_point";
};

struct root_point : basic_point< root_point > {
    using basic_point::basic_point;
    constexpr static const char* name = "root_point";

    cube_point operator + (frag_point) const noexcept (true);
};

struct dimension : triple_comparison< dimension > {
    std::size_t x;
    std::size_t y;
    std::size_t z;

    dimension() = default;
    dimension(std::size_t x, std::size_t y, std::size_t z) noexcept (true) :
        x(x), y(y), z(z) {}
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
    std::size_t to_offset(root_point p) const noexcept (true);
};

struct frag_dim : basic_dim< cube_dim > {
    using basic_dim::basic_dim;
    std::size_t to_offset(frag_point p) const noexcept (true);
};

std::ostream& operator << (std::ostream& o, const point& rhs);
std::ostream& operator << (std::ostream& o, const dimension& rhs);

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
 * fragment is identified by it's root coordinate, the fragment-local 0,0 in
 * the top-left coorner.
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
 *  root, anchor:
 *      The names root and anchors work as identifiers for individual
 *      fragments, and is the upper-left corner (0,0) in the global system of a
 *      specific fragment
 */
class cubecoords {
    public:
        cubecoords(cube_dim, frag_dim) noexcept (true);

        /*
         * map global x, y, z -> m, n, k in the fragment
         */
        frag_point to_local(cube_point) const noexcept (true);
        /*
         * get the root (fragment-ID) of any global coordinate
         */
        root_point root_of(cube_point) const noexcept (true);

        /*
         * map a root and local coordinate to the global coordinate. This is
         * the inverse operation of root_of and to_local,
         * x == root_of(x) + to_local(x)
         */
        cube_point to_global(root_point, frag_point) const noexcept (true);

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

point global_to_local(point global, dimension fragment_size) noexcept (true);
point local_to_global(point local, point root)               noexcept (true);
point global_to_root(point global, dimension fragment_size)  noexcept (true);

point offset_to_point(std::size_t offset, dimension dim) noexcept (true);
std::size_t point_to_offset( point p, dimension dim )    noexcept (true);
std::size_t local_to_global(std::size_t local,
                            dimension fragment_size,
                            dimension cube_size,
                            point root) noexcept (true);

struct bins {
    std::vector< point > keys;
    std::vector< std::size_t > itrs;
    std::vector< std::size_t > data;

    using iterator = decltype(data.cbegin());

    struct bin {
        point key;
        iterator first;
        iterator last;

        bin() = default;
        bin(iterator fst, iterator lst) : first(fst), last(lst) {}

        iterator begin() const noexcept (true) {
            return this->first;
        }

        iterator end() const noexcept (true) {
            return this->last;
        }
    };

    bin at(std::size_t i) const noexcept (true) {
        bin x;
        x.first = this->data.begin() + this->itrs[i];
        x.last  = this->data.begin() + this->itrs[i + 1];
        x.key   = this->keys[i];
        return x;
    }
};

bins bin(dimension fragment_size,
         dimension cube_size,
         const std::vector< point >& xs) noexcept (false);

}

#endif //SEISMIC_CLOUD_HPP
