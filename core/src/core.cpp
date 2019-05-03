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
