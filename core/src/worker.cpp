#include <cstring>
#include <string>
#include <vector>

#include <fmt/format.h>
#include <zmq.hpp>

#include <oneseismic/azure.hpp>
#include <oneseismic/geometry.hpp>
#include <oneseismic/transfer.hpp>
#include <oneseismic/tasks.hpp>

#include "log.hpp"
#include "core.pb.h"

namespace {

struct module {
    static constexpr const char* name() { return "fragment"; }
};

using log = one::basic_log< module >;

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
        one::FS< 3 > id;
        std::vector< float > data;
    };

    void oncomplete(
        const one::buffer& b,
        const one::batch& batch,
        const std::string& id,
        long http_code) override;

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
    std::vector< tile > tiles;
};

void slice::oncomplete(
        const one::buffer& b,
        const one::batch& batch,
        const std::string& id,
        long http_code) {

    if (http_code != 200) {
        /* log error, maybe abort the job */
        throw std::runtime_error(
            fmt::format("HTTP {} in slice.fragment fetch", http_code)
        );
    }

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
    auto* layout = inner->mutable_layout();
    layout->set_iterations(this->lay.iterations);
    layout->set_chunk_size(this->lay.chunk_size);
    layout->set_initial_skip(this->lay.initial_skip);
    layout->set_superstride(this->lay.superstride);
    layout->set_substride(this->lay.substride);

    oneseismic::fragment_id id;
    inner->clear_tiles();
    for (const auto& outcome : this->tiles) {
        auto* tile = inner->add_tiles();
        id.set_dim0(outcome.id[0]);
        id.set_dim1(outcome.id[1]);
        id.set_dim2(outcome.id[2]);
        *tile->mutable_id() = id;
        *tile->mutable_v() = { outcome.data.begin(), outcome.data.end() };
    }
}

void slice::prepare(const oneseismic::fetch_request& req) {
    assert(req.has_slice());
    assert(req.shape().dim0() > 0);
    assert(req.shape().dim1() > 0);
    assert(req.shape().dim2() > 0);

    one::FS< 3 > shape {
        std::size_t(req.shape().dim0()),
        std::size_t(req.shape().dim1()),
        std::size_t(req.shape().dim2()),
    };

    this->dim = one::dimension< 3 >(req.slice().dim());
    this->idx = req.slice().idx();
    this->lay = shape.slice_stride(this->dim);
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
            log::log(
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
    batch.fragment_shape = fmt::format(
        "src/{}-{}-{}",
        req.shape().dim0(),
        req.shape().dim1(),
        req.shape().dim2()
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

}

namespace one {

void fragment_task::run(
        transfer& xfer,
        zmq::socket_t& input,
        zmq::socket_t& output) {

    /*
     * TODO: Keep the protobuf instances alive, as re-using handles is a lot
     * more efficient than reallocating them every time.
     *
     * https://github.com/protocolbuffers/protobuf/blob/master/docs/performance.md
     *
     * TODO: maintain individual response instances in the action, as they can
     * reuse the same oneof every invocation
     */
    oneseismic::fetch_request request;
    oneseismic::fetch_response response;
    std::string outmsg;
    all_actions actions;

    zmq::message_t in;
    input.recv(in, zmq::recv_flags::none);
    const auto ok = request.ParseFromArray(in.data(), in.size());
    if (!ok) {
        /* log bad request, then be ready to receive new message */
        /* TODO: log the actual bytes received too */
        log::log("badly formatted protobuf message");
        return;
    }

    auto& action = actions.select(request);
    auto batch = make_batch(request);
    xfer.perform(batch, action);

    action.serialize(response);
    response.set_requestid(request.requestid());
    response.SerializeToString(&outmsg);
    zmq::message_t out(outmsg.data(), outmsg.size());
    output.send(out, zmq::send_flags::none);
}

}
