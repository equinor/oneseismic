#include <catch/catch.hpp>

#include <oneseismic/geometry.hpp>
#include <oneseismic/messages.hpp>
#include <oneseismic/process.hpp>

using namespace Catch::Matchers;

one::slice_task default_slice_task() {
    one::slice_task input;
    input.pid    = "some-pid";
    input.token  = "some-token";
    input.guid   = "some-guid";
    input.prefix = "src";
    input.ext    = "f32";

    input.storage_endpoint = "some-endpoint";
    input.shape      = { 64, 64, 64 };
    input.shape_cube = { 720, 860, 251 };

    input.dim = 0;
    input.idx = 0;
    return input;
}

TEST_CASE("slice.fragments generates the right IDs from a task") {
    auto input = default_slice_task();
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

/*
 * Manually extract the dim1 (crossline) to serve as reference data. While the
 * data is random, two independent implementations corresponding gives high
 * confidence.
 */
void add_dim1_line(
    std::vector< float >& out,
    const std::vector< float >& chunk
) {
    const int len = 3;
    out.insert(out.end(), chunk.data() +  3, chunk.data() +  3 + len);
    out.insert(out.end(), chunk.data() + 12, chunk.data() + 12 + len);
    out.insert(out.end(), chunk.data() + 21, chunk.data() + 21 + len);
}

template < typename T >
T unpack(const std::string& s) {
    T t;
    t.unpack(s.data(), s.data() + s.size());
    return t;
}

/*
 * This test should could be extended implemented to test more directions,
 * shape combinations etc.
 */
TEST_CASE("Slices extracted from chunks matches hand-extracted slice") {
    auto input = default_slice_task();
    input.dim = 1;
    input.idx = 1;
    input.ids = {
        { 0, 0, 0 },
        { 0, 0, 1 },
        { 0, 1, 0 },
        { 0, 1, 1 },
    };
    input.shape      = { 3, 3, 3 };
    input.shape_cube = { 5, 5, 5 };

    const auto msg = input.pack();
    auto slice = one::proc::make("slice");
    slice->init(msg.data(), msg.size());

    auto expected = std::vector< float >();
    for (int i = 0; i < int(input.ids.size()); ++i) {
        auto blob = GENERATE(
            take(1,
                chunk(3 * 3 * 3, random(-10000.0f, 10000.0f))
            )
        );
        slice->add(i,
            reinterpret_cast< const char* >(blob.data()),
            int(blob.size() * sizeof(float))
        );
        add_dim1_line(expected, blob);
    }

    auto output = unpack< one::slice_tiles >(slice->pack());
    std::vector< float > extracted;
    for (const auto& tile : output.tiles)
        extracted.insert(extracted.end(), tile.v.begin(), tile.v.end());
    CHECK_THAT(extracted, Equals(expected));
}

TEST_CASE("slice.add is not sensitive to order") {
    auto input = default_slice_task();
    input.ids = {
        { 0, 0, 0 },
        { 0, 0, 1 },
    };
    input.shape      = { 1, 1, 1 };
    input.shape_cube = { 2, 2, 2 };

    const auto msg = input.pack();
    auto slice = one::proc::make("slice");
    slice->init(msg.data(), msg.size());

    std::vector< float > expected[] = {
        { 1 },
        { 2 },
    };
    using record = std::tuple< int, int >;
    auto order = GENERATE(table< int, int >({
        { 0, 1 },
        { 1, 0 },
    }));
    const auto l = std::get< 0 >(order);
    const auto r = std::get< 1 >(order);

    slice->add(l, (char*)expected[l].data(), sizeof(float));
    slice->add(r, (char*)expected[r].data(), sizeof(float));

    auto unpacked = unpack< one::slice_tiles >(slice->pack());
    CHECK(unpacked.tiles.size() == 2);
    CHECK_THAT(unpacked.tiles.at(0).v, Equals(expected[0]));
    CHECK_THAT(unpacked.tiles.at(1).v, Equals(expected[1]));
}

one::curtain_task default_curtain_task() {
    one::curtain_task input;
    input.pid    = "some-pid";
    input.token  = "some-token";
    input.guid   = "some-guid";
    input.prefix = "src";
    input.ext    = "f32";

    input.storage_endpoint = "some-endpoint";
    input.shape      = { 64, 64, 64 };
    input.shape_cube = { 720, 860, 251 };

    return input;
}

TEST_CASE("curtain.fragments generates the right IDs from a task") {
    auto input = default_curtain_task();
    auto slice = one::proc::make("curtain");

    SECTION("None when ids are empty") {
        input.ids = {};
        const auto msg = input.pack();
        slice->init(msg.data(), msg.size());
        CHECK(slice->fragments() == "");
    }

    SECTION("Without delimiter when there is 1 id") {
        input.ids = {
            one::single {
                { 1, 1, 1 },  /* fragment-id */
                { { 0, 0 } }, /* local (x,y) in fragment */
            },
        };
        const auto msg = input.pack();
        slice->init(msg.data(), msg.size());
        const auto expected = "src/64-64-64/1-1-1.f32";

        CHECK(slice->fragments() == expected);
    }

    SECTION("Delimited when there are many ids") {
        input.ids = {
            one::single {
                { 0, 1, 2 },
                { { 0, 0 } },
            },
            one::single {
                { 2, 1, 1 },
                { { 0, 0 } },
            },
        };
        const auto msg = input.pack();
        slice->init(msg.data(), msg.size());
        const auto expected =
            "src/64-64-64/0-1-2.f32" ";"
            "src/64-64-64/2-1-1.f32"
        ;
        CHECK(slice->fragments() == expected);
    }

    SECTION("Independently of the # of traces per fragment") {
        input.ids = {
            one::single {
                { 0, 1, 2 },
                {},
            },
            one::single {
                { 2, 1, 1 },
                { { 0, 0 }, { 1, 1 } },
            },
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

TEST_CASE("Curtains extracted from chunks matches hand-extracted curtain") {
    /*
     * This test is built by figuring out by hand the traces to extract. It
     * uses magic constants for dimensions everywhere, and is a reasonably
     * straight-forward "known input" test.
     *
     * The tests does deserve some more love, but the "unit" here is pretty
     * opaque, so we can really only test the fairly nested objects coming out
     * by observing add(). The interface might need some tuning to make it
     * easier to test it without so much ceremony.
     */
    auto input = default_curtain_task();
    input.ids = {
        one::single { {0, 0, 0}, { {2, 1}, {2, 2} } },
        one::single { {0, 0, 1}, { {2, 1}, {2, 2} } },
        one::single { {0, 1, 0}, { {0, 0} } },
        one::single { {0, 1, 1}, { {0, 0} } },
    };
    input.shape      = { 3, 3, 3 };
    input.shape_cube = { 5, 5, 5 };

    const auto msg = input.pack();
    auto slice = one::proc::make("curtain");
    slice->init(msg.data(), msg.size());

    auto expected = std::vector< one::trace >();
    for (int i = 0; i < int(input.ids.size()); ++i) {
        const auto blob = GENERATE(
            take(1,
                chunk(3 * 3 * 3, random(-10000.0f, 10000.0f))
            )
        );
        slice->add(i,
            reinterpret_cast< const char* >(blob.data()),
            int(blob.size() * sizeof(float))
        );

        const auto& single = input.ids[i];
        for (auto& coord : single.coordinates) {
            one::trace t;
            t.coordinates = {
                single.id[0] * 3 + coord[0],
                single.id[1] * 3 + coord[1],
                single.id[2] * 3,
            };
            auto itr = blob.begin() + coord[0] * 3 * 3 + coord[1] * 3;
            t.v.assign(itr, itr + 3);
            expected.push_back(t);
        }
    }

    auto packed = slice->pack();
    auto output = unpack< one::curtain_traces >(slice->pack());

    REQUIRE(expected.size() == output.traces.size());
    for (int i = 0; i < int(expected.size()); ++i) {
        const auto& trace = output.traces.at(i);
        CHECK_THAT(expected[i].coordinates, Equals(trace.coordinates));
        CHECK_THAT(expected[i].v, Equals(trace.v));
    }
}

TEST_CASE("All process kinds can be constructed") {
    CHECK( one::proc::make("slice"));
    CHECK( one::proc::make("curtain"));
    CHECK(!one::proc::make("unknown"));
}
