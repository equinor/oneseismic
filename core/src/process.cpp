#include <ciso646>
#include <numeric>
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

one::gvt< 3 > gvt3(const one::basic_task& task) {
    assert(task.shape[0] > 0);
    assert(task.shape[1] > 0);
    assert(task.shape[2] > 0);

    one::FS< 3 > fs {
        std::size_t(task.shape[0]),
        std::size_t(task.shape[1]),
        std::size_t(task.shape[2]),
    };

    one::CS< 3 > cs {
        std::size_t(task.shape_cube[0]),
        std::size_t(task.shape_cube[1]),
        std::size_t(task.shape_cube[2]),
    };

    return { cs, fs };
}

class slice : public proc {
public:
    void init(const char* msg, int len) override;
    virtual void add(int, const char* chunk, int len) override;
    std::string pack() override;

private:
    one::slice_task  input;
    one::slice_tiles output;

    one::dimension< 3 > dim = one::dimension< 3 >(0);
    int idx;
    one::slice_layout layout;
    one::gvt< 2 > gvt;
};

class curtain : public proc {
public:
    void init(const char* msg, int len) override;
    virtual void add(int, const char* chunk, int len) override;
    std::string pack() override;

private:
    one::curtain_task   input;
    one::curtain_bundle output;
    one::gvt< 3 >       gvt;
    std::vector< int >  traceindex;
};

}

std::unique_ptr< proc > proc::make(const std::string& kind) noexcept (false) {
    if (kind == "slice")
        return std::make_unique< slice >();
    if (kind == "curtain")
        return std::make_unique< curtain >();
    else
        return nullptr;
}

void proc::set_prefix(const basic_task& task) noexcept (false) {
    fmt::format_to(
        std::back_inserter(this->prefix),
        "{}/{}/",
        task.prefix,
        fmt::join(task.shape, "-")
    );
}

void proc::add_fragment(
        const std::string& id,
        const std::string& ext)
noexcept (false) {
    if (not this->frags.empty())
        this->frags.push_back(';');

    this->frags += this->prefix;
    this->frags += id;

    if (not ext.empty()) {
        this->frags += '.';
        this->frags += ext;
    }
}

void proc::clear() noexcept (true) {
    this->prefix.clear();
    this->frags.clear();
}

const std::string& proc::fragments() const {
    return this->frags;
}

namespace {

void slice::init(const char* msg, int len) {
    this->clear();
    this->input.unpack(msg, msg + len);
    this->output.tiles.resize(this->input.ids.size());
    this->output.attr = this->input.attribute;

    const auto g3 = gvt3(this->input);
    const auto& fragment_shape = g3.fragment_shape();

    this->set_prefix(this->input);
    this->dim = g3.mkdim(this->input.dim);
    this->idx = this->input.idx;
    this->layout = fragment_shape.slice_stride(this->dim);
    this->gvt = g3.squeeze(this->dim);

    for (const auto& id : this->input.ids) {
        const auto name = fmt::format("{}", fmt::join(id, "-"));
        this->add_fragment(name, this->input.ext);
    }
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

void curtain::init(const char* msg, int len) {
    this->clear();
    this->input.unpack(msg, msg + len);
    this->gvt = gvt3(this->input);
    this->set_prefix(this->input);

    const auto& ids = this->input.ids;

    for (const auto& single : ids) {
        const auto name = fmt::format("{}", fmt::join(single.id, "-"));
        this->add_fragment(name, this->input.ext);
    }

    const auto zsize = [this](const auto& id) noexcept {
        /*
         * Compute the size, in floats, of a block of (sub)traces. A block is
         * made up of all the trace segments in one (i,j,k) fragment, and maybe
         * padded.
         */
        const auto zdim     = this->gvt.mkdim(gvt.ndims - 1);
        const auto zheight  = this->gvt.fragment_shape()[zdim];
        const auto fid      = id3(id.id);
        const auto zreal    = zheight - this->gvt.padding(fid, zdim);
        return zreal * id.coordinates.size();
    };

    /*
     * The curtain call uses an auxillary table to figure out where to write
     * traces as they are extracted from fragments. This simplifies the
     * algorithm greatly, and means that add() can be called in parallel (!)[1]
     * since every add() should write to a different segment of the output.
     *
     * The traceindex [k] contains the *starting position* of the add(k)
     * output. This is assigned by scanning size of the traces/coordinate-list
     * to extract per fragment.
     *
     * By making traceindex.size() = id.size() + 1, we can avoid a bunch of
     * special cases, and the # of traces in total can be read at
     * traceindex.back().
     *
     * [1] as long as the key-argument to add is distinct
     */
    this->traceindex.resize(ids.size() + 1);
    this->traceindex[0] = 0;
    std::transform(
        ids.begin(),
        ids.end(),
        this->traceindex.begin() + 1,
        zsize
    );
    std::partial_sum(
        this->traceindex.begin(),
        this->traceindex.end(),
        this->traceindex.begin()
    );

    const auto csize = [](const auto& x) noexcept {
        return x.coordinates.size();
    };

    const auto zdim    = this->gvt.mkdim(gvt.ndims - 1);
    const auto zheight = this->gvt.fragment_shape()[zdim];
    const auto zmax    = this->gvt.nsamples(zdim);
    this->output.attr = this->input.attribute;
    this->output.size = ids.size();
    this->output.zlength = zmax;
    this->output.major.reserve(this->output.size * 2);
    this->output.minor.reserve(this->output.size * 2);
    this->output.values.resize(this->traceindex.back());
    for (const auto& id : ids) {
        const auto zfst = id.id[zdim] * zheight;
        const auto zlst = std::min(zfst + zheight, zmax);
        this->output.major.push_back(id.offset);
        this->output.major.push_back(id.offset + csize(id));
        this->output.minor.push_back(zfst);
        this->output.minor.push_back(zlst);
    }
}

void curtain::add(int key, const char* chunk, int len) {
    const auto& id = this->input.ids[key];

    const auto fid = id3(id.id);
    const auto zdim    = this->gvt.mkdim(gvt.ndims - 1);
    const auto zpad    = this->gvt.padding(fid, zdim);
    const auto zheight = this->gvt.fragment_shape()[zdim] - zpad;

    auto* dst = this->output.values.data() +  this->traceindex[key];
    for (const auto& coord : id.coordinates) {
        const auto fp = one::FP< 3 > {
            std::size_t(coord[0]),
            std::size_t(coord[1]),
            std::size_t(0),
        };
        const auto off = this->gvt.fragment_shape().to_offset(fp);
        const auto src = chunk + off * sizeof(float);
        std::memcpy(dst, src, sizeof(float) * zheight);
        dst += zheight;
    }
}

std::string curtain::pack() {
    return this->output.pack();
}

}

}
