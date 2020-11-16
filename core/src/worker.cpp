#include <cassert>
#include <cstring>
#include <exception>
#include <string>
#include <vector>

#include <fmt/format.h>
#include <spdlog/spdlog.h>
#include <zmq_addon.hpp>
#include <zmq.hpp>

#include <oneseismic/azure.hpp>
#include <oneseismic/geometry.hpp>
#include <oneseismic/messages.hpp>
#include <oneseismic/tasks.hpp>
#include <oneseismic/transfer.hpp>

namespace {

class slice : public one::transfer_configuration, public one::slice_tiles {
public:
    void prepare(const one::slice_fetch&) noexcept (false);

    void oncomplete(
        const one::buffer& b,
        const one::batch& batch,
        const std::string& id) override;
    void clear();

private:
    /*
     * There's no default constructor for dimension, so just ghetto-default it
     * to 0
     */
    one::dimension< 3 > dim = one::dimension< 3 >(0);
    int idx;
    one::slice_layout layout;
    one::gvt< 2 > gvt;
};

one::FID< 3 > id3_from_string(const std::string& id) {
    int i, j, k;
    const auto matched = std::sscanf(id.c_str(), "%d-%d-%d", &i, &j, &k);
    if (matched != 3) {
        const auto msg = "expected FID<3> in format, %d-%d-%d, was {}";
        throw std::invalid_argument(fmt::format(msg, id));
    }
    return { std::size_t(i), std::size_t(j), std::size_t(k) };
}

void slice::oncomplete(
        const one::buffer& b,
        const one::batch& batch,
        const std::string& id) {

    one::tile t;
    const auto squeezed_id = id3_from_string(id).squeeze(this->dim);
    const auto tile_layout = this->gvt.injection_stride(squeezed_id);
    t.iterations   = tile_layout.iterations;
    t.chunk_size   = tile_layout.chunk_size;
    t.initial_skip = tile_layout.initial_skip;
    t.superstride  = tile_layout.superstride;
    t.substride    = tile_layout.substride;

    t.v.resize(this->layout.iterations * this->layout.chunk_size);
    auto* dst = reinterpret_cast< std::uint8_t* >(t.v.data());
    auto* src = b.data() + this->layout.initial_skip * this->idx * sizeof(float);
    for (auto i = 0; i < this->layout.iterations; ++i) {
        std::memcpy(dst, src, this->layout.chunk_size * sizeof(float));
        dst += this->layout.substride * sizeof(float);
        src += this->layout.superstride * sizeof(float);
    }

    this->tiles.push_back(t);
}

void slice::prepare(const one::slice_fetch& req) {
    assert(req.shape[0] > 0);
    assert(req.shape[1] > 0);
    assert(req.shape[2] > 0);

    one::FS< 3 > fragment_shape {
        std::size_t(req.shape[0]),
        std::size_t(req.shape[1]),
        std::size_t(req.shape[2]),
    };

    one::CS< 3 > cube_shape {
        std::size_t(req.cube_shape[0]),
        std::size_t(req.cube_shape[1]),
        std::size_t(req.cube_shape[2]),
    };

    this->dim = one::dimension< 3 >(req.dim);
    this->idx = req.lineno;
    this->layout = fragment_shape.slice_stride(this->dim);
    this->gvt = one::gvt< 2 >(
        cube_shape.squeeze(this->dim),
        fragment_shape.squeeze(this->dim)
    );

    const auto& cs = this->gvt.cube_shape();
    this->shape.assign(cs.begin(), cs.end());
}

void slice::clear() {
  this->tiles.clear();
}

one::batch make_batch(const one::slice_fetch& req) noexcept (false) {
    one::batch batch;
    batch.guid = req.guid;
    batch.storage_endpoint = req.storage_endpoint;
    batch.auth = fmt::format("Bearer {}", req.token);
    batch.fragment_shape = fmt::format("src/{}", fmt::join(req.shape, "-"));

    for (const auto& id : req.ids) {
        const auto s = fmt::format("{}", fmt::join(id, "-"));
        batch.fragment_ids.push_back(s);
    }

    return batch;
}

}

namespace one {

/**
 * Cache instances - for rationale, see manifest_task::impl
 */
class fragment_task::impl {
public:
    void parse(const zmq::multipart_t& task);

    zmq::multipart_t failure(const std::string& key) noexcept (false);

    std::string pid;
    std::string part;
    slice_fetch query;
    slice       result;
    working_storage storage;
};

void fragment_task::connect_working_storage(const std::string& addr) {
    this->p->storage.connect(addr);
}

void fragment_task::impl::parse(const zmq::multipart_t& task) {
    assert(task.size() == 3);
    const auto& pid  = task[0];
    const auto& part = task[1];
    const auto& body = task[2];

    this->pid.assign( static_cast< const char* >(pid.data()),  pid.size());
    this->part.assign(static_cast< const char* >(part.data()), part.size());
    const auto* fst = static_cast< const char* >(body.data());
    const auto* lst = fst + body.size();
    this->query.unpack(fst, lst);
}

zmq::multipart_t fragment_task::impl::failure(const std::string& key)
noexcept (false) {
    zmq::multipart_t msg;
    msg.addstr(this->pid);
    msg.addstr(key);
    return msg;
}

void fragment_task::run(
        transfer& xfer,
        zmq::socket_t& input,
        zmq::socket_t& output,
        zmq::socket_t& failure) try {

    zmq::multipart_t process(input);
    this->p->parse(process);

    const auto& query = this->p->query;
    auto& action = this->p->result;
    action.prepare(query);
    auto batch = make_batch(query);
    xfer.perform(batch, action);

    const auto id = fmt::format("{}:{}", this->p->pid, this->p->part);
    const auto packed = action.pack();
    this->p->storage.put(id, packed);
    action.clear();
    /*
     * TODO: catch other network related errors that should not bring down the
     * process (currently will because of unhandled exceptions)
     */
} catch (const bad_message&) {
    /* TODO: log the actual bytes received too */
    /* TODO: log sender */
    spdlog::error(
            "pid={}, badly formatted protobuf message",
            this->p->pid
    );
    this->p->failure("bad-message").send(failure);
} catch (const notfound& e) {
    spdlog::warn(
            "pid={}, fragment not found: '{}'",
            this->p->pid,
            e.what()
    );
    this->p->failure("fragment-not-found").send(failure);
} catch (const unauthorized&) {
    /*
     * TODO: log the headers?
     * TODO: log manifest url?
     */
    spdlog::info("pid={}, not authorized", this->p->pid);
    this->p->failure("fragment-not-authorized").send(failure);
} catch (const storage_error& e) {
    spdlog::warn("pid={}, storage error: {}", this->p->pid, e.what());
    this->p->failure("storage-error").send(failure);
}

fragment_task::fragment_task() : p(new impl()) {}
fragment_task::~fragment_task() = default;

}
