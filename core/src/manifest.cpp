#include <string>

#include <fmt/format.h>
#include <nlohmann/json.hpp>
#include <zmq.hpp>

#include <oneseismic/tasks.hpp>
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

    void onfailure(
            const one::buffer&,
            const one::batch&,
            const std::string&) override {
        // TODO: Should there be some sort of automatic retry, or should that
        // decision be pushed up? For now, just fail
        // TODO: It is *very* likely this should also be responsible to
        // weed out requests to non-existing cubes, so it is likely it should
        // push an error message down instead
        throw std::runtime_error("Error fetching manifest");
    }

    one::buffer doc;
};

void set_slice_request(
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
        throw std::runtime_error(
            fmt::format("lineno (= {}) not found in index", lineno)
        );
    }

    const auto pin = std::distance(index.begin(), itr);
    auto gvt = geometry(manifest_dimensions, api.shape());

    const auto ids = gvt.slice(
        one::dimension< 3 >(dim),
        pin / gvt.fragment_shape()[dim]
    );

    for (const auto& id : ids) {
        auto* c = req.add_ids();
        c->set_dim0(id[0]);
        c->set_dim1(id[1]);
        c->set_dim2(id[2]);
    }

    auto* slice = req.mutable_slice();
    slice->set_dim(dim);
    slice->set_idx(pin % gvt.fragment_shape()[dim]);
}

}

void manifest_task::run(
        transfer& xfer,
        zmq::socket_t& input,
        zmq::socket_t& output) {
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
    zmq::message_t req;
    input.recv(req, zmq::recv_flags::none);
    const auto ok = apirequest.ParseFromArray(req.data(), req.size());
    if (!ok) {
        /* log bad request, then be ready to receive new message */
        return;
    }

    /* fetch manifest */
    one::batch batch;
    batch.root = apirequest.root();
    batch.guid = apirequest.guid();
    batch.fragment_ids.resize(1);
    manifest_cfg cfg;
    try {
        xfer.perform(batch, cfg);
    } catch (...) {
        /*
         * what to do here depends on why this failed - maybe re-init the
         * transfer object, maybe re-init the sockets (by breaking the loop),
         * maybe take down the service
         */
        throw;
    }

    nlohmann::json manifest;
    try {
        manifest = nlohmann::json::parse(cfg.doc);
    } catch (nlohmann::json::parse_error& e) {
        /* log error, and await new request */
        return;
    }

    /* set request type-independent parameters */
    /* these really shouldn't fail, and should mean immediate debug */
    fetchrequest.set_requestid(apirequest.requestid());
    fetchrequest.set_root(apirequest.root());
    fetchrequest.set_guid(apirequest.guid());
    *fetchrequest.mutable_shape() = apirequest.shape();

    /* set request type-specific parameters */
    switch (apirequest.function_case()) {
        using anyof = oneseismic::api_request;

        case anyof::kSlice:
            set_slice_request(fetchrequest, apirequest, manifest);
            break;

        default:
            /*
             * this means a malformed input message - log the error, then
             * just await new request
             */
            throw std::runtime_error("api: unknown request type");
            return;
    }

    /* forward request to workers */
    fetchrequest.SerializeToString(&msg);
    zmq::message_t rep(msg.data(), msg.size());
    /* make this communication multi-part, to track liveness? */
    /* send shouldn't fail (?) in zmq, or at least internally retry (?) */
    output.send(rep, zmq::send_flags::none);
}

}
