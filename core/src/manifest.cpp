#include <cassert>
#include <exception>
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

struct bad_message : std::exception {};
struct line_not_found : std::invalid_argument {
    using invalid_argument::invalid_argument;
};

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

/*
 * Simple automation for parsing api_request messages
 */
class api_request : public oneseismic::api_request {
public:
    void parse(const zmq::message_t&) noexcept (false);
};

void api_request::parse(const zmq::message_t& msg) noexcept (false) {
    const auto ok = this->ParseFromArray(msg.data(), msg.size());
    if (!ok) throw bad_message();
}

/*
 * This class is fairly straight-forward automation over the somewhat clunky
 * protobuf API.
 */
class fetch_request : public oneseismic::fetch_request {
public:
    /*
     * Set the basic, shared fields in a request (ID, guid etc)
     */
    void basic(const api_request&) noexcept (false);

    const std::string& serialize() noexcept (false);

    std::vector< one::FID< 3 > >
    slice(const api_request& req, const nlohmann::json&) noexcept (false);

    template < typename Itr >
    void assign(Itr first, Itr last) noexcept (false);

private:
    /*
     * Cache the string-buffer used for serialization, to avoid allocating new
     * memory every time a message is serialized. There's a practical upper
     * bound on how large messages get, so this should very quickly reach a
     * size where no more allocations will happen.
     */
    std::string serialized;
};

std::vector< one::FID< 3 > >
fetch_request::slice(
        const api_request& api,
        const nlohmann::json& manifest)
noexcept (false) {
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
        const auto msg = fmt::format("line (= {}) not found in index");
        throw line_not_found(msg);
    }

    const auto pin = std::distance(index.begin(), itr);
    auto gvt = geometry(manifest_dimensions, api.shape());

    auto* cs = this->mutable_cube_shape();
    cs->set_dim0(manifest_dimensions[0].size());
    cs->set_dim1(manifest_dimensions[1].size());
    cs->set_dim2(manifest_dimensions[2].size());

    auto* slice = this->mutable_slice();
    slice->set_dim(dim);
    slice->set_idx(pin % gvt.fragment_shape()[dim]);

    return gvt.slice(one::dimension< 3 >(dim), pin);
}

template < typename Itr >
void fetch_request::assign(Itr first, Itr last) noexcept (false) {
    this->clear_ids();
    std::for_each(first, last, [this] (const auto& id) {
        auto* c = this->add_ids();
        c->set_dim0(id[0]);
        c->set_dim1(id[1]);
        c->set_dim2(id[2]);
    });
}



const std::string& fetch_request::serialize() noexcept (false) {
    this->SerializeToString(&this->serialized);
    return this->serialized;
}

nlohmann::json get_manifest(
        one::transfer& xfer,
        const std::string& root,
        const std::string& storage_endpoint,
        const std::string& guid)
noexcept (false) {

    one::batch batch;
    batch.root = root;
    batch.storage_endpoint = storage_endpoint;
    batch.guid = guid;
    batch.fragment_ids.resize(1);
    manifest_cfg cfg;
    xfer.perform(batch, cfg); // TODO: get(object) -> bytes
    return nlohmann::json::parse(cfg.doc);
}

void fetch_request::basic(const api_request& req) {
    /* set request type-independent parameters */
    /* these really shouldn't fail, and should mean immediate debug */
    this->set_requestid(req.requestid());
    this->set_storage_endpoint(req.storage_endpoint());
    this->set_root(req.root());
    this->set_guid(req.guid());
    *this->mutable_fragment_shape() = req.shape();
}

std::size_t chunk_count(std::size_t jobs, std::size_t chunk_size) {
    /*
     * Return the number of chunk-size'd chunks needed to process all jobs
     */
    return (jobs + (chunk_size - 1)) / chunk_size;
}

}

/**
 *
 * A storage and cache class for the task, so that protobuf instances [1] and
 * other objects can be reused, without having to expose them in the headers.
 *
 * [1] https://github.com/protocolbuffers/protobuf/blob/master/docs/performance.md
 */
class manifest_task::impl {
public:
    void parse(const zmq::multipart_t& task);

    /*
     * Create a failure message with the current pid and an appropriate
     * category, for signaling downstream that a job must be failed for some
     * reason.
     *
     * While it's ugly that pid+address instances are reused, doing so should
     * mean less allocation pressure.
     */
    zmq::multipart_t failure(const std::string& key) noexcept (false);

    std::string pid;
    std::string address;
    api_request request;
    fetch_request query;
    int task_size = 10;
};

void manifest_task::impl::parse(const zmq::multipart_t& task) {
    assert(task.size() == 3);
    const auto& addr = task[0];
    const auto& pid  = task[1];
    const auto& body = task[2];

    this->address.assign(static_cast< const char* >(addr.data()), addr.size());
    this->pid.assign(    static_cast< const char* >(pid.data()),  pid.size());
    this->request.parse(body);
}

zmq::multipart_t manifest_task::impl::failure(const std::string& key)
noexcept (false) {
    zmq::multipart_t msg;
    msg.addstr(this->pid);
    msg.addstr(key);
    return msg;
}

/*
 * The run() function:
 *  - pulls a job from the session manager queue
 *  - reads the request and fetches a manifest from storage
 *  - reads the manifest and uses it to create job descriptions
 *  - pushes job description on the output queue
 *
 * This requires tons of helpers, and they all use exceptions to communicate
 * failures, whenever they can't do the job. That includes a failed transfer,
 * requesting objects that aren't in storage, and more.
 *
 * The outcome of a lot of failures is to signal the error back to whoevers
 * listening, discard this request, and wait for a new one. The most obvious
 * example is when a cube is requested that isn't in storage - detecting this
 * is the responsibility of the manifest task, whenever 404 NOTFOUND is
 * received from the storage. This isn't a hard error, just bad input, and
 * there are no fragment jobs to schedule.
 *
 * Centralised handling of failures also mean that the failure socket don't
 * have to be sent farther down the call chain, which really cleans up
 * interfaces and control flow.
 *
 * In most other code, it would make sense to have the catch much closer to the
 * source of the failure, but instead distinct exception types are used to
 * distinguish expected failures. Now, all failures are pretty much handled the
 * same way (signal error to session manager, log it, and continue operating),
 * and keeping them together gives a nice symmetry to it.
 *
 * This could have been accomplished with std::outcome or haskell's Either, but
 * since the function doesn't return anything anyway, it becomes a bit awkward.
 * Furthermore, the exceptions can be seen as a side-channel Either with the
 * failure source embedded.
 */
void manifest_task::run(
        transfer& xfer,
        zmq::socket_t& input,
        zmq::socket_t& output,
        zmq::socket_t& failure)
try {
    auto process = zmq::multipart_t(input);
    this->p->parse(process);

    const auto manifest = get_manifest(
            xfer,
            this->p->request.root(),
            this->p->request.storage_endpoint(),
            this->p->request.guid()
    );

    auto& query = this->p->query;
    query.basic(this->p->request);
    std::vector< one::FID< 3 > > ids;
    switch (this->p->request.function_case()) {
        using oneof = oneseismic::api_request;

        case oneof::kSlice:
            ids = query.slice(this->p->request, manifest);
            break;

        default:
            spdlog::error(
                "{} bad request variant (oneof)",
                this->p->pid
            );
            return;
    }

    const auto chunk_size = this->p->task_size;
    const auto chunks = chunk_count(ids.size(), chunk_size);
    auto first = ids.begin();
    auto end = ids.end();

    for (std::size_t i = 0; i < chunks; ++i) {
        if (first == end)
            break;

        auto last = std::min(first + chunk_size, end);
        query.assign(first, last);
        std::advance(first, std::distance(first, last));

        zmq::multipart_t msg;
        msg.addstr(this->p->address);
        msg.addstr(this->p->pid);
        msg.addstr(fmt::format("{}/{}", i, chunks));
        msg.addstr(this->p->query.serialize());
        msg.send(output);
        spdlog::info(
                "{} {}/{} queued for fragment retrieval",
                this->p->pid,
                i,
                chunks
        );
    }

} catch (const bad_message&) {
    spdlog::error(
            "{} badly formatted protobuf message",
            this->p->pid
    );
    this->p->failure("bad-message").send(failure);
} catch (const notfound& e) {
    spdlog::info(
            "{} {} manifest not found: '{}'",
            this->p->pid,
            this->p->request.guid(),
            e.what()
    );
    this->p->failure("manifest-not-found").send(failure);
} catch (const nlohmann::json::parse_error& e) {
    spdlog::error(
            "{} badly formatted manifest: {}/{}",
            this->p->pid,
            this->p->request.root(),
            this->p->request.guid()
    );
    spdlog::error(e.what());
    this->p->failure("json-parse-error").send(failure);
} catch (const line_not_found& e) {
    spdlog::info("{} {}", this->p->pid, e.what());
    this->p->failure("line-not-found").send(failure);
}

int manifest_task::max_task_size() const noexcept (true) {
    return this->p->task_size;
}

void manifest_task::max_task_size(int size) noexcept (false) {
    if (size <= 0) {
        const auto msg = "expected task size > 0, was {}";
        throw std::invalid_argument(fmt::format(msg, size));
    }
    this->p->task_size = size;
}

manifest_task::manifest_task() : p(new impl()) {}
manifest_task::~manifest_task() = default;

}
