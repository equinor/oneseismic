#ifndef SEISMIC_CLOUD_HPP
#define SEISMIC_CLOUD_HPP

#include <cassert>

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

std::ostream& operator << (std::ostream& o, const point& rhs) {
    return o << "(" << rhs.x << ", " << rhs.y << ", " << rhs.z << ")";
}

std::ostream& operator << (std::ostream& o, const dimension& rhs) {
    return o << "(" << rhs.x << ", " << rhs.y << ", " << rhs.z << ")";
}

point global_to_local( point global, dimension fragment_size ) {
    return {
        global.x % fragment_size.x,
        global.y % fragment_size.y,
        global.z % fragment_size.z,
    };
}

point local_to_global( point local, point root ) noexcept (true) {
    return {
        local.x + root.x,
        local.y + root.y,
        local.z + root.z,
    };
}

point global_to_root( point global, dimension fragment_size ) {
    return {
        (global.x / fragment_size.x) * fragment_size.x,
        (global.y / fragment_size.y) * fragment_size.y,
        (global.z / fragment_size.z) * fragment_size.z,
    };
}

std::size_t point_to_offset( point p, dimension dim ) {
    return p.x * dim.y * dim.z
         + p.y * dim.z
         + p.z;
}

point offset_to_point( std::size_t offset, dimension dim ) {
    return {
        (offset / (dim.y * dim.z)),
        (offset % (dim.y * dim.z)) / dim.z,
        (offset % (dim.y * dim.z)) % dim.z
    };
}

std::size_t local_to_global( std::size_t local,
                             dimension fragment_size,
                             dimension cube_size,
                             point root ) {

    auto local_point = offset_to_point( local, fragment_size );
    auto global = local_to_global( local_point, root );

    return point_to_offset( global, cube_size );
}

struct bins {
    std::vector< sc::point > keys;
    std::vector< std::size_t > itrs;
    std::vector< std::size_t > data;

    using iterator = decltype(data.cbegin());

    struct bin {
        sc::point key;
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

bins bin(sc::dimension fragment_size,
         sc::dimension cube_size,
         const std::vector< sc::point >& xs) noexcept (false) {

    using key = std::pair< sc::point, std::size_t >;
    auto points = std::vector< key >(xs.size());

    auto fragment_id = [fragment_size](const auto& p) noexcept (true) {
        const auto root  = sc::global_to_root(p, fragment_size);
        const auto local = sc::global_to_local(p, fragment_size);
        const auto pos   = sc::point_to_offset(local, fragment_size);
        return std::make_pair(root, pos);
    };

    std::transform(xs.begin(), xs.end(), points.begin(), fragment_id);
    std::sort(points.begin(), points.end());

    /*
     * If the input surface xs is empty, just return here.  should work and
     * still just output empty, but accessing vec.front() would be undefined.
     * Hopefully this check will never be true, because empty requests should
     * be handled further up in the stack
     */
    assert(!xs.empty());
    bins ret;
    if (xs.empty()) return ret;

    ret.data.resize(points.size());

    auto snd = [](const auto& x) noexcept (true) { return x.second; };
    std::transform(points.begin(), points.end(), ret.data.begin(), snd);

    auto prev = points.front().first;
    std::size_t i = 0;
    ret.itrs.push_back(i);
    ret.keys.push_back(prev);
    for (const auto& p : points) {
        ++i;

        if (p.first == prev) continue;

        prev = p.first;
        ret.itrs.push_back(i - 1);
        ret.keys.push_back(prev);
    }

    ret.itrs.push_back(points.size());

    return ret;
}

}

#endif //SEISMIC_CLOUD_HPP
