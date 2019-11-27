#include <algorithm>
#include <cassert>
#include <iostream>
#include <vector>

#include <seismic-cloud/seismic-cloud.hpp>

namespace sc {

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

std::size_t cubecoords::size(dimension dim) const noexcept (false) {
    auto segments = [](std::size_t global, std::size_t local) noexcept(true) {
        // integer division galore!
        return (global + (local - 1)) / local;
    };
    switch (dim.v) {
        case 0: return segments(this->global_dims.x, this->fragment_dims.x);
        case 1: return segments(this->global_dims.y, this->fragment_dims.y);
        case 2: return segments(this->global_dims.z, this->fragment_dims.z);
        default:
            throw std::invalid_argument(
                "unsupported dimension " + std::to_string(dim.v)
            );
    }
}

std::vector< fragment_id >
cubecoords::slice(dimension dim, std::size_t pin)
noexcept (false) {
    /*
     * A fairly straight-forward (although a bit slower than it had to) way of
     * getting the fragment IDs that slice a cube. Not quite as fast as it
     * could be, and could be made into an iterator too, but good enough for
     * now due to its simplicity.
     *
     * The problem really boils down the cartesian product of [0, fragments) for
     * all dimensions, except the pinned one (range of 1).
     */

    if (pin >= this->size(dimension{dim.v}))
        throw std::invalid_argument("dimension out-of-range");

    const auto minx = dim.v == 0 ? pin : 0;
    const auto miny = dim.v == 1 ? pin : 0;
    const auto minz = dim.v == 2 ? pin : 0;

    const auto maxx = dim.v == 0 ? (pin + 1) : this->size(dimension{0});
    const auto maxy = dim.v == 1 ? (pin + 1) : this->size(dimension{1});
    const auto maxz = dim.v == 2 ? (pin + 1) : this->size(dimension{2});

    assert(maxx <= this->size(dimension{0}));
    assert(maxy <= this->size(dimension{1}));
    assert(maxz <= this->size(dimension{2}));

    const auto elems = (maxx - minx) * (maxy - miny) * (maxz - minz);
    auto result = std::vector< fragment_id >();
    result.reserve(elems);

    for (auto x = minx; x < maxx; ++x)
    for (auto y = miny; y < maxy; ++y)
    for (auto z = minz; z < maxz; ++z)
        result.emplace_back(x, y, z);

    assert(result.size() == elems && "fragments should be exactly this many");
    return result;
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

}
