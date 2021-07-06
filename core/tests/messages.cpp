#include <algorithm>
#include <cstring>
#include <string>

#include <catch/catch.hpp>
#include <fmt/format.h>

#include <oneseismic/messages.hpp>

using namespace Catch::Matchers;

namespace {

bool operator == (const one::basic_task& lhs, const one::basic_task& rhs) {
    return lhs.pid              == rhs.pid
        && lhs.token            == rhs.token
        && lhs.guid             == rhs.guid
        && lhs.storage_endpoint == rhs.storage_endpoint
        && lhs.shape            == rhs.shape
        && lhs.function         == rhs.function
    ;
}

bool operator == (const one::slice_task& lhs, const one::slice_task& rhs) {
    return static_cast< const one::basic_task& >(lhs) == rhs
        && lhs.dim == rhs.dim
        && lhs.idx == rhs.idx
        && lhs.ids == rhs.ids
    ;
}

bool operator == (const one::volumedesc& lhs, const one::volumedesc& rhs) {
    return lhs.prefix == rhs.prefix
        && lhs.ext    == rhs.ext
        && lhs.shapes == rhs.shapes
    ;
}

bool operator == (const one::attributedesc& lhs, const one::attributedesc& rhs) {
    return lhs.prefix       == rhs.prefix
        && lhs.ext          == rhs.ext
        && lhs.type         == rhs.type
        && lhs.layout       == rhs.layout
        && lhs.labels       == rhs.labels
        && lhs.shapes       == rhs.shapes
    ;
}


bool operator == (const one::manifestdoc& lhs, const one::manifestdoc& rhs) {
    /*
     * Silly hack to make the vector operator == find the (unnamed) operator ==
     * defined here.
     */
    const auto eq = [](const auto& l, const auto& r) {
        return l == r;
    };

    if (!std::equal(lhs.vol.begin(), lhs.vol.end(), rhs.vol.begin(), eq))
        return false;
    if (!std::equal(lhs.attr.begin(), lhs.attr.end(), rhs.attr.begin(), eq))
        return false;

    return lhs.line_numbers == rhs.line_numbers
        && lhs.line_labels  == rhs.line_labels
    ;
}

}

TEST_CASE("well-formed slice-query is unpacked correctly") {
    const auto doc = R"({
        "pid": "some-pid",
        "token": "on-behalf-of-token",
        "guid": "object-id",
        "storage_endpoint": "https://storage.com",
        "manifest": {
            "format-version": 1,
            "data": [
                {
                    "file-extension": "f32",
                    "shapes": [[1]],
                    "prefix": "prefix",
                    "resolution": "source"
                }
            ],
            "attributes": [],
            "line-numbers": [[10]],
            "line-labels": ["dim-0"]
        },
        "shape": [64, 64, 64],
        "function": "slice",
        "args": {
            "kind": "lineno",
            "dim": 0,
            "val": 10
        }
    })";

    one::volumedesc vol;
    vol.prefix = "prefix";
    vol.ext = "f32";
    vol.shapes = { { 1 } };

    one::manifestdoc manifest;
    manifest.vol.push_back(vol);
    manifest.line_numbers = { { 10 } };
    manifest.line_labels = { "dim-0" };

    one::slice_query query;
    query.unpack(doc, doc + std::strlen(doc));
    CHECK(query.pid   == "some-pid");
    CHECK(query.token == "on-behalf-of-token");
    CHECK(query.guid  == "object-id");
    CHECK(query.manifest == manifest);
    CHECK(query.storage_endpoint == "https://storage.com");
    CHECK_THAT(query.shape,      Equals(std::vector< int >{ 64,  64,  64}));
    CHECK(query.dim == 0);
    CHECK(query.idx == 0);
}

TEST_CASE("unpacking query with missing field fails") {
    const auto entries = std::vector< std::string > {
        R"("pid": "some-pid")",
        R"("token": "on-behalf-of-token")",
        R"("guid": "object-id")",
        R"("manifest": { "dimensions": [[]] })",
        R"("storage_endpoint": "http://storage.com")",
        R"("shape": [64, 64, 64])",
        R"("function": "slice")",
    };

    for (std::size_t i = 0; i < entries.size(); ++i) {
        auto parts = entries;
        const auto key = parts[i].substr(0, parts[i].find(":"));
        SECTION(fmt::format("when key {} is missing", key)) {
            parts.erase(parts.begin() + i);
            const auto doc = fmt::format("{{\n{}\n}}", fmt::join(parts, ",\n"));
            const auto fst = doc.data();
            const auto lst = doc.data() + doc.size();
            one::slice_query query;
            CHECK_THROWS(query.unpack(fst, lst));
        }

    }
}

TEST_CASE("unpacking message with wrong function tag fails") {
    const auto doc = R"({
        "pid": "some-pid",
        "token": "on-behalf-of-token",
        "guid": "object-id",
        "manifest": { "dimensions": [[]] },
        "storage_endpoint": "https://storage.com",
        "shape": [64, 64, 64],
        "function": "broken",
        "args": {
            "dim": 0,
            "lineno": 10
        }
    })";

    one::slice_query query;
    CHECK_THROWS(query.unpack(doc, doc + sizeof(doc)));
}

TEST_CASE("slice-task can round trip packing") {
    one::slice_task task;
    task.pid = "pid";
    task.token = "token";
    task.guid = "guid";
    task.storage_endpoint = "https://storage.com";
    task.shape = { 64, 64, 64 };
    task.shape_cube = { 512, 512, 512 };
    task.function = "slice";
    task.dim = 1;
    task.idx = 2;
    task.ids = {
        { 0, 1, 2 },
        { 3, 4, 5 },
    };

    const auto packed = task.pack();
    INFO(packed);
    one::slice_task unpacked;
    unpacked.unpack(packed.data(), packed.data() + packed.size());

    CHECK(task == unpacked);
}
