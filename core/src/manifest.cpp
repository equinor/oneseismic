#include <cassert>
#include <chrono>
#include <string>

#include <fmt/format.h>
#include <nlohmann/json.hpp>
#include <spdlog/spdlog.h>
#include <zmq.hpp>
#include <zmq_addon.hpp>

#include <oneseismic/tasks.hpp>
#include <oneseismic/azure.hpp>
#include <oneseismic/transfer.hpp>
#include <oneseismic/geometry.hpp>

#include "core.pb.h"

namespace one {

namespace {

one::gvt< 3 > geometry(
        const nlohmann::json& dimensions,
        const oneseismic::fragment_shape& shape) noexcept (false) {
    return one::gvt< 3 > {
        { dimensions[0].size(),
          dimensions[1].size(),
          dimensions[2].size(), },
        { std::size_t(shape.dim0()),
          std::size_t(shape.dim1()),
          std::size_t(shape.dim2()), }
    };
}

struct manifest_cfg : public one::transfer_configuration {
    void oncomplete(
            const one::buffer& buffer,
            const one::batch&,
            const std::string&) override {
        /* TODO: in debug, store the string too? */
        this->doc = buffer;
    }

    one::buffer doc;
};

using clock = std::chrono::steady_clock;

bool set_slice_request(
        oneseismic::fetch_request& req,
        const oneseismic::api_request& api,
        const nlohmann::json& manifest) noexcept (false) {
    assert(api.has_slice());

    const auto dim = api.slice().dim();
    const auto lineno = api.slice().lineno();

    /*
     * TODO:
     * faster to not make vector, but rather parse-and-compare individual
     * integers?
     */
    const auto& manifest_dimensions = manifest["dimensions"];
    const auto index = manifest_dimensions[dim].get< std::vector< int > >();
    auto itr = std::find(index.begin(), index.end(), lineno);
    if (itr == index.end()) {
        spdlog::info(
                "{}: lineno (= {}) not found in index",
                api.requestid(),
                lineno
        );
        return false;
    }

    const auto pin = std::distance(index.begin(), itr);
    auto gvt = geometry(manifest_dimensions, api.shape());

    const auto ids = gvt.slice(
        one::dimension< 3 >(dim),
        pin
    );

    auto* cs = req.mutable_cube_shape();
    cs->set_dim0(manifest_dimensions[0].size());
    cs->set_dim1(manifest_dimensions[1].size());
    cs->set_dim2(manifest_dimensions[2].size());

    for (const auto& id : ids) {
        auto* c = req.add_ids();
        c->set_dim0(id[0]);
        c->set_dim1(id[1]);
        c->set_dim2(id[2]);
    }

    auto* slice = req.mutable_slice();
    slice->set_dim(dim);
    slice->set_idx(pin % gvt.fragment_shape()[dim]);
    return true;
}

}

void manifest_task::run(
        transfer& xfer,
        zmq::socket_t& input,
        zmq::socket_t& output,
        zmq::socket_t& fail) {

    using timepoint = std::chrono::time_point< clock >;
    struct timer {
        timepoint start;
        timepoint recv;
        timepoint parse;
        timepoint transfer;
        timepoint json;
        timepoint serialize;
        timepoint send;
    } timer;

    /*
     * These should be cached probably, as there are performance implications
     * to not reusing them. Exposing the generated code in headers is pretty
     * bad though, so something clever needs to be done.
     */
    std::string msg;
    oneseismic::api_request apirequest;
    oneseismic::fetch_request fetchrequest;

    /*
     * Virtually any failure here means socket restart, with the exception
     * of EINTR, which means interrupted and rather a process tear down.
     *
     * Sockets are configured from the outside, so regardless it's time to exit.
     */
    timer.start = clock::now();

    zmq::multipart_t multi(input);
    const zmq::message_t& req = multi.back();
    timer.recv = clock::now();
    const auto ok = apirequest.ParseFromArray(req.data(), req.size());
    if (!ok) {
        /* log bad request, then be ready to receive new message */
        /* TODO: log the actual bytes received too */
        spdlog::warn("badly formatted protobuf message");
        return;
    }

    timer.parse = clock::now();
    const auto& requestid = apirequest.requestid();

    /* fetch manifest */
    one::batch batch;
    batch.root = apirequest.root();
    batch.storage_endpoint = apirequest.storage_endpoint();
    batch.guid = apirequest.guid();
    batch.fragment_ids.resize(1);
    manifest_cfg cfg;
    try {
        xfer.perform(batch, cfg);
    } catch (const notfound& e) {
        spdlog::info("{} not found: '{}'", batch.guid, e.what());

        const auto signal = fmt::format("notfound: {}", requestid);
        zmq::message_t msg(signal.data(), signal.size());
        multi.remove();
        multi.add(std::move(msg));
        multi.send(fail);
        return;
    } catch (...) {
        /*
         * what to do here depends on why this failed - maybe re-init the
         * transfer object, maybe re-init the sockets (by breaking the loop),
         * maybe take down the service
         */
        throw;
    }
    timer.transfer = clock::now();

    nlohmann::json manifest;
    try {
        manifest = nlohmann::json::parse(cfg.doc);
    } catch (const nlohmann::json::parse_error& e) {
        /* log error, and await new request */
        spdlog::error(
            "{} - badly formatted manifest: {}/{}",
            requestid,
            batch.root,
            batch.storage_endpoint,
            batch.guid
        );
        return;
    }
    timer.json = clock::now();

    /* set request type-independent parameters */
    /* these really shouldn't fail, and should mean immediate debug */
    fetchrequest.set_requestid(apirequest.requestid());
    fetchrequest.set_root(apirequest.root());
    fetchrequest.set_storage_endpoint(apirequest.storage_endpoint());
    fetchrequest.set_guid(apirequest.guid());
    *fetchrequest.mutable_fragment_shape() = apirequest.shape();

    /* set request type-specific parameters */
    switch (apirequest.function_case()) {
        using oneof = oneseismic::api_request;

        case oneof::kSlice:
            if (not set_slice_request(fetchrequest, apirequest, manifest))
                return;
            break;

        default:
            /*
             * this means a malformed input message - log the error, then
             * just await new request
             */
            spdlog::debug(
                "{} - malformed input, bad request variant (oneof)",
                requestid
            );
            return;
    }

    /* forward request to workers */
    fetchrequest.SerializeToString(&msg);
    timer.serialize = clock::now();
    zmq::message_t rep(msg.data(), msg.size());
    /* send shouldn't fail (?) in zmq, or at least internally retry (?) */
    multi.remove();
    multi.add(std::move(rep));
    multi.send(output);

    timer.send = clock::now();
    const auto ms = [] (std::chrono::duration< double, std::milli> p) {
        using namespace std::chrono;
        return duration_cast< milliseconds >(p).count();
    };

    spdlog::info(
        "time-manifest "
        "id: {} "
        "recv: {} "
        "parse: {} "
        "xfer: {} "
        "json: {} "
        "format: {} "
        "send: {} "
        "total: {} "
        "unit: ms",
        apirequest.requestid(),
        ms(timer.recv      - timer.start),
        ms(timer.parse     - timer.recv),
        ms(timer.transfer  - timer.parse),
        ms(timer.json      - timer.transfer),
        ms(timer.serialize - timer.json),
        ms(timer.send      - timer.serialize),
        ms(timer.send      - timer.start)
    );
}

}
