#include <algorithm>
#include <cassert>
#include <iostream>
#include <vector>

#include <seismic-cloud/seismic-cloud.hpp>

namespace sc {

std::ostream& operator << (std::ostream& o, const point& rhs) {
    return o << "(" << rhs.x << ", " << rhs.y << ", " << rhs.z << ")";
}

std::ostream& operator << (std::ostream& o, const dimension& rhs) {
    return o << "(" << rhs.x << ", " << rhs.y << ", " << rhs.z << ")";
}

cube_point fragment_id::operator + (frag_point p) const noexcept (true) {
    return {
        this->x + p.x,
        this->y + p.y,
        this->z + p.z,
    };
}

cubecoords::cubecoords(cube_dim cube, frag_dim frag) noexcept (true) :
    global_dims(cube),
    fragment_dims(frag)
{}

frag_point cubecoords::to_local(cube_point p) const noexcept (true) {
    assert(p.x < global_dims.x);
    assert(p.y < global_dims.y);
    assert(p.z < global_dims.z);

    return {
        p.x % this->fragment_dims.x,
        p.y % this->fragment_dims.y,
        p.z % this->fragment_dims.z,
    };
}

cube_point cubecoords::to_global(fragment_id r, frag_point p)
const noexcept (true)
{
    const auto cp = cube_point {
        (r.x * this->fragment_dims.x) + p.x,
        (r.y * this->fragment_dims.y) + p.y,
        (r.z * this->fragment_dims.z) + p.z,
    };
    assert(cp.x < this->global_dims.x);
    assert(cp.y < this->global_dims.y);
    assert(cp.z < this->global_dims.z);
    return cp;
}

fragment_id cubecoords::frag_id(cube_point p) const noexcept (true) {
    assert(p.x < global_dims.x);
    assert(p.y < global_dims.y);
    assert(p.z < global_dims.z);

    const auto frag = this->fragment_dims;
    return {
        p.x / frag.x,
        p.y / frag.y,
        p.z / frag.z,
    };
}

cube_point cubecoords::from_offset(std::size_t o) const noexcept (true) {
    assert(o < this->global_size());
    const auto dim = this->global_dims;
    return {
        (o / (dim.y * dim.z)),
        (o % (dim.y * dim.z)) / dim.z,
        (o % (dim.y * dim.z)) % dim.z
    };
}

std::size_t cubecoords::global_size() const noexcept (true) {
    return this->global_dims.x
         * this->global_dims.y
         * this->global_dims.z
    ;
}

namespace {

template < typename Point, typename Dim >
std::size_t get_offset(Point p, Dim dim) noexcept (true) {
    return p.x * dim.y * dim.z
         + p.y * dim.z
         + p.z
    ;
}

}

std::size_t cube_dim::to_offset(cube_point p) const noexcept (true) {
    return get_offset(p, *this);
}
std::size_t cube_dim::to_offset(fragment_id p) const noexcept (true) {
    return get_offset(p, *this);
}
std::size_t frag_dim::to_offset(frag_point p) const noexcept (true) {
    return get_offset(p, *this);
}


point global_to_local(point global, dimension fragment_size) noexcept (true) {
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

point global_to_root(point global, dimension fragment_size) noexcept (true) {
    return {
        (global.x / fragment_size.x) * fragment_size.x,
        (global.y / fragment_size.y) * fragment_size.y,
        (global.z / fragment_size.z) * fragment_size.z,
    };
}

point offset_to_point(std::size_t offset, dimension dim) noexcept (true) {
    return {
        (offset / (dim.y * dim.z)),
        (offset % (dim.y * dim.z)) / dim.z,
        (offset % (dim.y * dim.z)) % dim.z
    };
}

std::size_t point_to_offset(point p, dimension dim) noexcept (true) {
    return p.x * dim.y * dim.z
         + p.y * dim.z
         + p.z;
}

std::size_t local_to_global(std::size_t local,
                             dimension fragment_size,
                             dimension cube_size,
                             point root) noexcept (true) {
    const auto local_point = offset_to_point(local, fragment_size);
    const auto global = local_to_global(local_point, root);
    return point_to_offset(global, cube_size);
}

bins bin(dimension fragment_size,
         dimension cube_size,
         const std::vector< point >& xs) noexcept (false) {

    using key = std::pair< point, std::size_t >;
    auto points = std::vector< key >(xs.size());

    auto fragment_id = [fragment_size](const auto& p) noexcept (true) {
        const auto root  = global_to_root(p, fragment_size);
        const auto local = global_to_local(p, fragment_size);
        const auto pos   = point_to_offset(local, fragment_size);
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
