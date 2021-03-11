#include <string>
#include <vector>

#include <fmt/format.h>

#include <oneseismic/geometry.hpp>
#include <oneseismic/messages.hpp>
#include <oneseismic/process.hpp>

namespace one {

void proc::set_fragment_shape(const std::string& shape) noexcept (false) {
    this->prefix = "src/" + shape + "/";
}

void proc::add_fragment(const std::string& id) noexcept (false) {
    if (not this->frags.empty())
        this->frags.push_back(';');

    this->frags += this->prefix;
    this->frags += id;
}

void proc::clear() noexcept (true) {
    this->prefix.clear();
    this->frags.clear();
}

void slice::init(const char* msg, int len) {
    this->clear();
    this->input.unpack(msg, msg + len);
    this->output.tiles.clear();

    assert(this->input.shape[0] > 0);
    assert(this->input.shape[1] > 0);
    assert(this->input.shape[2] > 0);

    one::FS< 3 > fragment_shape {
        std::size_t(this->input.shape[0]),
        std::size_t(this->input.shape[1]),
        std::size_t(this->input.shape[2]),
    };

    one::CS< 3 > cube_shape {
        std::size_t(this->input.shape_cube[0]),
        std::size_t(this->input.shape_cube[1]),
        std::size_t(this->input.shape_cube[2]),
    };

    this->set_fragment_shape(fmt::format("{}", fmt::join(fragment_shape, "-")));
    this->dim = one::dimension< 3 >(this->input.dim);
    this->idx = this->input.lineno;
    this->layout = fragment_shape.slice_stride(this->dim);
    this->gvt = one::gvt< 2 >(
        cube_shape.squeeze(this->dim),
        fragment_shape.squeeze(this->dim)
    );

    const auto& cs = this->gvt.cube_shape();
    this->output.shape.assign(cs.begin(), cs.end());
}

std::vector< std::string > slice::fragments() const {
    std::vector< std::string > ids;
    ids.reserve(this->input.ids.size());

    const auto prefix = fmt::format("src/{}", fmt::join(this->input.shape, "-"));
    for (const auto& id : this->input.ids) {
        ids.push_back(fmt::format("{}/{}.f32", prefix, fmt::join(id, "-")));
    }

    return ids;
}

void slice::add(int index, const char* chunk, int len) {
    one::tile t;
    const auto& triple = this->input.ids[index];
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

    this->output.tiles.push_back(t);
}

std::string slice::pack() {
    return this->output.pack();
}

}