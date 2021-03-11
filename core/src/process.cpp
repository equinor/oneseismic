#include <string>
#include <vector>

#include <fmt/format.h>

#include <oneseismic/geometry.hpp>
#include <oneseismic/messages.hpp>
#include <oneseismic/process.hpp>

namespace one {

namespace {

template < typename Seq >
one::FID< 3 > id3(const Seq& seq) noexcept (false) {
    return {
        std::size_t(seq[0]),
        std::size_t(seq[1]),
        std::size_t(seq[2]),
    };
}

}

std::unique_ptr< proc > proc::make(const std::string& kind) noexcept (false) {
    if (kind == "slice")
        return std::make_unique< slice >();
    else
        return nullptr;
}

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

const std::string& proc::fragments() const {
    return this->frags;
}

void slice::init(const char* msg, int len) {
    this->clear();
    this->input.unpack(msg, msg + len);
    this->output.tiles.resize(this->input.ids.size());

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

    for (const auto& id : this->input.ids)
        this->add_fragment(fmt::format("{}.f32", fmt::join(id, "-")));
}

void slice::add(int key, const char* chunk, int len) {
    auto& t = this->output.tiles[key];
    const auto squeezed_id = id3(this->input.ids[key]).squeeze(this->dim);
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
}

std::string slice::pack() {
    return this->output.pack();
}

}
