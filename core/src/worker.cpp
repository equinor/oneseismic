#include <cassert>
#include <cstring>
#include <exception>
#include <string>
#include <vector>

#include <fmt/format.h>
#include <spdlog/spdlog.h>
#include <zmq.hpp>
#include <zmq_addon.hpp>

#include <oneseismic/azure.hpp>
#include <oneseismic/geometry.hpp>
#include <oneseismic/transfer.hpp>
#include <oneseismic/tasks.hpp>

#include "core.pb.h"

namespace {

/*
 * Every request type (slice, trace, fragment) must know how to transform
 * itself into the correct message for the wire
 */
class wire {
public:
    virtual ~wire() = default;
    virtual void serialize(oneseismic::fetch_response&) const = 0;
    virtual void prepare(const oneseismic::fetch_request&) = 0;
};

/*
 * Union of transfer configuration and the response message serializer.
 */
class action : public one::transfer_configuration, public wire {};

class slice : public action {
public:
    struct tile {
        one::FID< 3 > id;
        std::vector< float > data;
    };

    void oncomplete(
        const one::buffer& b,
        const one::batch& batch,
        const std::string& id) override;

    void serialize(oneseismic::fetch_response&) const override;
    void prepare(const oneseismic::fetch_request& req) override;

private:
    /*
     * There's no default constructor for dimension, so just ghetto-default it
     * to 0
     */
    one::dimension< 3 > dim = one::dimension< 3 >(0);
    int idx;
    one::slice_layout lay;
    one::gvt< 3 > gvt;
    std::vector< tile > tiles;
};

void slice::oncomplete(
        const one::buffer& b,
        const one::batch& batch,
        const std::string& id) {

    auto t = tile();
    t.data.resize(this->lay.iterations * this->lay.chunk_size);
    auto* dst = reinterpret_cast< std::uint8_t* >(t.data.data());
    auto* src = b.data() + this->lay.initial_skip * this->idx * sizeof(float);

    for (auto i = 0; i < this->lay.iterations; ++i) {
        std::memcpy(dst, src, this->lay.chunk_size * sizeof(float));
        dst += this->lay.substride * sizeof(float);
        src += this->lay.superstride * sizeof(float);
    }

    std::sscanf(id.c_str(),
            "%d-%d-%d",
            &t.id[0],
            &t.id[1],
            &t.id[2]);
    this->tiles.push_back(t);
}

void slice::serialize(oneseismic::fetch_response& res) const {
    auto* inner = res.mutable_slice();

    oneseismic::fragment_id id;
    inner->clear_tiles();
    for (const auto& outcome : this->tiles) {
        auto* tile = inner->add_tiles();

        auto flattened_id = outcome.id;
        flattened_id[this->dim] = 0;
        const auto layout = this->gvt.slice_stride(this->dim, flattened_id);

        auto* l = tile->mutable_layout();

        l->set_iterations(layout.iterations);
        l->set_chunk_size(layout.chunk_size);
        l->set_initial_skip(layout.initial_skip);
        l->set_superstride(layout.superstride);
        l->set_substride(layout.substride);

        *tile->mutable_v() = { outcome.data.begin(), outcome.data.end() };
    }
}

void slice::prepare(const oneseismic::fetch_request& req) {
    assert(req.has_slice());
    assert(req.fragment_shape().dim0() > 0);
    assert(req.fragment_shape().dim1() > 0);
    assert(req.fragment_shape().dim2() > 0);

    one::FS< 3 > fragment_shape {
        std::size_t(req.fragment_shape().dim0()),
        std::size_t(req.fragment_shape().dim1()),
        std::size_t(req.fragment_shape().dim2()),
    };

    one::CS< 3 > cube_shape {
        std::size_t(req.cube_shape().dim0()),
        std::size_t(req.cube_shape().dim1()),
        std::size_t(req.cube_shape().dim2()),
    };
    cube_shape[req.slice().dim()] = 1;

    this->dim = one::dimension< 3 >(req.slice().dim());
    this->idx = req.slice().idx();
    this->lay = fragment_shape.slice_stride(this->dim);
    this->gvt = one::gvt< 3 >(cube_shape, fragment_shape);
}

class all_actions {
public:
    action& select(const oneseismic::fetch_request&) noexcept (false);

private:
    slice s;
};

action& all_actions::select(const oneseismic::fetch_request& req)
noexcept (false) {
    using msg = oneseismic::fetch_request;

    switch (req.function_case()) {
        case msg::kSlice:
            this->s.prepare(req);
            return this->s;

        default:
            spdlog::debug(
                "{} - malformed input, bad request variant (oneof)",
                req.requestid()
            );
            throw std::runtime_error("bad oneof");
    }
}

one::batch make_batch(const oneseismic::fetch_request& req) noexcept (false) {
    one::batch batch;
    batch.root = req.root();
    batch.guid = req.guid();
    batch.storage_endpoint = req.storage_endpoint();
    batch.fragment_shape = fmt::format(
        "src/{}-{}-{}",
        req.fragment_shape().dim0(),
        req.fragment_shape().dim1(),
        req.fragment_shape().dim2()
    );

    for (const auto& id : req.ids()) {
        batch.fragment_ids.push_back(fmt::format(
            "{}-{}-{}",
            id.dim0(),
            id.dim1(),
            id.dim2()
        ));
    }

    return batch;
}

struct bad_message : std::exception {};

class fetch_request : public oneseismic::fetch_request {
public:
    void parse(const zmq::message_t&) noexcept (false);
};

void fetch_request::parse(const zmq::message_t& msg) noexcept (false) {
    const auto ok = this->ParseFromArray(msg.data(), msg.size());
    if (!ok) throw bad_message();
}

class fetch_response : public oneseismic::fetch_response {
public:
    const std::string& serialize() noexcept (false);

private:
    std::string serialized;
};

const std::string& fetch_response::serialize() noexcept (false) {
    this->SerializeToString(&this->serialized);
    return this->serialized;
}

}

namespace one {

class fragment_task::impl {
public:
    /*
     * Transition into "processing" mode - this should be the first method
     * called whenever a message is received from the queue and a job is about
     * to start.
     */
    void start_processing(zmq::multipart_t& task);

    zmq::multipart_t failure(const std::string& key) noexcept (false);

    std::string pid;
    fetch_request query;
    fetch_response result;
    all_actions actions;
};

void fragment_task::impl::start_processing(zmq::multipart_t& task) {
    /*
     * Currently, the only thing this function does is parse and set the pid
     * which is not actually cleared between invocations (so calling methods
     * out-of-order is not detected). This can certainly change in the future
     * though, and gets the clunky read-bytes-from-message out of the way in
     * the body.
     */

    /*
     * The job argument is conceptually const, but can't be since the
     * zmq::multipart_t is not marked as such (even though it is)
     */
    const auto& pid = task.front();
    this->pid.assign(
            static_cast< const char* >(pid.data()),
            pid.size()
    );
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

    /*
     * TODO: Keep the protobuf instances alive, as re-using handles is a lot
     * more efficient than reallocating them every time.
     *
     * https://github.com/protocolbuffers/protobuf/blob/master/docs/performance.md
     *
     * TODO: maintain individual response instances in the action, as they can
     * reuse the same oneof every invocation
     */
    zmq::multipart_t envelope(input);
    this->p->start_processing(envelope);

    const auto& body = envelope.remove();
    auto& query = this->p->query;
    query.parse(body);

    auto& action = this->p->actions.select(query);
    auto batch = make_batch(query);
    xfer.perform(batch, action);

    auto& result = this->p->result;
    result.set_requestid(query.requestid());
    action.serialize(result);

    auto msg = envelope.clone();
    msg.addstr(result.serialize());
    msg.send(output);

    /*
     * TODO: catch other network related errors that should not bring down the
     * process (currently will because of unhandled exceptions)
     */
} catch (const bad_message&) {
    /* TODO: log the actual bytes received too */
    /* TODO: log sender */
    spdlog::error(
            "{} badly formatted protobuf message",
            this->p->pid
    );
    this->p->failure("bad-message").send(failure);
} catch (const notfound& e) {
    spdlog::warn(
            "{} fragment not found: '{}'",
            this->p->pid,
            e.what()
    );
    this->p->failure("fragment-not-found").send(failure);
}

fragment_task::fragment_task() : p(new impl()) {}
fragment_task::~fragment_task() = default;

}
