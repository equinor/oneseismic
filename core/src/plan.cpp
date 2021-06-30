#include <algorithm>
#include <cassert>
#include <iterator>
#include <string>
#include <vector>

#include <fmt/format.h>
#include <nlohmann/json.hpp>

#include <oneseismic/geometry.hpp>
#include <oneseismic/messages.hpp>
#include <oneseismic/plan.hpp>

namespace {

one::gvt< 3 > geometry(const one::basic_query& query) noexcept (false) {
    const auto& dimensions = query.manifest.dimensions;
    const auto& shape = query.shape;

    return one::gvt< 3 > {
        { dimensions[0].size(),
          dimensions[1].size(),
          dimensions[2].size(), },
        { std::size_t(shape[0]),
          std::size_t(shape[1]),
          std::size_t(shape[2]), }
    };
}

int task_count(int jobs, int task_size) {
    /*
     * Return the number of task-size'd tasks needed to process all jobs
     */
    const auto x = (jobs + (task_size - 1)) / task_size;
    assert(x != 0);
    if (x <= 0) {
        const auto msg = "task-count < 0; probably integer overflow";
        throw std::runtime_error(msg);
    }
    return x;
}

/*
 * Scheduling
 * ----------
 * Scheduling in this context means the process of:
 *   1. parse an incoming request, e.g. /slice/<dim>/<lineno>
 *   2. build all task descriptions (fragment id + what to extract from
 *      the fragment)
 *   3. split the set of tasks into units of work
 *
 * I/O, the sending of messages to worker nodes is outside this scope.
 *
 * It turns out that the high-level algorithm is largely independent of the
 * task description, so if the "task constructor" is dependency injected then
 * the overall algorithm can be shared between all endpoints.
 *
 * To make matters slightly more complicated, a lot of constraints and
 * functionality is encoded in the types used for messages. It *could*, and
 * still may in the future, be implemented with inheritance, but that approach
 * too comes with its own set of drawbacks.
 *
 * While the types are different, the algorithm *structure* is identical. This
 * makes it a good fit for templates. It comes with some complexity of
 * understanding, but makes adding new endpoints a lot easier, and the reuse of
 * implementation means shared improvements and faster correctness guarantees.
 *
 * This comes with a very real tax for comprehensibility. Templates do add some
 * noise, and the algorithm is split across multiple functions that can all be
 * customised. I anticipate little need for many customisations, but
 * supporting extra customisation points adds very little extra since it just
 * hooks into the machinery required by a single customisation point.
 *
 * The benefit is that adding new endpoints now is a *lot* easier and less
 * error prone.
 */

/*
 * Default implementations and customization points for the scheduling steps.
 * In general, you should only need to implement build() and header() for new
 * endpoints, but partition() and make() are made availble should there be a
 * need to customize them too.
 */
template < typename Input, typename Output >
struct schedule_maker {
    /*
     * Build the schedule - parse the incoming request and build the set of
     * fragment IDs and extraction descriptions. This function is specific to
     * the shape (slice, curtain, horizon etc) and comes with no default
     * implementation.
     *
     * The Output type should have a pack() method that returns a std::string
     */
    Output build(const Input&) noexcept (false);

    /*
     * Make a header. This function requires deep knowledge of the shape and
     * oneseismic geometry, and must be implemented for all shape types.
     *
     * The information in the process header is crucial for efficient and
     * precise assembly of the end-result on the client side. Otherwise,
     * clients must buffer the full response, then parse it and make sense of
     * the shape, keys, linenos etc after the fact, since data can arrive in
     * arbitrary order. This makes detecting errors more difficult too. The
     * process header should provide enough information for clients to properly
     * pre-allocate and build metadata to make sense of data as it is streamed.
     */
    one::process_header
    header(const Input&, int ntasks) noexcept (false);

    /*
     * Partition partitions an Output in-place and pack()s it into blobs of
     * task_size jobs. It assumes the Output type has a vector-like member
     * called 'ids'. This is a name lookup - should the member be named
     * something else or accessed in a different way then you must implement a
     * custom partition().
     */
    std::vector< std::string >
    partition(Output&, int task_size) noexcept (false);

    /*
     * Make a schedule() - calls build(), header(), and partition() in
     * sequence. The output vector should always have the header() as the
     * *last* element.
     */
    std::vector< std::string >
    schedule(const char* doc, int len, int task_size) noexcept (false);
};

template < typename Input, typename Output >
std::vector< std::string >
schedule_maker< Input, Output >::partition(
        Output& output,
        int task_size
) noexcept (false) {
    if (task_size < 1) {
        const auto msg = fmt::format("task_size (= {}) < 1", task_size);
        throw std::logic_error(msg);
    }

    const auto ids = output.ids;
    const auto ntasks = task_count(ids.size(), task_size);

    using std::begin;
    using std::end;
    auto fst = begin(ids);
    auto lst = end(ids);

    std::vector< std::string > xs;
    for (int i = 0; i < ntasks; ++i) {
        const auto last = std::min(fst + task_size, lst);
        output.ids.assign(fst, last);
        std::advance(fst, last - fst);
        xs.push_back(output.pack());
    }

    return xs;
}

template < typename Input, typename Output >
std::vector< std::string >
schedule_maker< Input, Output >::schedule(
        const char* doc,
        int len,
        int task_size)
noexcept (false) {
    Input in;
    in.unpack(doc, doc + len);
    auto fetch = this->build(in);
    auto sched = this->partition(fetch, task_size);

    const auto ntasks = int(sched.size());
    const auto head   = this->header(in, ntasks);
    sched.push_back(head.pack());
    return sched;
}

template <>
one::slice_task
schedule_maker< one::slice_query, one::slice_task >::build(
    const one::slice_query& query)
{
    auto task = one::slice_task(query);

    const auto& index = [](const auto& query) {
        try {
            return query.manifest.dimensions.at(query.dim);
        } catch (std::out_of_range&) {
            const auto msg = fmt::format(
                "args.dim (= {}) not in [0, {})",
                query.dim,
                query.manifest.dimensions.size()
            );
            throw one::not_found(msg);
        }
    }(query);

    const auto itr = std::find(index.begin(), index.end(), query.lineno);
    if (itr == index.end()) {
        const auto msg = "line (= {}) not found in index";
        throw one::not_found(fmt::format(msg, query.lineno));
    }

    const auto pin = std::distance(index.begin(), itr);
    auto gvt = geometry(query);

    const auto to_vec = [](const auto& x) {
        return std::vector< int > { int(x[0]), int(x[1]), int(x[2]) };
    };

    task.lineno = pin % gvt.fragment_shape()[query.dim];
    const auto ids = gvt.slice(gvt.mkdim(query.dim), pin);
    // TODO: name loop
    for (const auto& id : ids)
        task.ids.push_back(to_vec(id));

    return task;
}

template <>
one::process_header
schedule_maker< one::slice_query, one::slice_task >::header(
    const one::slice_query& query,
    int ntasks
) noexcept (false) {
    const auto& mdims = query.manifest.dimensions;
    const auto gvt  = geometry(query);
    const auto dim  = gvt.mkdim(query.dim);
    const auto gvt2 = gvt.squeeze(dim);
    const auto fs2  = gvt2.fragment_shape();

    one::process_header head;
    head.pid    = query.pid;
    head.ntasks = ntasks;

    /*
     * The shape of a slice are the dimensions of the survey squeezed in that
     * dimension.
     */
    for (std::size_t i = 0; i < fs2.size(); ++i) {
        const auto dim = gvt2.mkdim(i);
        head.shape.push_back(gvt2.nsamples(dim));
    }

    /*
     * Build the index from the line numbers for the directions !=
     * params.lineno
     */
    for (std::size_t i = 0; i < mdims.size(); ++i) {
        if (i == query.dim) continue;
        head.index.push_back(mdims[i]);
    }
    return head;
}

/*
 * Compute the cartesian coordinate of the label/line numbers. This is
 * effectively a glorified indexof() in practice, although conceptually it
 * maps from a user-oriented grid to its internal representation. The cartesian
 * coordinates are taken at face value by the rest of the system, and can be
 * used for lookup directly. From oneseismic's point of view, the grid labels
 * are forgotten after this function is called.
 */
void to_cartesian_inplace(
    const std::vector< int >& labels,
    std::vector< int >& xs)
noexcept (false) {
    assert(std::is_sorted(labels.begin(), labels.end()));

    auto indexof = [&labels](auto x) {
        const auto itr = std::lower_bound(labels.begin(), labels.end(), x);
        if (*itr != x) {
            const auto msg = fmt::format("lineno {} not in index");
            throw one::not_found(msg);
        }
        return std::distance(labels.begin(), itr);
    };

    std::transform(xs.begin(), xs.end(), xs.begin(), indexof);
}

template <>
one::curtain_task
schedule_maker< one::curtain_query, one::curtain_task >::build(
    const one::curtain_query& query)
{
    const auto less = [](const auto& lhs, const auto& rhs) noexcept (true) {
        return std::lexicographical_compare(
            lhs.id.begin(),
            lhs.id.end(),
            rhs.begin(),
            rhs.end()
        );
    };
    const auto equal = [](const auto& lhs, const auto& rhs) noexcept (true) {
        return std::equal(lhs.begin(), lhs.end(), rhs.begin());
    };

    auto task = one::curtain_task(query);
    auto dim0s = query.dim0s;
    auto dim1s = query.dim1s;
    to_cartesian_inplace(query.manifest.dimensions[0], dim0s);
    to_cartesian_inplace(query.manifest.dimensions[1], dim1s);
    auto& ids = task.ids;

    auto gvt = geometry(query);
    const auto zfrags  = gvt.fragment_count(gvt.mkdim(2));

    /*
     * Guess the number of coordinates per fragment. A reasonable assumption is
     * a plane going through a fragment, with a little bit of margin. Not
     * pre-reserving is perfectly fine, but we can save a bunch of allocations
     * in the average case by guessing well. It is reasonably short-lived, so
     * overestimating slightly should not be a problem.
     */
    const auto approx_coordinates_per_fragment =
        int(std::max(gvt.fragment_shape()[0], gvt.fragment_shape()[1]) * 1.2);

    /*
     * Pre-allocate the id objects by scanning the input and build the
     * one::single objects, sorted by id lexicographically. All fragments in
     * the column (z-axis) are generated from the x-y pair. This is essentially
     * constructing the "buckets" in advance, as many x/y pairs will end up in
     * the same "bin"/fragment.
     *
     * This is effectively
     *  ids = set([fragmentid(x, y, z) for z in zheight for (x, y) in input])
     *
     * but without any intermediary structures.
     *
     * The bins are lexicographically sorted.
     */
    for (int i = 0; i < int(dim0s.size()); ++i) {
        auto top_point = one::CP< 3 > {
            std::size_t(dim0s[i]),
            std::size_t(dim1s[i]),
            std::size_t(0),
        };
        const auto fid = gvt.frag_id(top_point);

        auto itr = std::lower_bound(ids.begin(), ids.end(), fid, less);
        if (itr == ids.end() or (not equal(itr->id, fid))) {
            one::single top;
            top.id.assign(fid.begin(), fid.end());
            top.coordinates.reserve(approx_coordinates_per_fragment);
            itr = ids.insert(itr, zfrags, top);
            for (int z = 0; z < zfrags; ++z, ++itr)
                itr->id[2] = z;
        }
    }

    /*
     * Traverse the x/y coordinates and put them in the correct bins/fragment
     * ids.
     */
    for (int i = 0; i < int(dim0s.size()); ++i) {
        const auto cp = one::CP< 3 > {
            std::size_t(dim0s[i]),
            std::size_t(dim1s[i]),
            std::size_t(0),
        };
        const auto fid = gvt.frag_id(cp);
        const auto lid = gvt.to_local(cp);
        auto itr = std::lower_bound(ids.begin(), ids.end(), fid, less);
        const auto end = itr + zfrags;
        for (auto block = itr; block != end; ++block) {
            block->coordinates.push_back({ int(lid[0]), int(lid[1]) });
        }
    }

    return task;
}

template <>
one::process_header
schedule_maker< one::curtain_query, one::curtain_task >::header(
    const one::curtain_query& query,
    int ntasks
) noexcept (false) {
    const auto& mdims = query.manifest.dimensions;

    one::process_header head;
    head.pid    = query.pid;
    head.ntasks = ntasks;

    const auto gvt  = geometry(query);
    const auto zpad = gvt.nsamples_padded(gvt.mkdim(gvt.ndims - 1));
    head.shape = {
        int(query.dim0s.size()),
        int(zpad),
    };

    head.index.push_back(query.dim0s);
    head.index.push_back(query.dim1s);
    head.index.push_back(mdims.back());
    to_cartesian_inplace(mdims[0], head.index[0]);
    to_cartesian_inplace(mdims[1], head.index[1]);
    return head;
}

}

namespace one {

std::vector< std::string >
mkschedule(const char* doc, int len, int task_size) noexcept (false) {
    const auto document = nlohmann::json::parse(doc, doc + len);
    const std::string function = document["function"];
    if (function == "slice") {
        auto slice = schedule_maker< slice_query, slice_task >{};
        return slice.schedule(doc, len, task_size);
    }
    if (function == "curtain") {
        auto curtain = schedule_maker< curtain_query, curtain_task >{};
        return curtain.schedule(doc, len, task_size);
    }
    throw std::logic_error("No handler for function " + function);
}

}
