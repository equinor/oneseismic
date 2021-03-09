#include <fmt/format.h>
#include <nlohmann/json.hpp>

#include <oneseismic/geometry.hpp>
#include <oneseismic/messages.hpp>

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
        out.shape_cube.push_back(dimension.size());

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

std::size_t task_count(std::size_t jobs, std::size_t task_size) {
    /*
     * Return the number of task-size'd tasks needed to process all jobs
     */
    return (jobs + (task_size - 1)) / task_size;
}

}

namespace one {

std::vector< std::vector< char > >
mkschedule(const char* doc, int len, int task_size) noexcept (false) {
    one::slice_task request;
    request.unpack(doc, doc + len);
    const auto manifest = nlohmann::json::parse(request.manifest);

    auto fetch = build_slice_fetch(request, manifest);
    const auto ids = fetch.ids;

    const auto ntasks = task_count(ids.size(), task_size);
    auto first = ids.begin();
    auto end = ids.end();

    std::vector< std::vector< char > > xs;
    for (int i = 0; i < int(ntasks); ++i) {
        if (first == end)
            break;

        auto last = std::min(first + task_size, end);
        std::advance(first, fetch.ids.size());
        auto packed = fetch.pack();
        xs.emplace_back(packed.begin(), packed.end());
    }

    return xs;
}

}
