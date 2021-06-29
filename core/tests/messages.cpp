#include <cstring>
#include <string>

#include <catch/catch.hpp>
#include <fmt/format.h>

#include <oneseismic/messages.hpp>

using namespace Catch::Matchers;

namespace {

bool operator == (const one::basic_query& lhs, const one::basic_query& rhs) {
    return lhs.token            == rhs.token
        && lhs.guid             == rhs.guid
        && lhs.storage_endpoint == rhs.storage_endpoint
        && lhs.shape            == rhs.shape
        && lhs.function         == rhs.function
    ;
}

bool operator == (const one::slice_query& lhs, const one::slice_query& rhs) {
    return static_cast< const one::basic_query& >(lhs) == rhs
        && lhs.dim    == rhs.dim
        && lhs.lineno == rhs.lineno
    ;
}

bool operator == (const one::basic_task& lhs, const one::basic_task& rhs) {
    return lhs.pid              == rhs.pid
        && lhs.token            == rhs.token
        && lhs.guid             == rhs.guid
        && lhs.storage_endpoint == rhs.storage_endpoint
        && lhs.shape            == rhs.shape
        && lhs.shape_cube       == rhs.shape_cube
        && lhs.function         == rhs.function
    ;
}

bool operator == (const one::slice_task& lhs, const one::slice_task& rhs) {
    return static_cast< const one::basic_task& >(lhs) == rhs
        && lhs.dim    == rhs.dim
        && lhs.lineno == rhs.lineno
        && lhs.ids    == rhs.ids
    ;
}

}

TEST_CASE("well-formed slice-query is unpacked correctly") {
    const auto doc = R"({
        "pid": "some-pid",
        "token": "on-behalf-of-token",
        "guid": "object-id",
        "storage_endpoint": "https://storage.com",
        "manifest": "{}",
        "shape": [64, 64, 64],
        "shape-cube": [128, 128, 128],
        "function": "slice",
        "params": {
            "dim": 0,
            "lineno": 10
        }
    })";

    one::slice_query query;
    query.unpack(doc, doc + std::strlen(doc));
    CHECK(query.pid   == "some-pid");
    CHECK(query.token == "on-behalf-of-token");
    CHECK(query.guid  == "object-id");
    CHECK(query.manifest == "{}");
    CHECK(query.storage_endpoint == "https://storage.com");
    CHECK_THAT(query.shape,      Equals(std::vector< int >{ 64,  64,  64}));
    CHECK_THAT(query.shape_cube, Equals(std::vector< int >{128, 128, 128}));
    CHECK(query.dim == 0);
    CHECK(query.lineno == 10);
}

TEST_CASE("unpacking query with missing field fails") {
    const auto entries = std::vector< std::string > {
        R"("pid": "some-pid")",
        R"("token": "on-behalf-of-token")",
        R"("guid": "object-id")",
        R"("manifest": "{}")",
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
            one::basic_query query;
            CHECK_THROWS(query.unpack(fst, lst));
        }

    }
}

TEST_CASE("unpacking message with wrong function tag fails") {
    const auto doc = R"({
        "pid": "some-pid",
        "token": "on-behalf-of-token",
        "guid": "object-id",
        "manifest": "{}",
        "storage_endpoint": "https://storage.com",
        "shape": [64, 64, 64],
        "function": "broken",
        "params": {
            "dim": 0,
            "lineno": 10
        }
    })";

    one::slice_query query;
    CHECK_THROWS(query.unpack(doc, doc + sizeof(doc)));
}

TEST_CASE("slice-query can round trip packing") {
    one::slice_query query;
    query.pid = "pid";
    query.token = "token";
    query.guid = "guid";
    query.manifest = "{}";
    query.storage_endpoint = "https://storage.com";
    query.shape = { 64, 64, 64 };
    query.shape_cube = { 128, 128, 128 };
    query.function = "slice";
    query.dim = 1;
    query.lineno = 2;

    const auto packed = query.pack();
    one::slice_query unpacked;
    unpacked.unpack(packed.data(), packed.data() + packed.size());

    CHECK(query == unpacked);
}

TEST_CASE("slice-query sets function to 'slice'") {
    one::slice_query query;
    query.pid = "pid";
    query.token = "token";
    query.guid = "guid";
    query.manifest = "{}";
    query.storage_endpoint = "https://storage.com";
    query.shape = { 64, 64, 64 };
    query.shape_cube = { 128, 128, 128 };
    query.function = "garbage";
    query.dim = 1;
    query.lineno = 2;

    const auto packed = query.pack();
    one::slice_query unpacked;
    unpacked.unpack(packed.data(), packed.data() + packed.size());
    CHECK(unpacked.function == "slice");
}

TEST_CASE("slice-task can round trip packing") {
    one::slice_task task;
    task.pid = "pid";
    task.token = "token";
    task.guid = "guid";
    task.storage_endpoint = "https://storage.com";
    task.shape = { 64, 64, 64 };
    task.shape_cube = { 128, 128, 128 };
    task.function = "slice";
    task.dim = 1;
    task.lineno = 2;
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
