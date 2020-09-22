#include <string>

#include <fmt/format.h>
#include <nlohmann/json.hpp>

#include <oneseismic/messages.hpp>

namespace one {

void to_json(nlohmann::json& doc, const common_task& task) noexcept (false) {
    doc["token"]            = task.token;
    doc["guid"]             = task.guid;
    doc["storage_endpoint"] = task.storage_endpoint;
    doc["shape"]            = task.shape;
    doc["function"]         = task.function;
}

void from_json(const nlohmann::json& doc, common_task& task) noexcept (false) {
    doc.at("token")           .get_to(task.token);
    doc.at("guid")            .get_to(task.guid);
    doc.at("storage_endpoint").get_to(task.storage_endpoint);
    doc.at("shape")           .get_to(task.shape);
    doc.at("function")        .get_to(task.function);
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

void to_json(nlohmann::json& doc, const slice_fetch& task) noexcept (false) {
    to_json(doc, static_cast< const slice_task& >(task));
    assert(task.cube_shape.size() == task.shape.size());
    doc["cube-shape"] = task.cube_shape;
    doc["ids"]        = task.ids;
}

void from_json(const nlohmann::json& doc, slice_fetch& task) noexcept (false) {
    from_json(doc, static_cast< slice_task& >(task));
    doc.at("cube-shape").get_to(task.cube_shape);
    doc.at("ids")       .get_to(task.ids);

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

}
