#include <algorithm>
#include <cassert>
#include <string>
#include <tuple>

#include <fmt/format.h>
#include <msgpack.hpp>
#include <nlohmann/json.hpp>

#include <oneseismic/messages.hpp>
#include <oneseismic/geometry.hpp>

namespace one {

/*
 * The go API server only sends plain-text messages as they're already tiny,
 * and contains no binary data. JSON is picked due to library support slightly
 * easier to pack/unpack, and far easier to inspect and debug.
 *
 * The Packable CRTP/mixin automates the generation of pack/unpack functions.
 */
template< typename T >
std::string Packable< T >::pack() const noexcept (false) {
    const auto& self = static_cast< const T& >(*this);
    return nlohmann::json(self).dump();
}

template< typename T >
void Packable< T >::unpack(const char* fst, const char* lst) noexcept (false) {
    const auto doc = nlohmann::json::parse(fst, lst);
    doc.get_to(static_cast< T& >(*this));
}

template< typename T >
std::string MsgPackable< T >::pack() const noexcept (false) {
    const auto& self = static_cast< const T& >(*this);
    const auto doc = nlohmann::json(self);
    const auto msg = nlohmann::json::to_msgpack(doc);
    return std::string(msg.begin(), msg.end());
}

template< typename T >
void MsgPackable< T >::unpack(const char* fst, const char* lst) noexcept (false) {
    auto& self = static_cast< T& >(*this);
    self = nlohmann::json::from_msgpack(fst, lst).get< T >();
}

namespace {

template < typename Packer, typename T >
void packarray_bin(Packer& packer, const std::vector< T >& vec)
noexcept (true) {
    const auto vsize = vec.size() * sizeof(T);
    const auto* body = reinterpret_cast< const char* >(vec.data());
    packer.pack_bin(vsize);
    packer.pack_bin_body(body, vsize);
}

}

std::string slice_tiles::pack() const noexcept (false) {
    msgpack::sbuffer buffer;
    msgpack::packer< decltype(buffer) > packer(buffer);

    /*
     * Pack the slice tiles as a tuple, which maps to the msgpack array, since
     * types are onto the value itself, not on the enclosing structure.
     *
     * This means consumers of the message must know the order of fields to
     * make sense of the message, but the space savings (and slightly lower
     * complexity of parsing) makes it worth it (compared to maps, this
     * typically reduces message size by half, which means 50% less network
     * traffic to users).
     */
    packer.pack_array(2);
    packer.pack(this->attr);
    packer.pack_array(tiles.size());
    for (const auto& tile : this->tiles) {
        packer.pack_array(6);
        packer.pack(tile.iterations);
        packer.pack(tile.chunk_size);
        packer.pack(tile.initial_skip);
        packer.pack(tile.superstride);
        packer.pack(tile.substride);
        packarray_bin(packer, tile.v);
    }

    return std::string(buffer.data(), buffer.size());
}

void ensurearray(const msgpack::v2::object& o) noexcept (false) {
    if (o.type != msgpack::v2::type::ARRAY) {
        const auto msg = fmt::format("expected array, was {}", o.type);
        throw std::logic_error(msg);
    }
};

void slice_tiles::unpack(const char* fst, const char* lst) noexcept (false) {
    /*
     * Unpack is a bit rough, but is only used for testing purposes
     */

    const auto result = msgpack::unpack(fst, std::distance(fst, lst));
    const auto& obj = result.get();

    ensurearray(obj);
    auto root  = obj.via.array.ptr;
    this->attr = root[0].as< std::string >();

    ensurearray(root[1]);
    const auto ntiles = root[1].via.array.size;
    const auto* tiles = root[1].via.array.ptr;

    this->tiles.clear();
    for (int i = 0; i < ntiles; ++i) {
        ensurearray(tiles[i]);
        auto ptile = tiles[i].via.array.ptr;
        tile t;
        t.iterations    = ptile[0].as< int >();
        t.chunk_size    = ptile[1].as< int >();
        t.initial_skip  = ptile[2].as< int >();
        t.superstride   = ptile[3].as< int >();
        t.substride     = ptile[4].as< int >();

        if (ptile[5].type != msgpack::v2::type::BIN)
            throw bad_value("tile.v should be BIN");
        auto tv = ptile[5].via.bin;
        t.v.resize(tv.size / sizeof(float));
        std::memcpy(t.v.data(), tv.ptr, tv.size);
        this->tiles.push_back(std::move(t));
    }
}

std::string curtain_bundle::pack() const noexcept (false) {
    msgpack::sbuffer buffer;
    msgpack::packer< decltype(buffer) > packer(buffer);

    packer.pack_array(5);
    packer.pack(this->attr);
    packer.pack(size);
    packer.pack(major);
    packer.pack(minor);
    packarray_bin(packer, this->values);
    return std::string(buffer.data(), buffer.size());
}

void curtain_bundle::unpack(const char* fst, const char* lst)
noexcept (false) {
    const auto result = msgpack::unpack(fst, std::distance(fst, lst));
    const auto& obj = result.get();
    ensurearray(obj);

    if (obj.via.array.size < 5)
        throw bad_message("expected array of len 5");

    obj.via.array.ptr[0] >> this->attr;
    obj.via.array.ptr[1] >> this->size;
    obj.via.array.ptr[2] >> this->major;
    obj.via.array.ptr[3] >> this->minor;

    auto tv = obj.via.array.ptr[4];
    if (tv.type != msgpack::v2::type::BIN)
        throw bad_value("curtain.values should be BIN");
    this->values.resize(tv.via.bin.size / sizeof(float));
    std::memcpy(this->values.data(), tv.via.bin.ptr, tv.via.bin.size);
}


void from_json(const nlohmann::json& doc, volumedesc& v) noexcept (false) {
    doc.at("prefix")        .get_to(v.prefix);
    doc.at("file-extension").get_to(v.ext);
    doc.at("shapes")        .get_to(v.shapes);
}

void to_json(nlohmann::json& doc, const volumedesc& v) noexcept (false) {
    doc["prefix"]         = v.prefix;
    doc["file-extension"] = v.ext;
    doc["shapes"]         = v.shapes;
}

void from_json(const nlohmann::json& doc, attributedesc& a) noexcept (false) {
    doc.at("prefix")        .get_to(a.prefix);
    doc.at("file-extension").get_to(a.ext);
    doc.at("type")          .get_to(a.type);
    doc.at("layout")        .get_to(a.layout);
    doc.at("labels")        .get_to(a.labels);
    doc.at("shapes")        .get_to(a.shapes);
}

void to_json(nlohmann::json& doc, const attributedesc& a) noexcept (false) {
    doc["prefix"]         = a.prefix;
    doc["file-extension"] = a.ext;
    doc["type"]           = a.type;
    doc["layout"]         = a.layout;
    doc["labels"]         = a.labels;
    doc["shapes"]         = a.shapes;
}

void from_json(const nlohmann::json& doc, manifestdoc& m) noexcept (false) {
    doc.at("line-numbers").get_to(m.line_numbers);
    doc.at("line-labels") .get_to(m.line_labels);
    doc.at("data")        .get_to(m.vol);
    doc.at("attributes")  .get_to(m.attr);
}

void to_json(nlohmann::json& doc, const manifestdoc& m) noexcept (false) {
    doc["line-numbers"] = m.line_numbers;
    doc["line-labels"]  = m.line_labels;
    doc["data"]         = m.vol;
    doc["attributes"]   = m.attr;
}

void to_json(nlohmann::json& doc, const basic_query& query) noexcept (false) {
    doc["pid"]              = query.pid;
    doc["token"]            = query.token;
    doc["guid"]             = query.guid;
    doc["manifest"]         = query.manifest;
    doc["storage_endpoint"] = query.storage_endpoint;
    doc["function"]         = query.function;
    doc["attributes"]       = query.attributes;
}

void from_json(const nlohmann::json& doc, basic_query& query) noexcept (false) {
    doc.at("pid")             .get_to(query.pid);
    doc.at("token")           .get_to(query.token);
    doc.at("guid")            .get_to(query.guid);
    doc.at("manifest")        .get_to(query.manifest);
    doc.at("storage_endpoint").get_to(query.storage_endpoint);
    doc.at("function")        .get_to(query.function);

    const auto optsitr = doc.find("opts");
    if (optsitr == doc.end()) return;

    const auto& opts = *optsitr;

    const auto attr = opts.find("attributes");
    if (attr != opts.end())
        attr->get_to(query.attributes);
}

void to_json(nlohmann::json& doc, const basic_task& task) noexcept (false) {
    doc["pid"]              = task.pid;
    doc["token"]            = task.token;
    doc["guid"]             = task.guid;
    doc["storage_endpoint"] = task.storage_endpoint;
    doc["prefix"]           = task.prefix;
    doc["ext"]              = task.ext;
    doc["shape"]            = task.shape;
    doc["shape-cube"]       = task.shape_cube;
    doc["function"]         = task.function;
    doc["attribute"]        = task.attribute;
    assert(task.shape_cube.size() == task.shape.size());
}

void from_json(const nlohmann::json& doc, basic_task& task) noexcept (false) {
    doc.at("pid")             .get_to(task.pid);
    doc.at("token")           .get_to(task.token);
    doc.at("guid")            .get_to(task.guid);
    doc.at("storage_endpoint").get_to(task.storage_endpoint);
    doc.at("prefix")          .get_to(task.prefix);
    doc.at("ext")             .get_to(task.ext);
    doc.at("shape")           .get_to(task.shape);
    doc.at("shape-cube")      .get_to(task.shape_cube);
    doc.at("function")        .get_to(task.function);
    doc.at("attribute")       .get_to(task.attribute);
}

void to_json(nlohmann::json& doc, const process_header& head) noexcept (false) {
    doc["pid"]          = head.pid;
    doc["function"]     = head.function;
    doc["nbundles"]     = head.nbundles;
    doc["ndims"]        = head.ndims;
    doc["index"]        = head.index;
    doc["labels"]       = head.labels;
    doc["shapes"]       = head.shapes;
    doc["attributes"]   = head.attributes;
}

void from_json(const nlohmann::json& doc, process_header& head) noexcept (false) {
    doc.at("pid")       .get_to(head.pid);
    doc.at("function")  .get_to(head.function);
    doc.at("nbundles")  .get_to(head.nbundles);
    doc.at("ndims")     .get_to(head.ndims);
    doc.at("index")     .get_to(head.index);
    doc.at("labels")    .get_to(head.labels);
    doc.at("shapes")    .get_to(head.shapes);
    doc.at("attributes").get_to(head.attributes);
}

void from_json(const nlohmann::json& doc, slice_query& query) noexcept (false) {
    from_json(doc, static_cast< basic_query& >(query));

    if (query.function != "slice") {
        const auto msg = "expected task 'slice', got {}";
        throw bad_message(fmt::format(msg, query.function));
    }

    const auto& lines = query.manifest.line_numbers;
    const auto& args = doc.at("args");

    args.at("dim").get_to(query.dim);
    if (!(0 <= query.dim && query.dim < lines.size())) {
        const auto msg = fmt::format(
            "args.dim (= {}) not in [0, {})",
            query.dim,
            lines.size()
        );
        throw not_found(msg);
    }

    const std::string& kind = args.at("kind");
    const int val = args.at("val");
    if (kind == "index") {
        query.idx = val;
    }
    else if (kind == "lineno") {
        const auto& index = lines[query.dim];
        const auto itr = std::find(index.begin(), index.end(), val);
        if (itr == index.end()) {
            const auto msg = "line (= {}) not found in index";
            throw not_found(fmt::format(msg, val));
        }
        query.idx = std::distance(index.begin(), itr);
    }
}

namespace {

/*
 * Compute the cartesian coordinate of the label/line numbers. This is
 * effectively a glorified indexof() in practice, although conceptually it maps
 * from a user-oriented grid to its internal representation. The cartesian
 * coordinates are taken at face value by the rest of the system, and can be
 * used for lookup directly. Past this point, oneseismic only works in the
 * cartesian grid and no longer cares about line numbers.
 */
void to_cartesian_inplace(
    const std::vector< int >& labels,
    std::vector< int >& xs)
noexcept (false) {
    assert(std::is_sorted(labels.begin(), labels.end()));

    auto fst = xs.begin();
    auto lst = xs.end();

    const auto indexof = [&labels](auto x) {
        const auto itr = std::lower_bound(labels.begin(), labels.end(), x);
        if (*itr != x) {
            const auto msg = fmt::format("lineno {} not in index");
            throw not_found(msg);
        }
        return std::distance(labels.begin(), itr);
    };

    std::transform(fst, lst, fst, indexof);
}

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

CP< 3 > top_point(const std::tuple< int, int > xy) noexcept (true) {
    return CP< 3 > {
        std::size_t(std::get< 0 >(xy)),
        std::size_t(std::get< 1 >(xy)),
        std::size_t(0),
    };
}

/*
 * Organise the x/y pairs so that all pairs that belong to the same vertical
 * fragment block are consecutive. This is important to make sure that all
 * traces in a fragment are handled in a single fragment fetch (the *by far*
 * most expensive operation in oneseismic).
 *
 * The function first groups the query by fragment-id, then by
 * fragment-point-id. The decoding is aware (and depends on) this with its
 * major/minor encoding of trace ID ranges.
 *
 * As a consequence, the response needs to be carefully evaluated by users,
 * since the traces may arrive in a different order than queried. This is
 * reasonable if you know that oneseismic will attempt to optimise the query,
 * but can easily catch users by surprise.
 */
void group_by_fragment_inplace(curtain_query& query) {
    auto& dim0s = query.dim0s;
    auto& dim1s = query.dim1s;

    /*
     * This uses a temporary vector-of-pairs to automatically get
     * lexicographical sorting for clarity and ease-of-implementation. This
     * does introduce an extra allocation and can be improved by sorting the
     * pair of vectors directly, but that's an optimisation for later.
     */
    std::vector< std::tuple< int, int > > pairs(query.dim0s.size());
    for (std::size_t i = 0; i < pairs.size(); ++i)
        pairs[i] = std::make_tuple(dim0s[i], dim1s[i]);

    const auto gvt = geometry(query);
    auto fragment_less = [&gvt] (const auto& lhs, const auto& rhs) noexcept {
        const auto left  = top_point(lhs);
        const auto right = top_point(rhs);
        return gvt.frag_id(left) < gvt.frag_id(right);
    };

    std::sort(pairs.begin(), pairs.end());
    std::stable_sort(pairs.begin(), pairs.end(), fragment_less);
    for (std::size_t i = 0; i < pairs.size(); ++i)
        std::tie(dim0s[i], dim1s[i]) = pairs[i];
}

}

void from_json(const nlohmann::json& doc, curtain_query& query) noexcept (false) {
    from_json(doc, static_cast< basic_query& >(query));

    if (query.function != "curtain") {
        const auto msg = "expected query 'curtain', got {}";
        throw bad_message(fmt::format(msg, query.function));
    }

    const auto& args = doc.at("args");

    std::vector< std::vector< int > > coords;
    args.at("coords").get_to(coords);
    query.dim0s.reserve(coords.size());
    query.dim1s.reserve(coords.size());

    try {
        for (const auto& pair : coords) {
            query.dim0s.push_back(pair.at(0));
            query.dim1s.push_back(pair.at(1));
        }
    } catch (std::out_of_range&) {
        throw bad_value("bad coord arg; expected list-of-pairs");
    }

    const std::string& kind = args.at("kind");
    if (kind == "index") {
        /* no-op - already cartesian indices */
    }
    else if (kind == "lineno") {
        const auto& line_numbers = query.manifest.line_numbers;
        to_cartesian_inplace(line_numbers[0], query.dim0s);
        to_cartesian_inplace(line_numbers[1], query.dim1s);
    }

    group_by_fragment_inplace(query);
}

void to_json(nlohmann::json& doc, const slice_task& task) noexcept (false) {
    to_json(doc, static_cast< const basic_task& >(task));
    doc["dim"] = task.dim;
    doc["idx"] = task.idx;
    doc["ids"] = task.ids;
}

void from_json(const nlohmann::json& doc, slice_task& task) noexcept (false) {
    from_json(doc, static_cast< basic_task& >(task));
    doc.at("dim").get_to(task.dim);
    doc.at("idx").get_to(task.idx);
    doc.at("ids").get_to(task.ids);

    if (task.ids.empty()) {
        /*
         * TODO:
         * Is this an error? Why is a request for zero fragments sent? It could
         * be silently discarded or properly logged, then discarded.
         *
         * Since everything eventually loops over this list of IDs, accepting
         * the message effectively silently discards it.
         */
        return;
    }

    const auto dims = task.ids.front().size();
    const auto same_size = [dims](const auto& id) noexcept (true) {
        return id.size() == dims;
    };

    if (!std::all_of(task.ids.begin(), task.ids.end(), same_size)) {
        throw bad_message("inconsistent dimensions");
    }
}

void to_json(nlohmann::json& doc, const tile& tile) noexcept (false) {
    doc["iterations"]   = tile.iterations;
    doc["chunk-size"]   = tile.chunk_size;
    doc["initial-skip"] = tile.initial_skip;
    doc["superstride"]  = tile.superstride;
    doc["substride"]    = tile.substride;
    doc["v"]            = tile.v;
}

void from_json(const nlohmann::json& doc, tile& tile) noexcept (false) {
    doc.at("iterations")  .get_to(tile.iterations);
    doc.at("chunk-size")  .get_to(tile.chunk_size);
    doc.at("initial-skip").get_to(tile.initial_skip);
    doc.at("superstride") .get_to(tile.superstride);
    doc.at("substride")   .get_to(tile.substride);
    doc.at("v")           .get_to(tile.v);
}

void to_json(nlohmann::json& doc, const slice_tiles& tiles) noexcept (false) {
    doc["attribute"]  = tiles.attr;
    doc["tiles"]      = tiles.tiles;
}

void from_json(const nlohmann::json& doc, slice_tiles& tiles) noexcept (false) {
    doc.at("attribute").get_to(tiles.attr);
    doc.at("tiles")    .get_to(tiles.tiles);
}

void to_json(nlohmann::json& doc, const single& single) noexcept (false) {
    doc["id"]          = single.id;
    doc["offset"]      = single.offset;
    doc["coordinates"] = single.coordinates;
}

void from_json(const nlohmann::json& doc, single& single) noexcept (false) {
    doc.at("id")         .get_to(single.id);
    doc.at("offset")     .get_to(single.offset);
    doc.at("coordinates").get_to(single.coordinates);
}

void to_json(nlohmann::json& doc, const curtain_task& curtain) noexcept (false) {
    to_json(doc, static_cast< const basic_task& >(curtain));
    doc["ids"] = curtain.ids;
}

void from_json(const nlohmann::json& doc, curtain_task& curtain) noexcept (false) {
    from_json(doc, static_cast< basic_task& >(curtain));
    doc.at("ids").get_to(curtain.ids);
}

/*
 * Explicitly instantiate classes with the packable interface, in order to
 * generate the pack()/unpack() code. The functions are defined and
 * instantiated here in order to avoid leaking nlohmann/json into the public
 * interface, which would require go (and other dependencies) to be aware of
 * it.
 */
template struct Packable< slice_query >;
template struct Packable< slice_task >;
template struct Packable< curtain_query >;
template struct Packable< curtain_task >;

template struct MsgPackable< process_header >;

}
