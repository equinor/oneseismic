#include "slice.h"

#include <cstdlib>
#include <cstring>
#include <memory>

#include <fmt/format.h>
#include <nlohmann/json.hpp>

#include <oneseismic/geometry.hpp>
#include <oneseismic/messages.hpp>

/*
 * Right now the implementation is a straight-up copy from the
 * core/src/manifest.cpp file, in order to more quickly and easily develop and
 * experiment with redis streams rather than ZMQ for job scheduling. The design
 * and implementation is likely to evolve rapidly from here.
 */
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

one::slice_fetch build_slice_fetch(
        const one::slice_task& task,
        const nlohmann::json& manifest)
noexcept (false) {
    auto out = one::slice_fetch(task);

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
        throw std::invalid_argument(fmt::format(msg, task.lineno));
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

template < typename T >
void* copyalloc(const T& x) {
    void* p = std::malloc(x.size() * sizeof(*x.data()));
    std::memcpy(p, x.data(), x.size());
    return p;
}

std::size_t task_count(std::size_t jobs, std::size_t task_size) {
    /*
     * Return the number of task-size'd tasks needed to process all jobs
     */
    return (jobs + (task_size - 1)) / task_size;
}

}

tasks* mkschedule(const char* doc, int len, int task_size) {
    one::slice_task request;
    request.unpack(doc, doc + len);
    const auto manifest = nlohmann::json::parse(request.manifest);

    auto fetch = build_slice_fetch(request, manifest);
    const auto ids = fetch.ids;

    const auto ntasks = task_count(ids.size(), task_size);
    auto first = ids.begin();
    auto end = ids.end();

    struct deleter {
        void operator () (tasks* t) noexcept (true) {
            cleanup(t);
        }
    };

    std::unique_ptr< tasks, deleter > cs(new tasks());
    cs->err = nullptr;
    cs->tasks = new task[ntasks];
    cs->size = ntasks;
    for (std::size_t i = 0; i < ntasks; ++i) {
        if (first == end)
            break;

        auto& task = cs->tasks[i];
        auto last = std::min(first + task_size, end);

        fetch.ids.assign(first, last);
        std::advance(first, fetch.ids.size());
        auto packed = fetch.pack();
        task.size  = packed.size();
        task.task = copyalloc(packed);
    }

    return cs.release();
}

void cleanup(tasks* cs) {
    if (!cs) return;

    for (int i = 0; i < cs->size; ++i) {
        std::free((void*)cs->tasks[i].task);
    }
    delete cs->err;
    delete cs->tasks;
    delete cs;
}
