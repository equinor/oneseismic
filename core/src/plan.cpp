#include <algorithm>
#include <cassert>
#include <iterator>
#include <sstream>
#include <string>
#include <vector>

#include <fmt/format.h>
#include <msgpack.hpp>
#include <nlohmann/json.hpp>

#include <oneseismic/geometry.hpp>
#include <oneseismic/messages.hpp>
#include <oneseismic/plan.hpp>

namespace one {

namespace {

gvt< 3 > geometry(const basic_query& query) noexcept (false) {
    const auto& dimensions = query.manifest.line_numbers;
    const auto& shape = query.shape();

    return gvt< 3 > {
        { dimensions[0].size(),
          dimensions[1].size(),
          dimensions[2].size(), },
        { std::size_t(shape[0]),
          std::size_t(shape[1]),
          std::size_t(shape[2]), }
    };
}

gvt< 3 > geometry(const basic_task& task) noexcept (true) {
    return gvt< 3 > {
        {
            std::size_t(task.shape_cube[0]),
            std::size_t(task.shape_cube[1]),
            std::size_t(task.shape_cube[2]),
        },
        {
            std::size_t(task.shape[0]),
            std::size_t(task.shape[1]),
            std::size_t(task.shape[2]),
        }
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

template < typename Query >
std::vector< std::string > normalized_attributes(const Query& q) {
    std::vector< std::string > attrs;
    attrs.reserve(q.attributes.size() * 2);

    for (const auto& attr : q.attributes) {
        if (attr == "cdp") {
            attrs.push_back("cdpx");
            attrs.push_back("cdpy");
        } else {
            attrs.push_back(attr);
        }
    };

    std::sort(attrs.begin(), attrs.end());
    attrs.erase(
        std::unique(attrs.begin(), attrs.end()),
        attrs.end()
    );

    return attrs;
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
    std::vector< Output > build(const Input&) noexcept (false);

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
    process_header
    header(const Input&, int ntasks) noexcept (false);

    /*
     * Partition partitions an Output in-place and pack()s it into blobs of
     * task_size jobs. It assumes the Output type has a vector-like member
     * called 'ids'. This is a name lookup - should the member be named
     * something else or accessed in a different way then you must implement a
     * custom partition().
     *
     * This function shall return \0-separated packed tasks. The last task
     * shall be terminated with a \0, so that partition.count() can be
     * implemented as std::count(begin, end, '\0'). While it is a slightly
     * surprising interface (the most straight-forward would be
     * vector-of-string), it makes processing the set-of-tasks slightly easier,
     * saves a few allocations, and signals that the output is a bag of bytes.
     */
    taskset partition(std::vector< Output >&, int task_size) noexcept (false);

    /*
     * Make a schedule() - calls build(), header(), and partition() in
     * sequence. The output vector should always have the header() as the
     * *last* element.
     */
    taskset schedule(const char* doc, int len, int task_size) noexcept (false);
};

template< typename Outputs >
int count_tasks(const Outputs& outputs, int task_size) noexcept (true) {
    const auto add = [task_size](auto acc, const auto& elem) noexcept (true) {
        return acc + task_count(elem.ids.size(), task_size);
    };
    return std::accumulate(outputs.begin(), outputs.end(), 0, add);
}

template < typename Input, typename Output >
taskset schedule_maker< Input, Output >::partition(
    std::vector< Output >& outputs,
    int task_size
) noexcept (false) {
    if (task_size < 1) {
        const auto msg = fmt::format("task_size (= {}) < 1", task_size);
        throw std::logic_error(msg);
    }

    taskset partitioned;
    partitioned.reserve(count_tasks(outputs, task_size));

    for (auto& output : outputs) {
        const auto ids = output.ids;
        const auto ntasks = task_count(ids.size(), task_size);

        using std::begin;
        using std::end;
        auto fst = begin(ids);
        auto lst = end(ids);

        for (int i = 0; i < ntasks; ++i) {
            const auto last = std::min(fst + task_size, lst);
            output.ids.assign(fst, last);
            std::advance(fst, last - fst);
            partitioned.append(output.pack());
        }
    }

    return partitioned;
}

std::string pack_with_envelope(const process_header& head) {
    /*
     * The response message format is designed in such as way that the clients
     * can choose to buffer and parse the message in one go, or stream it. This
     * means that the message *as a whole* must be valid msgpack message, and
     * not just a by-convention concatenation of independent messages.
     *
     * The whole response as msgpack looks like this:
     * [header, [part1, part2, part3, ...]]
     *
     * which in bytes looks like:
     * array(2) header array(n) part1 part2 part3
     *
     * where "space" means concatenation, and array(k) is array type tag and
     * length. This functions add these array tags around the header.
     */
    std::stringstream buffer;
    msgpack::packer< decltype(buffer) > packer(buffer);
    packer.pack_array(2);
    buffer << head.pack();
    packer.pack_array(head.nbundles);
    return buffer.str();
}

template < typename Input, typename Output >
taskset schedule_maker< Input, Output >::schedule(
        const char* doc,
        int len,
        int task_size)
noexcept (false) {
    Input in;
    in.unpack(doc, doc + len);
    in.attributes = normalized_attributes(in);
    auto fetch = this->build(in);
    auto sched = this->partition(fetch, task_size);

    const auto ntasks = int(sched.count());
    const auto head   = this->header(in, ntasks);
    sched.append(pack_with_envelope(head));
    return sched;
}

template < typename Query >
auto find_desc(const Query& query, const std::string& attr)
-> decltype(query.manifest.attr.begin()) {
    return std::find_if(
        query.manifest.attr.begin(),
        query.manifest.attr.end(),
        [&attr](const auto& desc) noexcept {
            return desc.type == attr;
        }
    );
}

template < typename Seq >
void append_vector_ids(
    std::vector< std::vector< int > >& dst,
    const Seq& src)
{
    using std::begin;
    using std::end;

    const auto to_vec = [](const auto& s) noexcept (false) {
        const auto convert = [](const auto x) noexcept (true) {
            return int(x);
        };

        std::vector< int > xs(s.size());
        std::transform(begin(s), end(s), begin(xs), convert);
        return xs;
    };

    auto append = std::back_inserter(dst);
    std::transform(begin(src), end(src), append, to_vec);
}

template <>
std::vector< slice_task >
schedule_maker< slice_query, slice_task >::build(const slice_query& query) {
    auto task = slice_task(query);
    std::vector< slice_task > tasks;
    tasks.reserve(query.attributes.size() + 1);

    const auto gvt = geometry(query);
    const auto dim = gvt.mkdim(query.dim);
    task.idx = gvt.fragment_shape().index(dim, query.idx);
    const auto ids = gvt.slice(dim, query.idx);
    append_vector_ids(task.ids, ids);
    tasks.push_back(task);

    for (const auto& attr : query.attributes) {
        /*
         * It's perfectly common for queries to request attributes that aren't
         * recorded for a survey - in this case, silently drop it
         */
        auto itr = find_desc(query, attr);
        if (itr == query.manifest.attr.end())
            continue;

        auto task = slice_task(query, *itr);
        const auto gvt3 = geometry(task);
        /*
         * Attributes are really 2D volumes (depth = 1), but stored as 3D
         * volumes to make querying them trivial. However, when requesting
         * attributes for z-slices, the index will almost always not be 0 (the
         * only valid z-index in the attributes surface), but this applies only
         * for queries where dim = z. Modulus moves the index back into the
         * grid, and is a no-op for any index a valid dimension.
         */
        const auto idx = query.idx % gvt3.cube_shape()[dim];
        task.idx = gvt3.fragment_shape().index(dim, idx);
        const auto ids = gvt3.slice(dim, idx);
        append_vector_ids(task.ids, ids);
        tasks.push_back(task);
    }

    return tasks;
}

template <>
process_header
schedule_maker< slice_query, slice_task >::header(
    const slice_query& query,
    int ntasks
) noexcept (false) {
    const auto& mdims = query.manifest.line_numbers;

    process_header head;
    head.pid        = query.pid;
    head.function   = functionid::slice;
    head.nbundles   = ntasks;
    head.ndims      = mdims.size() - 1;
    head.attributes = query.attributes;

    /*
     * Build the index from the line numbers for the directions !=
     * params.lineno
     */
    for (std::size_t i = 0; i < mdims.size(); ++i) {
        if (i == query.dim) continue;
        head.index.push_back(mdims[i].size());
    }
    for (std::size_t i = 0; i < mdims.size(); ++i) {
        if (i == query.dim) continue;
        head.index.insert(head.index.end(), mdims[i].begin(), mdims[i].end());
    }
    return head;
}

template <>
std::vector< curtain_task >
schedule_maker< curtain_query, curtain_task >::build(
    const curtain_query& query)
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

    std::vector< curtain_task > tasks;
    tasks.emplace_back(query);
    auto& ids = tasks.back().ids;
    const auto& dim0s = query.dim0s;
    const auto& dim1s = query.dim1s;

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
        auto top_point = CP< 3 > {
            std::size_t(dim0s[i]),
            std::size_t(dim1s[i]),
            std::size_t(0),
        };
        const auto fid = gvt.frag_id(top_point);

        auto itr = std::lower_bound(ids.begin(), ids.end(), fid, less);
        if (itr == ids.end() or (not equal(itr->id, fid))) {
            single top;
            top.id.assign(fid.begin(), fid.end());
            top.coordinates.reserve(approx_coordinates_per_fragment);
            top.offset = i;
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
        const auto cp = CP< 3 > {
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

    return tasks;
}

template <>
process_header
schedule_maker< curtain_query, curtain_task >::header(
    const curtain_query& query,
    int ntasks
) noexcept (false) {
    const auto& mdims = query.manifest.line_numbers;

    process_header head;
    head.pid        = query.pid;
    head.function   = functionid::curtain;
    head.nbundles   = ntasks;
    head.ndims      = mdims.size();
    head.attributes = query.attributes;

    auto& index = head.index;

    index.push_back(query.dim0s .size());
    index.push_back(query.dim1s .size());
    index.push_back(mdims.back().size());

    index.insert(index.end(), query.dim0s .begin(), query.dim0s .end());
    index.insert(index.end(), query.dim1s .begin(), query.dim1s .end());
    index.insert(index.end(), mdims.back().begin(), mdims.back().end());

    return head;
}

}

taskset mkschedule(const char* doc, int len, int task_size) noexcept (false) {
    const auto document = nlohmann::json::parse(doc, doc + len);
    /*
     * Right now, only format-version: 1 is supported, but checking the format
     * version allow for multiple document versions to be supported as storage
     * migrates between the representation. Dispatch here to different
     * query-builder routines, depending on the format version.
     */
    const auto& manifest = document.at("manifest");
    if (manifest.at("format-version") != 1) {
        const auto msg = fmt::format(
            "unsupported format-version; expected {}, was {}",
            1,
            int(document.at("format-version"))
        );
        throw bad_document(msg);
    }

    const std::string function = document.at("function");
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
