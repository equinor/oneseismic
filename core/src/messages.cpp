#include <string>

#include <fmt/format.h>
#include <nlohmann/json.hpp>

#include <oneseismic/messages.hpp>

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
    auto& self = static_cast< T& >(*this);
    self = doc.get< T >();
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

/*
 * Explicitly instantiate classes with the packable interface, in order to
 * generate the pack()/unpack() code. The functions are defined and
 * instantiated here in order to avoid leaking nlohmann/json into the public
 * interface, which would require go (and other dependencies) to be aware of
 * it.
 */
template class Packable< process_header >;
template class Packable< slice_query >;
template class Packable< slice_task >;
template class Packable< curtain_query >;
template class Packable< curtain_task >;

template class MsgPackable< slice_tiles >;
template class MsgPackable< curtain_traces >;

void to_json(nlohmann::json& doc, const manifestdoc& m) noexcept (false) {
    doc["dimensions"] = m.dimensions;
}

void from_json(const nlohmann::json& doc, manifestdoc& m) noexcept (false) {
    doc.at("dimensions").get_to(m.dimensions);
}

void to_json(nlohmann::json& doc, const basic_query& query) noexcept (false) {
    doc["pid"]              = query.pid;
    doc["token"]            = query.token;
    doc["guid"]             = query.guid;
    doc["manifest"]         = query.manifest;
    doc["storage_endpoint"] = query.storage_endpoint;
    doc["shape"]            = query.shape;
    doc["function"]         = query.function;
}

void from_json(const nlohmann::json& doc, basic_query& query) noexcept (false) {
    doc.at("pid")             .get_to(query.pid);
    doc.at("token")           .get_to(query.token);
    doc.at("guid")            .get_to(query.guid);
    doc.at("manifest")        .get_to(query.manifest);
    doc.at("storage_endpoint").get_to(query.storage_endpoint);
    doc.at("shape")           .get_to(query.shape);
    doc.at("function")        .get_to(query.function);
}

void to_json(nlohmann::json& doc, const basic_task& task) noexcept (false) {
    doc["pid"]              = task.pid;
    doc["token"]            = task.token;
    doc["guid"]             = task.guid;
    doc["storage_endpoint"] = task.storage_endpoint;
    doc["shape"]            = task.shape;
    doc["shape-cube"]       = task.shape_cube;
    doc["function"]         = task.function;
    assert(task.shape_cube.size() == task.shape.size());
}

void from_json(const nlohmann::json& doc, basic_task& task) noexcept (false) {
    doc.at("pid")             .get_to(task.pid);
    doc.at("token")           .get_to(task.token);
    doc.at("guid")            .get_to(task.guid);
    doc.at("storage_endpoint").get_to(task.storage_endpoint);
    doc.at("shape")           .get_to(task.shape);
    doc.at("shape-cube")      .get_to(task.shape_cube);
    doc.at("function")        .get_to(task.function);
}

void to_json(nlohmann::json& doc, const process_header& head) noexcept (false) {
    doc["pid"]    = head.pid;
    doc["ntasks"] = head.ntasks;
    doc["shape"]  = head.shape;
    doc["index"]  = head.index;
}

void from_json(const nlohmann::json& doc, process_header& head) noexcept (false) {
    doc.at("pid")   .get_to(head.pid);
    doc.at("ntasks").get_to(head.ntasks);
    doc.at("shape") .get_to(head.shape);
    doc.at("index") .get_to(head.index);
}

void to_json(nlohmann::json& doc, const slice_query& query) noexcept (false) {
    to_json(doc, static_cast< const basic_query& >(query));
    doc["function"] = "slice";
    auto& params = doc["params"];
    params["dim"]    = query.dim;
    params["lineno"] = query.lineno;
}

void from_json(const nlohmann::json& doc, slice_query& query) noexcept (false) {
    from_json(doc, static_cast< basic_query& >(query));

    if (query.function != "slice") {
        const auto msg = "expected task 'slice', got {}";
        throw bad_message(fmt::format(msg, query.function));
    }

    const auto& params = doc.at("params");
    params.at("dim")   .get_to(query.dim);
    params.at("lineno").get_to(query.lineno);
}

void from_json(const nlohmann::json& doc, curtain_query& query) noexcept (false) {
    from_json(doc, static_cast< basic_query& >(query));

    if (query.function != "curtain") {
        const auto msg = "expected query 'curtain', got {}";
        throw bad_message(fmt::format(msg, query.function));
    }

    const auto& params = doc.at("params");

    std::vector< std::vector< int > > coords;
    params.at("coords").get_to(coords);
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
}

void to_json(nlohmann::json& doc, const slice_task& task) noexcept (false) {
    to_json(doc, static_cast< const basic_task& >(task));
    doc["dim"]    = task.dim;
    doc["lineno"] = task.lineno;
    doc["ids"]    = task.ids;
}

void from_json(const nlohmann::json& doc, slice_task& task) noexcept (false) {
    from_json(doc, static_cast< basic_task& >(task));
    doc.at("dim")   .get_to(task.dim);
    doc.at("lineno").get_to(task.lineno);
    doc.at("ids")   .get_to(task.ids);

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
    doc["shape"] = tiles.shape;
    doc["tiles"] = tiles.tiles;
}

void from_json(const nlohmann::json& doc, slice_tiles& tiles) noexcept (false) {
    doc.at("shape").get_to(tiles.shape);
    doc.at("tiles").get_to(tiles.tiles);
}

void to_json(nlohmann::json& doc, const single& single) noexcept (false) {
    doc["id"]          = single.id;
    doc["coordinates"] = single.coordinates;
}

void from_json(const nlohmann::json& doc, single& single) noexcept (false) {
    doc.at("id")         .get_to(single.id);
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

void to_json(nlohmann::json& doc, const trace& trace) noexcept (false) {
    doc["coordinates"] = trace.coordinates;
    doc["v"]           = trace.v;
}

void from_json(const nlohmann::json& doc, trace& trace) noexcept (false) {
    doc.at("coordinates").get_to(trace.coordinates);
    doc.at("v")          .get_to(trace.v);
}

void to_json(nlohmann::json& doc, const curtain_traces& traces) noexcept (false) {
    doc["traces"] = traces.traces;
}

void from_json(const nlohmann::json& doc, curtain_traces& traces) noexcept (false) {
    doc.at("traces").get_to(traces.traces);
}

}
