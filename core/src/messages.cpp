#include <string>

#include <fmt/format.h>
#include <nlohmann/json.hpp>

#include <oneseismic/messages.hpp>

namespace one {

void to_json(nlohmann::json& doc, const common_task& task) noexcept (false) {
    assert(task.shape_cube.size() == task.shape.size());
    doc["pid"]              = task.pid;
    doc["token"]            = task.token;
    doc["guid"]             = task.guid;
    doc["manifest"]         = task.manifest;
    doc["storage_endpoint"] = task.storage_endpoint;
    doc["shape"]            = task.shape;
    doc["shape-cube"]       = task.shape_cube;
    doc["function"]         = task.function;
}

void from_json(const nlohmann::json& doc, common_task& task) noexcept (false) {
    doc.at("pid")             .get_to(task.pid);
    doc.at("token")           .get_to(task.token);
    doc.at("guid")            .get_to(task.guid);
    doc.at("manifest")        .get_to(task.manifest);
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

void to_json(nlohmann::json& doc, const slice_task& task) noexcept (false) {
    to_json(doc, static_cast< const common_task& >(task));
    doc["function"] = "slice";
    auto& params = doc["params"];
    params["dim"]    = task.dim;
    params["lineno"] = task.lineno;
}

void from_json(const nlohmann::json& doc, slice_task& task) noexcept (false) {
    from_json(doc, static_cast< common_task& >(task));

    if (task.function != "slice") {
        const auto msg = "expected task 'slice', got {}";
        throw bad_message(fmt::format(msg, task.function));
    }

    const auto& params = doc.at("params");
    params.at("dim")   .get_to(task.dim);
    params.at("lineno").get_to(task.lineno);
}

void to_json(nlohmann::json& doc, const curtain_task& task) noexcept (false) {
    to_json(doc, static_cast< const common_task& >(task));
    doc["function"] = "curtain";
    auto& params = doc["params"];
    params["dim0s"] = task.dim0s;
    params["dim1s"] = task.dim1s;
}

void from_json(const nlohmann::json& doc, curtain_task& task) noexcept (false) {
    from_json(doc, static_cast< common_task& >(task));

    if (task.function != "curtain") {
        const auto msg = "expected task 'curtain', got {}";
        throw bad_message(fmt::format(msg, task.function));
    }

    const auto& params = doc.at("params");
    params.at("dim0s").get_to(task.dim0s);
    params.at("dim1s").get_to(task.dim1s);
}

void to_json(nlohmann::json& doc, const slice_fetch& task) noexcept (false) {
    to_json(doc, static_cast< const slice_task& >(task));
    doc["ids"]        = task.ids;
}

void from_json(const nlohmann::json& doc, slice_fetch& task) noexcept (false) {
    from_json(doc, static_cast< slice_task& >(task));
    doc.at("ids")       .get_to(task.ids);

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

void to_json(nlohmann::json& doc, const curtain_fetch& curtain) noexcept (false) {
    to_json(doc, static_cast< const curtain_task& >(curtain));
    doc["ids"] = curtain.ids;
}

void from_json(const nlohmann::json& doc, curtain_fetch& curtain) noexcept (false) {
    from_json(doc, static_cast< curtain_task& >(curtain));
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

/*
 * The go API server only sends plain-text messages as they're already tiny,
 * and contains no binary data. JSON is picked due to library support slightly
 * easier to pack/unpack, and far easier to inspect and debug.
 */
void common_task::unpack(const char* fst, const char* lst) noexcept (false) {
    const auto doc = nlohmann::json::parse(fst, lst);
    *this = doc.get< common_task >();
}

std::string common_task::pack() const {
    return nlohmann::json(*this).dump();
}

void process_header::unpack(const char* fst, const char* lst) noexcept (false) {
    const auto doc = nlohmann::json::parse(fst, lst);
    *this = doc.get< process_header >();
}

std::string process_header::pack() const {
    return nlohmann::json(*this).dump();
}

void slice_task::unpack(const char* fst, const char* lst) noexcept (false) {
    const auto doc = nlohmann::json::parse(fst, lst);
    *this = doc.get< slice_task >();
}

std::string slice_task::pack() const {
    return nlohmann::json(*this).dump();
}

void slice_fetch::unpack(const char* fst, const char* lst) noexcept (false) {
    const auto doc = nlohmann::json::parse(fst, lst);
    *this = doc.get< slice_fetch >();
}

std::string slice_fetch::pack() const {
    return nlohmann::json(*this).dump();
}

void slice_tiles::unpack(const char* fst, const char* lst) noexcept (false) {
    *this = nlohmann::json::from_msgpack(fst, lst).get< slice_tiles >();
}

std::string slice_tiles::pack() const {
    const auto doc = nlohmann::json(*this);
    const auto msg = nlohmann::json::to_msgpack(doc);
    return std::string(msg.begin(), msg.end());
}

void curtain_fetch::unpack(const char* fst, const char* lst) noexcept (false) {
    const auto doc = nlohmann::json::parse(fst, lst);
    *this = doc.get< curtain_fetch >();
}

std::string curtain_fetch::pack() const {
    return nlohmann::json(*this).dump();
}

void curtain_task::unpack(const char* fst, const char* lst) noexcept (false) {
    const auto doc = nlohmann::json::parse(fst, lst);
    *this = doc.get< curtain_task >();
}

std::string curtain_task::pack() const {
    return nlohmann::json(*this).dump();
}

void curtain_traces::unpack(const char* fst, const char* lst) noexcept (false) {
    *this = nlohmann::json::from_msgpack(fst, lst).get< curtain_traces >();
}

std::string curtain_traces::pack() const {
    const auto doc = nlohmann::json(*this);
    const auto msg = nlohmann::json::to_msgpack(doc);
    return std::string(msg.begin(), msg.end());
}

}
