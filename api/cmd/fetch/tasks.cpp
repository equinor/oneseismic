#include <string>
#include <vector>

#include <fmt/format.h>

#include <oneseismic/geometry.hpp>
#include <oneseismic/messages.hpp>

#include "tasks.h"

/*
 * Interface for a process. The go code calls the C api, but the implementation
 * is just a named constructor and an abstract base class. Different API
 * endpoints (slice, horizon, curtain etc) can be implemented as different
 * derived classes.
 */
class proc {
public:
    virtual void init(const char* msg, int len) = 0;

    /*
     * A vector-of-fragment IDs for this process.
     */
    virtual std::vector< std::string > fragments() const = 0;

    /*
     * The add() function is responsible for taking a fragment (size len) and
     * extract the endpoint-specified shape from it, e.g. for /slice/0 it
     * should extract the slice along the inline.
     */
    virtual void add(int index, const char* chunk, int len) = 0;
    virtual std::string pack() = 0;

    virtual ~proc() = default;

    std::string errmsg;
    std::string frags;
    std::string packed;
};

class slice : public proc, private one::slice_tiles {
public:
    void init(const char* msg, int len) override;
    std::vector< std::string > fragments() const override;
    virtual void add(int, const char* chunk, int len) override;
    std::string pack() override;

private:
    one::slice_fetch req;

    one::dimension< 3 > dim = one::dimension< 3 >(0);
    int idx;
    one::slice_layout layout;
    one::gvt< 2 > gvt;
};

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
        std::size_t(this->req.cube_shape[0]),
        std::size_t(this->req.cube_shape[1]),
        std::size_t(this->req.cube_shape[2]),
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

proc* newproc(const char* cid) try {
    const auto id = std::string(cid);
    if (id == "slice") {
        auto* p = new slice();
        return p;
    } else {
        return nullptr;
    }
} catch (...) {
    return nullptr;
}

void cleanup(proc* p) {
    if (p) delete p;
}

const char* errmsg(proc* p) {
    return p->errmsg.c_str();
}

bool init(proc* p, const void* msg, int len) {
    try {
        p->init(static_cast< const char* >(msg), len);
        return true;
    } catch (std::exception& e) {
        p->errmsg = e.what();
        return false;
    }
}

const char* fragments(proc* p) {
    p->frags = fmt::format("{}", fmt::join(p->fragments(), ";"));
    return p->frags.c_str();
}

bool add(proc* p, int index, const void* chunk, int len) {
    try {
        p->add(index, static_cast< const char* >(chunk), len);
        return true;
    } catch (std::exception& e) {
        p->errmsg = e.what();
        return false;
    }
}

packed pack(proc* p) {
    packed pd;
    try {
        p->packed = p->pack();
        pd.err = false;
        pd.size = p->packed.size();
        pd.body = p->packed.data();
    } catch (std::exception& e) {
        p->errmsg = e.what();
        pd.err = true;
    }
    return pd;
}
