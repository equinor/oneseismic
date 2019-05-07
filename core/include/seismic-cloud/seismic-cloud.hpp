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

struct dimension : triple_comparison< dimension > {
    std::size_t x;
    std::size_t y;
    std::size_t z;

    dimension() = default;
    dimension(std::size_t x, std::size_t y, std::size_t z) noexcept (true) :
        x(x), y(y), z(z) {}
};

std::ostream& operator << (std::ostream& o, const point& rhs);
std::ostream& operator << (std::ostream& o, const dimension& rhs);

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
