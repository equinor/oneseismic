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

/*
 * Stupid helper to make it less noisy to unpack the parallel arrays in the
 * curtain input to a geometry/cube-point.
 *
 * Curtain (query) input:
 *  intersections: [[x1 y1] [x2 y2] [x3 y3]]
 *
 * _query message:
 *  dim0s: [x1 x2 x3]
 *  dim1s: [y1 y2 y3]
 *
 * maps to coordinates (x1 y1 0) (x2 y2 0) (x3 y3 0)
 *
 * The top point can then be used to identify the containing fragment ID and
 * its z-axis column.
 */
[[nodiscard]]
CP< 3 > top_cubepoint(
    const std::vector< int >& xs,
    const std::vector< int >& ys,
    int i)
noexcept (false) {
    return CP< 3 > {
        std::size_t(xs[i]),
        std::size_t(ys[i]),
        std::size_t(0),
    };
}

[[nodiscard]]
std::array< int, 2 > coordinate(const FP< 3 > localid) noexcept (false) {
    return {
        int(localid[0]),
        int(localid[1]),
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

template < typename Query >
auto find_attribute(const Query& query, const std::string& attr)
-> std::tuple< decltype(query.manifest.attr.begin()), bool> {
    auto itr = std::find_if(
        query.manifest.attr.begin(),
        query.manifest.attr.end(),
        [&attr](const auto& desc) noexcept {
            return desc.type == attr;
        }
    );

    return { itr, itr != query.manifest.attr.end() };
}

std::vector< std::array< int, 3 > >
convert(const std::vector< FID< 3 > >& xs) noexcept (false) {
    auto conv = [](const auto& fid) noexcept {
        std::array< int, 3 > x;
        std::copy_n(fid.begin(), x.size(), x.begin());
        return x;
    };

    std::vector< std::array< int, 3 > > out(xs.size());
    std::transform(xs.begin(), xs.end(), out.begin(), conv);
    return out;
}

std::vector< slice_task > build(const slice_query& query) {
    std::vector< slice_task > tasks;
    tasks.reserve(query.attributes.size() + 1);

    tasks.emplace_back(query);
    for (const auto& attr : query.attributes) {
        auto [itr, found] = find_attribute(query, attr);
        if (not found)
            continue;

        tasks.emplace_back(query, *itr);
    }

    for (auto& task : tasks) {
        const auto gvt = geometry(task);
        const auto dim = gvt.mkdim(query.dim);
        task.idx = gvt.fragment_shape().index(dim, query.idx);
        task.ids = convert(gvt.slice(dim, query.idx));
    };

    return tasks;
}

process_header header(const slice_query& query, int ntasks)
noexcept (false) {
    const auto& mdims = query.manifest.line_numbers;

    process_header head;
    head.pid        = query.pid;
    head.function   = functionid::slice;
    head.nbundles   = ntasks;
    head.ndims      = mdims.size();
    head.labels     = query.manifest.line_labels;
    head.attributes.push_back("data");
    head.attributes.insert(
        head.attributes.end(),
        query.attributes.begin(),
        query.attributes.end()
    );

    /*
     * Build the (line number) index of the output. Notice that the queried
     * direction is also included here, so that users can get infer what line
     * was queried (useful when source is index or coordinate) and the
     * direction of the output.
     */
    for (std::size_t i = 0; i < mdims.size(); ++i) {
        if (i != query.dim) {
            head.index.push_back(mdims[i].size());
        } else {
            head.index.push_back(1);
        }
    }
    for (std::size_t i = 0; i < mdims.size(); ++i) {
        if (i != query.dim) {
            head.index.insert(
                head.index.end(),
                mdims[i].begin(),
                mdims[i].end()
            );
        } else {
            head.index.push_back(mdims[i][query.idx]);
        }
    }

    /*
     * Record the shapes of the output. The first attribute is always 'data'
     * (the payload/values/trace values), and its shape always matches those of
     * the index. One of the dimensions is 1 (e.g. when querying an inline, the
     * first dimension is 1), so users with numpy probably wants to squeeze the
     * array before use. How to handle these 1-dimensions is left to the users.
     */
    auto& shapes = head.shapes;
    shapes.push_back(head.ndims);
    shapes.insert(
        shapes.end(),
        head.index.begin(),
        head.index.begin() + head.ndims
    );

    for (const auto& attr : query.attributes) {
        shapes.push_back(head.ndims);
        shapes.insert(
            shapes.end(),
            head.index.begin(),
            head.index.begin() + head.ndims
        );

        /*
         * If the query is vertical (in/crossline) then the attributes should
         * all be 1D arrays (one-per-trace). When it is a time/depth slice, the
         * output is a field and the attributes are 2D. This maps the attribute
         * shapes from/to:
         *
         * dim0: [1, N, M] -> [1, N, 1]
         * dim1: [N, 1, M] -> [N, 1, 1]
         * dim2: [N, M, 1] -> [N, M, 1]
         */
        shapes.back() = 1;
    }

    return head;
}

/*
 * The curtain query is expanded to the set of fragment IDs that contain all
 * the input traces, including z-axis. This for-purpose sorted-by-bucket-map
 * class helps maintain the invariant and provides simple interface for the
 * curtain.
 *
 * It inherits storage, clear, and move semantics from vector, but provides a
 * find() and at() for map lookup.
 *
 * The main achievement is getting all the hairy binary search out of the way,
 * and provide a more suitable abstraction for the build(curtain) function.
 */
struct flat_map : public std::vector< single > {
    using iterator = std::vector< single >::iterator;

    iterator at(FID< 3 > id) noexcept (true) {
        const auto less = [](const auto& lhs, const auto& rhs) noexcept {
            return std::lexicographical_compare(
                lhs.id.begin(),
                lhs.id.end(),
                rhs.begin(),
                rhs.end()
            );
        };

        const auto lessid = [](const auto& lhs, const auto& rhs) noexcept {
            return lhs.id < rhs.id;
        };

        assert(std::is_sorted(this->begin(), this->end(), lessid));
        return std::lower_bound(this->begin(), this->end(), id, less);
    }

    std::tuple< iterator, bool > find(FID< 3 > fid) noexcept (true) {
        const auto equal = [](const auto& lhs, const auto& rhs) noexcept {
            return std::equal(lhs.begin(), lhs.end(), rhs.begin());
        };
        auto itr   = this->at(fid);
        auto found = (itr != this->end()) and equal(itr->id, fid);
        return { itr, found };
    }
};

std::vector< curtain_task > build(const curtain_query& query) {
    std::vector< curtain_task > tasks;

    tasks.emplace_back(query);
    for (const auto& attr : query.attributes) {
        /*
         * It's perfectly common for queries to request attributes that aren't
         * recorded for a survey - in this case, silently drop it
         */
        auto [itr, found] = find_attribute(query, attr);
        if (not found)
            continue;

        tasks.emplace_back(query, *itr);
    }

    flat_map ids;
    for (auto& task : tasks) {
        ids.clear();
        const auto gvt = geometry(task);
        const auto zheight = gvt.fragment_count(gvt.mkdim(2));

        /*
         * Guess the number of coordinates per fragment. A reasonable
         * assumption is a plane going through a fragment, with a little bit of
         * margin. Not pre-reserving is perfectly fine, but we can save a bunch
         * of allocations in the average case by guessing well. It is
         * reasonably short-lived, so overestimating slightly should not be a
         * problem.
         */
        const auto approx_coordinates_per_fragment = int(
            1.2 * std::max(gvt.fragment_shape()[0], gvt.fragment_shape()[1])
        );

        for (int i = 0; i < int(query.dim0s.size()); ++i) {
            const auto top = top_cubepoint(query.dim0s, query.dim1s, i);
            const auto fid = gvt.frag_id(top);

            auto [itr, found] = ids.find(fid);
            if (not found) {
                /*
                 * Generate and insert all the fragments in this column.
                 * For attributes, zheight should be 1
                 */
                single block {};
                assert(fid.size() == block.id.size());
                std::copy_n(fid.begin(), block.id.size(), block.id.begin());
                block.coordinates.reserve(approx_coordinates_per_fragment);
                block.offset = i;
                itr = ids.insert(itr, zheight, block);
                for (int z = 0; z < zheight; ++z)
                    (itr + z)->id[2] = z;
            }

            const auto lid = coordinate(gvt.to_local(top));
            std::for_each(itr, itr + zheight, [lid](auto& block) {
                block.coordinates.push_back(lid);
            });
        }

        task.ids = std::move(ids);
    };

    return tasks;
}

process_header header( const curtain_query& query, int ntasks)
noexcept (false) {
    const auto& mdims = query.manifest.line_numbers;

    process_header head;
    head.pid        = query.pid;
    head.function   = functionid::curtain;
    head.nbundles   = ntasks;
    head.ndims      = mdims.size();
    head.labels     = query.manifest.line_labels;
    head.attributes.push_back("data");
    head.attributes.insert(
        head.attributes.end(),
        query.attributes.begin(),
        query.attributes.end()
    );

    auto& index = head.index;

    index.push_back(query.dim0s .size());
    index.push_back(query.dim1s .size());
    index.push_back(mdims.back().size());

    const auto& line_numbers = query.manifest.line_numbers;
    for (auto x : query.dim0s) index.push_back(line_numbers[0][x]);
    for (auto x : query.dim1s) index.push_back(line_numbers[1][x]);
    index.insert(index.end(), mdims.back().begin(), mdims.back().end());

    /*
     * The curtain is already pretty constrained in its output shapes, since it
     * can only query "vertically", which makes attributes always 1D
     */
    auto& shapes = head.shapes;
    shapes.push_back(2);
    shapes.insert(shapes.end(), index.begin() + 1, index.begin() + 3);

    for (const auto& attr : query.attributes) {
        shapes.push_back(1);
        shapes.push_back(head.index.front());
    }

    return head;
}



template< typename Outputs >
int count_tasks(const Outputs& outputs, int task_size) noexcept (true) {
    const auto add = [task_size](auto acc, const auto& elem) noexcept (true) {
        return acc + task_count(elem.ids.size(), task_size);
    };
    return std::accumulate(outputs.begin(), outputs.end(), 0, add);
}

/*
 * Partitions an Output in-place and pack()s it into blobs of task_size jobs.
 * It assumes the Output type has a vector-like member called 'ids'. This is a
 * name lookup - should the member be named something else or accessed in a
 * different way then you must implement a custom partition().
 *
 * This function shall return \0-separated packed tasks. The last task
 * shall be terminated with a \0, so that partition.count() can be
 * implemented as std::count(begin, end, '\0'). While it is a slightly
 * surprising interface (the most straight-forward would be
 * vector-of-string), it makes processing the set-of-tasks slightly easier,
 * saves a few allocations, and signals that the output is a bag of bytes.
 */
template < typename Output >
taskset partition(std::vector< Output >& outputs, int task_size)
noexcept (false) {
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

/*
 * Scheduling
 * ----------
 * Scheduling in this context means the process of:
 *   1. parse an incoming request
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
 * It relies on a few types, functions, and their contracts:
 *
 * Input (type)
 *  like the messages *_query family
 * Output (type)
 *  like the messages *_task family
 *  The Output type should have a pack() method that returns a std::string
 *
 * vector< Output > build(Input)
 *  Build the schedule - parse the incoming request and build the set of
 *  fragment IDs and extraction descriptions. This function is specific to
 *  the shape (slice, curtain, horizon etc) and comes with no default
 *  implementation.
 *
 * process_header header(Input, ntasks)
 *  Make a header. This function requires deep knowledge of the shape and
 *  oneseismic geometry, and must be implemented for all shape types.
 *
 *  The information in the process header is crucial for efficient and precise
 *  assembly of the end-result on the client side. Otherwise, clients must
 *  buffer the full response, then parse it and make sense of the shape, keys,
 *  linenos etc after the fact, since data can arrive in arbitrary order. This
 *  makes detecting errors more difficult too. The process header should
 *  provide enough information for clients to properly pre-allocate and build
 *  metadata to make sense of data as it is streamed.
 */

template < typename Input >
taskset schedule(Input& in, const char* doc, int len, int task_size)
noexcept (false) {
    in.unpack(doc, doc + len);
    in.attributes = normalized_attributes(in);
    auto fetch = build(in);
    auto sched = partition(fetch, task_size);
    const auto ntasks = int(sched.count());
    const auto head   = header(in, ntasks);
    sched.append(pack_with_envelope(head));
    return sched;
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
        slice_query q;
        return schedule(q, doc, len, task_size);
    }
    if (function == "curtain") {
        curtain_query q;
        return schedule(q, doc, len, task_size);
    }
    throw std::logic_error("No handler for function " + function);
}

}
