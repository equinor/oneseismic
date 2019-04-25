#ifndef SEISMIC_CLOUD_HPP
#define SEISMIC_CLOUD_HPP

namespace sc {

struct point {
    std::size_t x;
    std::size_t y;
    std::size_t z;

    bool operator < (const point& rhs) const noexcept (true) {
        if (this->x < rhs.x) return true;
        if (this->y < rhs.y) return true;
        if (this->z < rhs.z) return true;
        return false;
    }

    bool operator == (const point& rhs) const noexcept (true) {
        return this->x == rhs.x
           and this->y == rhs.y
           and this->z == rhs.z;
    }
};

std::ostream& operator << (std::ostream& o, const point& rhs) {
    o << "(" << rhs.x << ", " << rhs.y << ", " << rhs.z << ")";
    return o;
}

using dimension = point;

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
