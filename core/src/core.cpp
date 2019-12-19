#include <algorithm>
#include <cassert>
#include <functional>
#include <iostream>
#include <numeric>
#include <vector>

#include <fmt/core.h>
#include <fmt/ranges.h>

#include <seismic-cloud/seismic-cloud.hpp>

namespace sc {

namespace {

template < typename Range >
auto product(const Range& r) noexcept (true)
    -> std::decay_t< decltype(*std::begin(r)) > {
    using Return = std::decay_t< decltype(*std::begin(r)) >;
    return std::accumulate(
        std::begin(r),
        std::end(r),
        1,
        std::multiplies< Return >()
    );
}

}

template < typename Base, std::size_t Dims >
std::string basic_tuple< Base, Dims >::string() const {
    const auto& self = static_cast< const base_type& >(*this);
    return fmt::format("{}", fmt::join(self, "-"));
}

template < std::size_t Dims >
cubecoords< Dims >::cubecoords(Cube_dimension cube, Frag_dimension frag) noexcept (true) :
    global_dims(cube),
    fragment_dims(frag)
{}

template < std::size_t Dims >
frag_point< Dims >
cubecoords< Dims >::to_local(Cube_point p)
const noexcept (true) {
    Frag_point tmp;
    for (int i = 0; i < Dims; ++i) {
        assert(p[i] < this->global_dims[i]);
        tmp[i] = p[i] % this->fragment_dims[i];
    }

    return tmp;
}

template < std::size_t Dims >
cube_point< Dims > cubecoords< Dims >::to_global(Fragment_id r, Frag_point p)
const noexcept (true) {
    auto cp = Cube_point();
    for (int i = 0; i < Dims; ++i) {
        cp[i] = (r[i] * this->fragment_dims[i]) + p[i];
        assert(cp[i] < this->global_dims[i]);
    }

    return cp;
}

template < std::size_t Dims >
fragment_id< Dims > cubecoords< Dims >::frag_id(Cube_point p) const noexcept (true) {
    const auto frag = this->fragment_dims;
    Fragment_id tmp;
    for (int i = 0; i < Dims; ++i) {
        assert(p[i] < this->global_dims[i]);
        tmp[i] = p[i] / frag[i];
    }

    return tmp;
}

//template < std::size_t Dims >
//cube_point< Dims > cubecoords< Dims >::from_offset(std::size_t o) const noexcept (true) {
//    assert(o < this->global_size());
//    const auto dim = this->global_dims;
//
//    return {
//        (o / (dim.y * dim.z)),
//        (o % (dim.y * dim.z)) / dim.z,
//        (o % (dim.y * dim.z)) % dim.z
//    };
//}

template < std::size_t Dims >
std::size_t cube_dimension< Dims >::slice_size(dimension< Dims > dim)
const noexcept (true) {
    auto dims = *this;
    dims[dim.v] = 1;
    return product(dims);
}

template < std::size_t Dims >
std::size_t frag_dimension< Dims >::slice_size(dimension< Dims > dim)
const noexcept (true) {
    auto dims = *this;
    dims[dim.v] = 1;
    return product(dims);
}

template < std::size_t Dims >
std::size_t cubecoords< Dims >::global_size() const noexcept (true) {
    return product(this->global_dims);
}

template < std::size_t Dims >
std::size_t cubecoords< Dims >::size(Dimension dim) const noexcept (false) {
    const auto global = this->global_dims[dim.v];
    const auto local  = this->fragment_dims[dim.v];
    return (global + (local - 1)) / local;
}

namespace {

/*
 * N-dimensional cartesian product
 *
 * This is a bit whacky. It's certainly possible to compile-time generate
 * arbitrary depths of nested loops, but it's not pretty [1], and it's even
 * worse without C++17. From a few simple tests, it looks like it generates
 * pretty much the same assembly as hand-written nested loops, but the code
 * being more complex means more opportunities for the compiler to mess up -
 * also, the code is quite hard to read.
 *
 * We assume that the number of dimensions are fairly limited, so by hand
 * implement the cartesian product. It's a bit tedious, but likely a one-time
 * job, but has the benefit of giving the compiler a much easier time
 * unrolling, and is straight-forward to understand.
 *
 * [1] https://stackoverflow.com/questions/34535795/n-dimensionally-nested-metaloops-with-templates/34601545
 */
template < typename Fn >
void cartesian_product(
    Fn&& push_back,
    const std::array< std::size_t, 1 >& begins,
    const std::array< std::size_t, 1 >& ends) {

    std::array< std::size_t, 1 > frame;
    for (frame[0] = begins[0]; frame[0] < ends[0]; ++frame[0])
        push_back(frame);
}

template < typename Fn >
void cartesian_product(
    Fn&& push_back,
    const std::array< std::size_t, 2 >& begins,
    const std::array< std::size_t, 2 >& ends) {

    std::array< std::size_t, 2 > frame;
    for (frame[0] = begins[0]; frame[0] < ends[0]; ++frame[0])
    for (frame[1] = begins[1]; frame[1] < ends[1]; ++frame[1])
        push_back(frame);
}

template < typename Fn >
void cartesian_product(
    Fn&& push_back,
    const std::array< std::size_t, 3 >& begins,
    const std::array< std::size_t, 3 >& ends) {

    std::array< std::size_t, 3 > frame;
    for (frame[0] = begins[0]; frame[0] < ends[0]; ++frame[0])
    for (frame[1] = begins[1]; frame[1] < ends[1]; ++frame[1])
    for (frame[2] = begins[2]; frame[2] < ends[2]; ++frame[2])
        push_back(frame);
}

template < typename Fn >
void cartesian_product(
    Fn&& push_back,
    const std::array< std::size_t, 4 >& begins,
    const std::array< std::size_t, 4 >& ends) {

    std::array< std::size_t, 4 > frame;
    for (frame[0] = begins[0]; frame[0] < ends[0]; ++frame[0])
    for (frame[1] = begins[1]; frame[1] < ends[1]; ++frame[1])
    for (frame[2] = begins[2]; frame[2] < ends[2]; ++frame[2])
    for (frame[3] = begins[3]; frame[3] < ends[3]; ++frame[3])
        push_back(frame);
}

template < typename Fn >
void cartesian_product(
    Fn&& push_back,
    const std::array< std::size_t, 5 >& begins,
    const std::array< std::size_t, 5 >& ends) {

    std::array< std::size_t, 5 > frame;
    for (frame[0] = begins[0]; frame[0] < ends[0]; ++frame[0])
    for (frame[1] = begins[1]; frame[1] < ends[1]; ++frame[1])
    for (frame[2] = begins[2]; frame[2] < ends[2]; ++frame[2])
    for (frame[3] = begins[3]; frame[3] < ends[3]; ++frame[3])
    for (frame[4] = begins[4]; frame[4] < ends[4]; ++frame[4])
        push_back(frame);
}

template < typename Fn, std::size_t Dims >
void cartesian_product(
    Fn&&,
    const std::array< std::size_t, Dims >&,
    const std::array< std::size_t, Dims >&) {
    /*
     * static-assert the fallthrough cases (0, unsupported dims) to give better
     * compile error messages
     */
    static_assert(
        Dims != 0,
        "0 dimensions does not make sense, probably a template value issue"
    );

    static_assert(not Dims,
        "Unsupported dimensions: to add support for more dimensions, "
        "add another overload of cartesian_product"
    );
}

}

template < std::size_t Dims >
std::vector< fragment_id< Dims > >
cubecoords< Dims >::slice(Dimension dim, std::size_t pin)
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

    if (pin >= this->size(dim))
        throw std::invalid_argument("dimension out-of-range");

    const auto begins = [&] () noexcept (true) {
        std::array< std::size_t, Dims > xs = {};
        xs[dim.v] = pin;
        return xs;
    }();

    const auto ends = [&, this] () noexcept (true) {
        std::array< std::size_t, Dims > xs;
        for (std::size_t i = 0; i < std::size_t(Dims); ++i)
            xs[i] = this->size(Dimension(i));
        xs[dim.v] = pin + 1;
        return xs;
    }();

    /* (max1 - min1) * (max2 - min2) ... */
    const auto elems = std::inner_product(
        std::begin(ends),
        std::end(ends),
        std::begin(begins),
        1,
        std::multiplies<>(),
        std::minus<>()
    );

    auto result = std::vector< Fragment_id >();
    result.reserve(elems);
    auto push_back = [&](auto val) {
        result.emplace_back(val);
    };

    cartesian_product(push_back, begins, ends);
    assert(result.size() == elems && "fragments should be exactly this many");
    return result;
}

namespace {

template < typename Point, typename Dim >
std::size_t get_offset(const Point& p, const Dim& d) noexcept (true) {
    /*
     * Equivalent to:
     *  return p.x * dim.y * dim.z
     *       + p.y * dim.z
     *       + p.z
     */
    std::array< std::size_t, Dim::dimensions > dim_product;
    dim_product.back() = 1;

    std::partial_sum(
        std::rbegin(d),
        std::rend(d) - 1,
        std::rbegin(dim_product) + 1,
        std::multiplies<>()
    );

    return std::inner_product(
        std::begin(p),
        std::end(p),
        std::begin(dim_product),
        0
    );
}

}

template < std::size_t Dims >
std::size_t cube_dimension< Dims >::to_offset(cube_point< Dims > p)
const noexcept (true) {
    return get_offset(p, *this);
}

template < std::size_t Dims >
std::size_t cube_dimension< Dims >::to_offset(fragment_id< Dims > p)
const noexcept (true) {
    return get_offset(p, *this);
}

template < std::size_t Dims >
std::size_t frag_dimension< Dims >::to_offset(frag_point< Dims > p)
const noexcept (true) {
    return get_offset(p, *this);
}

template class cubecoords     < 3 >;
template class cube_dimension < 3 >;
template class frag_dimension < 3 >;
template class fragment_id    < 3 >;
template class basic_tuple< fragment_id< 3 >, 3 >;

}
