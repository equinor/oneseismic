#include <string>
#include <vector>

#include <fmt/format.h>

#include <oneseismic/geometry.hpp>
#include <oneseismic/messages.hpp>
#include <oneseismic/process.hpp>

namespace one {

void slice::init(const char* msg, int len) {
    this->req.unpack(msg, msg + len);
    this->tiles.clear();

    assert(this->req.shape[0] > 0);
    assert(this->req.shape[1] > 0);
    assert(this->req.shape[2] > 0);

    one::FS< 3 > fragment_shape {
        std::size_t(this->req.shape[0]),
        std::size_t(this->req.shape[1]),
        std::size_t(this->req.shape[2]),
    };

    one::CS< 3 > cube_shape {
        std::size_t(this->req.shape_cube[0]),
        std::size_t(this->req.shape_cube[1]),
        std::size_t(this->req.shape_cube[2]),
    };

    this->dim = one::dimension< 3 >(this->req.dim);
    this->idx = this->req.lineno;
    this->layout = fragment_shape.slice_stride(this->dim);
    this->gvt = one::gvt< 2 >(
        cube_shape.squeeze(this->dim),
        fragment_shape.squeeze(this->dim)
    );

    const auto& cs = this->gvt.cube_shape();
    this->shape.assign(cs.begin(), cs.end());
}

std::vector< std::string > slice::fragments() const {
    std::vector< std::string > ids;
    ids.reserve(this->req.ids.size());

    const auto prefix = fmt::format("src/{}", fmt::join(this->req.shape, "-"));
    for (const auto& id : this->req.ids) {
        ids.push_back(fmt::format("{}/{}.f32", prefix, fmt::join(id, "-")));
    }

    return ids;
}

void slice::add(int index, const char* chunk, int len) {
    one::tile t;
    const auto& triple = this->req.ids[index];
    const auto id3 = one::FID< 3 > {
        std::size_t(triple[0]),
        std::size_t(triple[1]),
        std::size_t(triple[2]),
    };
    const auto squeezed_id = id3.squeeze(this->dim);
    const auto tile_layout = this->gvt.injection_stride(squeezed_id);
    t.iterations   = tile_layout.iterations;
    t.chunk_size   = tile_layout.chunk_size;
    t.initial_skip = tile_layout.initial_skip;
    t.superstride  = tile_layout.superstride;
    t.substride    = tile_layout.substride;

    t.v.resize(this->layout.iterations * this->layout.chunk_size);
    auto* dst = reinterpret_cast< std::uint8_t* >(t.v.data());
    auto* src = chunk + this->layout.initial_skip * this->idx * sizeof(float);
    for (auto i = 0; i < this->layout.iterations; ++i) {
        std::memcpy(dst, src, this->layout.chunk_size * sizeof(float));
        dst += this->layout.substride * sizeof(float);
        src += this->layout.superstride * sizeof(float);
    }

    this->tiles.push_back(t);
}

std::string slice::pack() {
    return this->one::slice_tiles::pack();
}

}
