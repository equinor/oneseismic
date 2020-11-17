#include <cstring>
#include <string>

#include <catch/catch.hpp>
#include <fmt/format.h>

#include <oneseismic/messages.hpp>

using namespace Catch::Matchers;

namespace {

bool operator == (const one::common_task& lhs, const one::common_task& rhs) {
    return lhs.token            == rhs.token
        && lhs.guid             == rhs.guid
        && lhs.storage_endpoint == rhs.storage_endpoint
        && lhs.shape            == rhs.shape
        && lhs.function         == rhs.function
    ;
}

bool operator == (const one::slice_task& lhs, const one::slice_task& rhs) {
    return static_cast< const one::common_task& >(lhs) == rhs
        && lhs.dim    == rhs.dim
        && lhs.lineno == rhs.lineno
    ;
}

}

TEST_CASE("well-formed slice-task is unpacked correctly") {
    const auto doc = R"({
        "token": "on-behalf-of-token",
        "guid": "object-id",
        "storage_endpoint": "https://storage.com",
        "manifest": "{}",
        "shape": [64, 64, 64],
        "function": "slice",
        "params": {
            "dim": 0,
            "lineno": 10
        }
    })";

    one::slice_task task;
    task.unpack(doc, doc + std::strlen(doc));
    CHECK(task.token == "on-behalf-of-token");
    CHECK(task.guid  == "object-id");
    CHECK(task.manifest == "{}");
    CHECK(task.storage_endpoint == "https://storage.com");
    CHECK_THAT(task.shape, Equals(std::vector< int >{64, 64, 64}));
    CHECK(task.dim == 0);
    CHECK(task.lineno == 10);
}

TEST_CASE("unpacking task with missing field fails") {
    const auto entries = std::vector< std::string > {
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
            one::common_task task;
            CHECK_THROWS(task.unpack(fst, lst));
        }

    }
}

TEST_CASE("unpacking message with wrong function tag fails") {
    const auto doc = R"({
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

    one::slice_task task;
    CHECK_THROWS(task.unpack(doc, doc + sizeof(doc)));
}

TEST_CASE("slice-task can round trip packing") {
    one::slice_task task;
    task.token = "token";
    task.guid = "guid";
    task.manifest = "{}";
    task.storage_endpoint = "https://storage.com";
    task.shape = { 64, 64, 64 };
    task.function = "slice";
    task.dim = 1;
    task.lineno = 2;

    const auto packed = task.pack();
    one::slice_task unpacked;
    unpacked.unpack(packed.data(), packed.data() + packed.size());

    CHECK(task == unpacked);
}

TEST_CASE("slice-task sets function to 'slice'") {
    one::slice_task task;
    task.token = "token";
    task.guid = "guid";
    task.manifest = "{}";
    task.storage_endpoint = "https://storage.com";
    task.shape = { 64, 64, 64 };
    task.function = "garbage";
    task.dim = 1;
    task.lineno = 2;

    const auto packed = task.pack();
    one::slice_task unpacked;
    unpacked.unpack(packed.data(), packed.data() + packed.size());
    CHECK(unpacked.function == "slice");
}

TEST_CASE("slice-fetch can round trip packing") {
    one::slice_fetch task;
    task.token = "token";
    task.guid = "guid";
    task.manifest = "{}";
    task.storage_endpoint = "https://storage.com";
    task.shape = { 64, 64, 64 };
    task.function = "slice";
    task.dim = 1;
    task.lineno = 2;
    task.cube_shape = { 128, 128, 128 };
    task.ids = {
        { 0, 1, 2 },
        { 3, 4, 5 },
    };

    const auto packed = task.pack();
    one::slice_fetch unpacked;
    unpacked.unpack(packed.data(), packed.data() + packed.size());

    CHECK(task == unpacked);
}
