#include <cassert>
#include <exception>
#include <string>

#include <fmt/format.h>
#include <nlohmann/json.hpp>
#include <spdlog/spdlog.h>
#include <zmq.hpp>
#include <zmq_addon.hpp>

#include <oneseismic/azure.hpp>
#include <oneseismic/geometry.hpp>
#include <oneseismic/messages.hpp>
#include <oneseismic/tasks.hpp>
#include <oneseismic/transfer.hpp>

namespace one {

struct line_not_found : std::invalid_argument {
    using invalid_argument::invalid_argument;
};

namespace {

one::gvt< 3 > geometry(
        const nlohmann::json& dimensions,
        const nlohmann::json& shape) noexcept (false) {
    return one::gvt< 3 > {
        { dimensions[0].size(),
          dimensions[1].size(),
          dimensions[2].size(), },
        { shape[0].get< std::size_t >(),
          shape[1].get< std::size_t >(),
          shape[2].get< std::size_t >(), }
    };
}

slice_fetch build_slice_fetch(
        const slice_task& task,
        const nlohmann::json& manifest)
noexcept (false) {
    auto out = slice_fetch(task);

    /*
     * TODO:
     * faster to not make vector, but rather parse-and-compare individual
     * integers?
     */
    const auto& manifest_dimensions = manifest["dimensions"];
    const auto index = manifest_dimensions[task.dim].get< std::vector< int > >();
    const auto itr = std::find(index.begin(), index.end(), task.lineno);
    if (itr == index.end()) {
        const auto msg = "line (= {}) not found in index";
        throw line_not_found(fmt::format(msg, task.lineno));
    }

    const auto pin = std::distance(index.begin(), itr);
    auto gvt = geometry(manifest_dimensions, task.shape);

    // TODO: name loop
    for (const auto& dimension : manifest_dimensions)
        out.cube_shape.push_back(dimension.size());

    const auto to_vec = [](const auto& x) {
        return std::vector< int > { int(x[0]), int(x[1]), int(x[2]) };
    };

    out.lineno = pin % gvt.fragment_shape()[task.dim];
    const auto ids = gvt.slice(one::dimension< 3 >(task.dim), pin);
    // TODO: name loop
    for (const auto& id : ids)
        out.ids.push_back(to_vec(id));
    return out;
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
     * While it's ugly that pid instances are reused, doing so should
     * mean less allocation pressure.
     */
    zmq::multipart_t failure(const std::string& key) noexcept (false);

    std::string pid;
    slice_task request;
    int task_size = 10;
    working_storage storage;
};

void manifest_task::connect_working_storage(const std::string& addr) {
    this->p->storage.connect(addr);
}

void manifest_task::impl::parse(const zmq::multipart_t& task) {
    assert(task.size() == 2);
    const auto& pid  = task[0];
    const auto& body = task[1];

    this->pid.assign(static_cast< const char* >(pid.data()),  pid.size());
    // C++17 string_view
    const auto* fst = static_cast< const char* >(body.data());
    const auto* lst = static_cast< const char* >(fst + body.size());
    this->request.unpack(fst, lst);
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

std::string make_result_header(int chunks) {
    nlohmann::json header;
    header["parts"] = chunks;
    return header.dump();
}

void manifest_task::run(
        zmq::socket_t& input,
        zmq::socket_t& output,
        zmq::socket_t& failure)
try {
    auto process = zmq::multipart_t(input);
    this->p->parse(process);

    const auto& pid = this->p->pid;
    const auto& request = this->p->request;
    const auto manifest = nlohmann::json::parse(request.manifest);

    spdlog::info( "pid={}, max_task_size={}", pid, this->max_task_size());

    auto fetch = build_slice_fetch(request, manifest);
    const auto ids = fetch.ids;
    const auto chunk_size = this->p->task_size;
    const auto chunks = chunk_count(ids.size(), chunk_size);
    auto first = ids.begin();
    auto end = ids.end();

    /*
     * TODO: object name generation should have tests on both sides to ensure
     * consistency
     */
    const auto header = make_result_header(chunks);
    const auto header_id = fmt::format("{}:header.json", pid);
    this->p->storage.put(header_id, header);

    for (std::size_t i = 0; i < chunks; ++i) {
        if (first == end)
            break;

        auto last = std::min(first + chunk_size, end);

        fetch.ids.assign(first, last);
        std::advance(first, fetch.ids.size());

        zmq::multipart_t msg;
        msg.addstr(pid);
        msg.addstr(fmt::format("{}/{}", i, chunks));
        msg.addstr(fetch.pack());
        msg.send(output);
        spdlog::info("pid={}, part={}/{} queued for fragment retrieval", pid, i, chunks);
    }
} catch (const bad_message&) {
    spdlog::error(
            "pid={}, badly formatted protobuf message",
            this->p->pid
    );
    this->p->failure("bad-message").send(failure);
} catch (const unauthorized&) {
    /*
     * TODO: log the headers?
     * TODO: log manifest url?
     */
    spdlog::info("pid={}, not authorized", this->p->pid);
    this->p->failure("manifest-not-authorized").send(failure);
} catch (const notfound& e) {
    spdlog::info(
            "pid={}, {} manifest not found: '{}'",
            this->p->pid,
            this->p->request.guid,
            e.what()
    );
    this->p->failure("manifest-not-found").send(failure);
} catch (const nlohmann::json::parse_error& e) {
    spdlog::error(
            "pid={}, badly formatted manifest: {}",
            this->p->pid,
            this->p->request.guid
    );
    spdlog::error(e.what());
    this->p->failure("json-parse-error").send(failure);
} catch (const line_not_found& e) {
    spdlog::info("pid={}, {}", this->p->pid, e.what());
    this->p->failure("line-not-found").send(failure);
} catch (const storage_error& e) {
    spdlog::warn("pid={}, storage error: {}", this->p->pid, e.what());
    this->p->failure("storage-error").send(failure);
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
