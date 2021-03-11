#include <catch/catch.hpp>

#include <oneseismic/geometry.hpp>
#include <oneseismic/messages.hpp>
#include <oneseismic/process.hpp>

one::slice_fetch default_slice_fetch() {
    one::slice_fetch input;
    input.pid   = "some-pid";
    input.token = "some-token";
    input.guid  = "some-guid";

    input.storage_endpoint = "some-endpoint";
    input.shape      = { 64, 64, 64 };
    input.shape_cube = { 720, 860, 251 };

    input.dim    = 0;
    input.lineno = 0;
    return input;
}

TEST_CASE("Slice IDs are generated from a task") {
    auto input = default_slice_fetch();
    auto slice = one::proc::make("slice");

    SECTION("None when ids are empty") {
        input.ids = {};
        const auto msg = input.pack();
        slice->init(msg.data(), msg.size());
        CHECK(slice->fragments() == "");
    }

    SECTION("Without delimiter when there is 1 id") {
        input.ids = {
            { 0, 1, 2 },
        };
        const auto msg = input.pack();
        slice->init(msg.data(), msg.size());
        CHECK(slice->fragments() == "src/64-64-64/0-1-2.f32");
    }

    SECTION("Delimited when there are many ids") {
        input.ids = {
            { 0, 1, 2 },
            { 2, 1, 1 },
        };
        const auto msg = input.pack();
        slice->init(msg.data(), msg.size());
        const auto expected =
            "src/64-64-64/0-1-2.f32" ";"
            "src/64-64-64/2-1-1.f32"
        ;
        CHECK(slice->fragments() == expected);
    }
}

TEST_CASE("All process kinds can be constructed") {
    CHECK( one::proc::make("slice"));
    CHECK(!one::proc::make("unknown"));
}
