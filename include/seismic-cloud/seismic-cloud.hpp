#ifndef SEISMIC_CLOUD_HPP
#define SEISMIC_CLOUD_HPP

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

    bool operator == (const T& rhs) const noexcept (true) {
        const auto& self = *static_cast< const T* >(this);
        return self.x == rhs.x
           and self.y == rhs.y
           and self.z == rhs.z
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

point local_to_global( point local, point root ) {
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

}

#endif //SEISMIC_CLOUD_HPP
