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
