#include <catch/catch.hpp>

#include <oneseismic/geometry.hpp>
#include "generators.hpp"

using namespace Catch::Matchers;

SCENARIO( "Converting between global and local coordinates" ) {

    GIVEN("A point in global grid is divisible "
          "by the subcube dimensions") {
        one::CP< 3 > p {100, 200, 110};
        one::CS< 3 > cube_size {2000, 2000, 1000};
        one::FS< 3 > frag_size {20, 20, 10};

        const auto co = one::gvt< 3 >(cube_size, frag_size);

        WHEN("Converting to local coordinates") {
            const auto local = co.to_local(p);

            THEN("The point should be in origo in "
                 "the local coordinate system") {
                CHECK(local == one::FP< 3 > {0, 0, 0});
            }

            THEN("The point can be converted back to global coordinates") {
                auto root = co.frag_id(p);
                auto result = co.to_global(root, local);
                CHECK(result == p);
            }
        }
    }

    GIVEN( "A point in global grid not divisible "
           "by the fragment dimension< 3 >s" ) {
        one::CP< 3 > p {55, 67, 88};
        one::CS< 3 > cube {220, 200, 100};
        one::FS< 3 > frag {22, 20, 10};

        const auto co = one::gvt< 3 >(cube, frag);

        WHEN("Converting to local coordinates") {
            const auto local = co.to_local(p);

            THEN("The point is correctly converted to local coordiantes") {
                CHECK(local == one::FP< 3 > {11, 7, 8});
            }

            THEN("The point can be converted back to global coordiantes") {
                const auto root = co.frag_id(p);
                const auto result = co.to_global(root, local);
                CHECK(result == p);
            }
        }
    }

    GIVEN("Points that should be mapped to the fragment (upper) corners") {
        const one::CP< 3 > p1 {98, 59, 54};
        const one::CP< 3 > p2 {65, 79, 109};

        const one::FS< 3 > frag1 {33, 20, 11};
        const one::FS< 3 > frag2 {22, 20, 10};

        const one::CS< 3 > cube {220, 200, 1000};

        const auto co1 = one::gvt< 3 >(cube, frag1);
        const auto co2 = one::gvt< 3 >(cube, frag2);

        WHEN("Converting to local coordinates") {
            const auto local1 = co1.to_local(p1);
            const auto local2 = co2.to_local(p2);

            THEN("The point is mapped to the subcubes (upper) corner") {
                CHECK(local1 == one::FP< 3 > {32, 19, 10});
                CHECK(local2 == one::FP< 3 > {21, 19, 9});
            }

            THEN("The point can be converted back to global coordinates") {
                const auto root1 = co1.frag_id(p1);
                const auto root2 = co2.frag_id(p2);

                const auto result1 = co1.to_global(root1, local1);
                const auto result2 = co2.to_global(root2, local2);

                CHECK(result1 == p1);
                CHECK(result2 == p2);
            }
        }
    }
}

TEST_CASE("Global slice indices are mapped to fragment-local indices") {
    const auto fs = one::FS< 3 > { 2, 3, 4 };
    using Dim = one::FS< 3 >::Dimension;
    CHECK(fs.index(Dim(0), 3) == 1);
    CHECK(fs.index(Dim(1), 3) == 0);
    CHECK(fs.index(Dim(2), 3) == 3);
}

TEST_CASE("Squeezing gvt") {
    const auto original = one::gvt< 3 >(
        one::CS< 3 >(6, 9, 18),
        one::FS< 3 >(2, 3, 5)
    );

    using Dimension = decltype(original)::Dimension;
    SECTION("in Dimension(0)") {
        const auto squeezed = original.squeeze(Dimension(0));
        const auto cs = squeezed.cube_shape();
        const auto fs = squeezed.fragment_shape();
        CHECK(squeezed.cube_shape().size() == 2);
        CHECK(squeezed.cube_shape()[0] == original.cube_shape()[1]);
        CHECK(squeezed.cube_shape()[1] == original.cube_shape()[2]);

        CHECK(squeezed.fragment_shape().size() == 2);
        CHECK(squeezed.fragment_shape()[0] == original.fragment_shape()[1]);
        CHECK(squeezed.fragment_shape()[1] == original.fragment_shape()[2]);
    }

    SECTION("in Dimension(1)") {
        const auto squeezed = original.squeeze(Dimension(1));
        const auto cs = squeezed.cube_shape();
        const auto fs = squeezed.fragment_shape();
        CHECK(squeezed.cube_shape().size() == 2);
        CHECK(squeezed.cube_shape()[0] == original.cube_shape()[0]);
        CHECK(squeezed.cube_shape()[1] == original.cube_shape()[2]);

        CHECK(squeezed.fragment_shape().size() == 2);
        CHECK(squeezed.fragment_shape()[0] == original.fragment_shape()[0]);
        CHECK(squeezed.fragment_shape()[1] == original.fragment_shape()[2]);
    }

    SECTION("in Dimension(2)") {
        const auto squeezed = original.squeeze(Dimension(2));
        const auto cs = squeezed.cube_shape();
        const auto fs = squeezed.fragment_shape();
        CHECK(squeezed.cube_shape().size() == 2);
        CHECK(squeezed.cube_shape()[0] == original.cube_shape()[0]);
        CHECK(squeezed.cube_shape()[1] == original.cube_shape()[1]);

        CHECK(squeezed.fragment_shape().size() == 2);
        CHECK(squeezed.fragment_shape()[0] == original.fragment_shape()[0]);
        CHECK(squeezed.fragment_shape()[1] == original.fragment_shape()[1]);
    }
}

TEST_CASE("gvt counts fragments and samples right") {
    auto cube = one::gvt< 3  >(
        { 9, 15, 23 },
        { 3,  9,  5 }
    );

    SECTION("in dimension 0") {
        const auto d = cube.mkdim(0);
        CHECK(cube.fragment_count(d)  == 3);
        CHECK(cube.nsamples(d)        == 9);
        CHECK(cube.nsamples_padded(d) == 9);
    }

    SECTION("in dimension 1") {
        const auto d = cube.mkdim(1);
        CHECK(cube.fragment_count(d)  == 2);
        CHECK(cube.nsamples(d)        == 15);
        CHECK(cube.nsamples_padded(d) == 18);
    }

    SECTION("in dimension 2") {
        const auto d = cube.mkdim(2);
        CHECK(cube.fragment_count(d)  == 5);
        CHECK(cube.nsamples(d)        == 23);
        CHECK(cube.nsamples_padded(d) == 25);
    }
}

TEST_CASE("Generate the fragments capturing an inline") {
    auto cube = one::gvt< 3  >(
        { 9, 15, 23 },
        { 3,  9,  5 }
    );

    CHECK(cube.fragment_count(one::dimension< 3 >{0}) == 3);
    CHECK(cube.fragment_count(one::dimension< 3 >{1}) == 2);
    CHECK(cube.fragment_count(one::dimension< 3 >{2}) == 5);

    const auto result = cube.slice(one::dimension< 3 >{0}, 0);
    const auto expected = decltype(result) {
        { 0, 0, 0 },
        { 0, 0, 1 },
        { 0, 0, 2 },
        { 0, 0, 3 },
        { 0, 0, 4 },
        { 0, 1, 0 },
        { 0, 1, 1 },
        { 0, 1, 2 },
        { 0, 1, 3 },
        { 0, 1, 4 },
    };

    CHECK_THAT(result, Equals(expected));
}

TEST_CASE("Generate the fragments capturing a crossline") {
    auto cube = one::gvt< 3 > {
        { 9, 15, 23 },
        { 3,  9,  5 },
    };

    CHECK(cube.fragment_count(one::dimension< 3 >{0}) == 3);
    CHECK(cube.fragment_count(one::dimension< 3 >{1}) == 2);
    CHECK(cube.fragment_count(one::dimension< 3 >{2}) == 5);

    const auto result = cube.slice(one::dimension< 3 >{1}, 11);
    const auto expected = decltype(result) {
        { 0, 1, 0 },
        { 0, 1, 1 },
        { 0, 1, 2 },
        { 0, 1, 3 },
        { 0, 1, 4 },

        { 1, 1, 0 },
        { 1, 1, 1 },
        { 1, 1, 2 },
        { 1, 1, 3 },
        { 1, 1, 4 },

        { 2, 1, 0 },
        { 2, 1, 1 },
        { 2, 1, 2 },
        { 2, 1, 3 },
        { 2, 1, 4 },
    };

    CHECK_THAT(result, Equals(expected));
}

TEST_CASE("Generate the fragments capturing a time slice") {
    auto cube = one::gvt< 3 > {
        { 9, 15, 23 },
        { 3,  9,  5 },
    };

    CHECK(cube.fragment_count(one::dimension< 3 >{0}) == 3);
    CHECK(cube.fragment_count(one::dimension< 3 >{1}) == 2);
    CHECK(cube.fragment_count(one::dimension< 3 >{2}) == 5);

    const auto result = cube.slice(one::dimension< 3 >{2}, 17);
    const auto expected = decltype(result) {
        { 0, 0, 3 },
        { 0, 1, 3 },

        { 1, 0, 3 },
        { 1, 1, 3 },

        { 2, 0, 3 },
        { 2, 1, 3 },
    };
    CHECK_THAT(result, Equals(expected));
}

TEST_CASE("Figure out an global offset [0, len(survey)) from a point") {
    const auto cube = one::CS< 3 >(9, 15, 23);
    const auto expected = 2495;
    const auto p = one::CP< 3 >(7, 3, 11);
    auto result = cube.to_offset(p);
    CHECK(result == expected);
}

TEST_CASE("fragment-id string generation") {
    const auto id = one::FID< 3 >(3, 5, 7);
    CHECK("3-5-7" == id.string());
}

namespace {

const auto exdims = one::FS< 3 >(3, 5, 7);
const auto exfragment = std::vector< unsigned char > {
    0x0, 0x0, 0x0, 0x0,
    0x0, 0x0, 0x1, 0x0,
    0x0, 0x0, 0x2, 0x0,
    0x0, 0x0, 0x3, 0x0,
    0x0, 0x0, 0x4, 0x0,
    0x0, 0x0, 0x5, 0x0,
    0x0, 0x0, 0x6, 0x0,

    0x0, 0x1, 0x0, 0x0,
    0x0, 0x1, 0x1, 0x0,
    0x0, 0x1, 0x2, 0x0,
    0x0, 0x1, 0x3, 0x0,
    0x0, 0x1, 0x4, 0x0,
    0x0, 0x1, 0x5, 0x0,
    0x0, 0x1, 0x6, 0x0,

    0x0, 0x2, 0x0, 0x0,
    0x0, 0x2, 0x1, 0x0,
    0x0, 0x2, 0x2, 0x0,
    0x0, 0x2, 0x3, 0x0,
    0x0, 0x2, 0x4, 0x0,
    0x0, 0x2, 0x5, 0x0,
    0x0, 0x2, 0x6, 0x0,

    0x0, 0x3, 0x0, 0x0,
    0x0, 0x3, 0x1, 0x0,
    0x0, 0x3, 0x2, 0x0,
    0x0, 0x3, 0x3, 0x0,
    0x0, 0x3, 0x4, 0x0,
    0x0, 0x3, 0x5, 0x0,
    0x0, 0x3, 0x6, 0x0,

    0x0, 0x4, 0x0, 0x0,
    0x0, 0x4, 0x1, 0x0,
    0x0, 0x4, 0x2, 0x0,
    0x0, 0x4, 0x3, 0x0,
    0x0, 0x4, 0x4, 0x0,
    0x0, 0x4, 0x5, 0x0,
    0x0, 0x4, 0x6, 0x0,

    0x1, 0x0, 0x0, 0x0,
    0x1, 0x0, 0x1, 0x0,
    0x1, 0x0, 0x2, 0x0,
    0x1, 0x0, 0x3, 0x0,
    0x1, 0x0, 0x4, 0x0,
    0x1, 0x0, 0x5, 0x0,
    0x1, 0x0, 0x6, 0x0,

    0x1, 0x1, 0x0, 0x0,
    0x1, 0x1, 0x1, 0x0,
    0x1, 0x1, 0x2, 0x0,
    0x1, 0x1, 0x3, 0x0,
    0x1, 0x1, 0x4, 0x0,
    0x1, 0x1, 0x5, 0x0,
    0x1, 0x1, 0x6, 0x0,

    0x1, 0x2, 0x0, 0x0,
    0x1, 0x2, 0x1, 0x0,
    0x1, 0x2, 0x2, 0x0,
    0x1, 0x2, 0x3, 0x0,
    0x1, 0x2, 0x4, 0x0,
    0x1, 0x2, 0x5, 0x0,
    0x1, 0x2, 0x6, 0x0,

    0x1, 0x3, 0x0, 0x0,
    0x1, 0x3, 0x1, 0x0,
    0x1, 0x3, 0x2, 0x0,
    0x1, 0x3, 0x3, 0x0,
    0x1, 0x3, 0x4, 0x0,
    0x1, 0x3, 0x5, 0x0,
    0x1, 0x3, 0x6, 0x0,

    0x1, 0x4, 0x0, 0x0,
    0x1, 0x4, 0x1, 0x0,
    0x1, 0x4, 0x2, 0x0,
    0x1, 0x4, 0x3, 0x0,
    0x1, 0x4, 0x4, 0x0,
    0x1, 0x4, 0x5, 0x0,
    0x1, 0x4, 0x6, 0x0,

    0x2, 0x0, 0x0, 0x0,
    0x2, 0x0, 0x1, 0x0,
    0x2, 0x0, 0x2, 0x0,
    0x2, 0x0, 0x3, 0x0,
    0x2, 0x0, 0x4, 0x0,
    0x2, 0x0, 0x5, 0x0,
    0x2, 0x0, 0x6, 0x0,

    0x2, 0x1, 0x0, 0x0,
    0x2, 0x1, 0x1, 0x0,
    0x2, 0x1, 0x2, 0x0,
    0x2, 0x1, 0x3, 0x0,
    0x2, 0x1, 0x4, 0x0,
    0x2, 0x1, 0x5, 0x0,
    0x2, 0x1, 0x6, 0x0,

    0x2, 0x2, 0x0, 0x0,
    0x2, 0x2, 0x1, 0x0,
    0x2, 0x2, 0x2, 0x0,
    0x2, 0x2, 0x3, 0x0,
    0x2, 0x2, 0x4, 0x0,
    0x2, 0x2, 0x5, 0x0,
    0x2, 0x2, 0x6, 0x0,

    0x2, 0x3, 0x0, 0x0,
    0x2, 0x3, 0x1, 0x0,
    0x2, 0x3, 0x2, 0x0,
    0x2, 0x3, 0x3, 0x0,
    0x2, 0x3, 0x4, 0x0,
    0x2, 0x3, 0x5, 0x0,
    0x2, 0x3, 0x6, 0x0,

    0x2, 0x4, 0x0, 0x0,
    0x2, 0x4, 0x1, 0x0,
    0x2, 0x4, 0x2, 0x0,
    0x2, 0x4, 0x3, 0x0,
    0x2, 0x4, 0x4, 0x0,
    0x2, 0x4, 0x5, 0x0,
    0x2, 0x4, 0x6, 0x0,
};

}

std::vector< unsigned char > slice(one::slice_layout stride, std::size_t pin) {
    auto outcome = std::vector< unsigned char >();
    const auto start = pin * stride.initial_skip;
    const auto superstride = stride.superstride * sizeof(float);
    const auto chunk_size  = stride.chunk_size  * sizeof(float);
    auto pos = start * sizeof(float);
    for (auto i = 0; i < stride.iterations; ++i) {
        outcome.insert(
            outcome.end(),
            exfragment.begin() + pos,
            exfragment.begin() + pos + chunk_size
        );
        pos += superstride;
    }

    return outcome;
}

TEST_CASE("Extracting a dimension-0 slice from a fragment") {
    const auto expected = [=] {
        auto tmp = std::vector< unsigned char >();
        for (unsigned char i = 0; i < exdims[1]; ++i) {
            for (unsigned char k = 0; k < exdims[2]; ++k) {
                unsigned char t[] = { 0x1, 0x0, 0x0, 0x0, };
                t[1] = i;
                t[2] = k;
                tmp.insert(tmp.end(), std::begin(t), std::end(t));
            }
        }
        return tmp;
    }();

    const auto pin = 1;
    const auto stride = exdims.slice_stride(one::dimension< 3 >(0));
    const auto outcome = slice(stride, pin);
    CHECK_THAT(outcome, Equals(expected));
}

TEST_CASE("Extracting a dimension-1 slice from a fragment") {
    const auto expected = [=] {
        auto tmp = std::vector< unsigned char >();
        for (unsigned char i = 0; i < exdims[0]; ++i) {
            for (unsigned char k = 0; k < exdims[2]; ++k) {
                unsigned char t[] = { 0x0, 0x1, 0x0, 0x0, };
                t[0] = i;
                t[2] = k;
                tmp.insert(tmp.end(), std::begin(t), std::end(t));
            }
        }
        return tmp;
    }();

    const auto pin = 1;
    const auto stride = exdims.slice_stride(one::dimension< 3 >(1));
    const auto outcome = slice(stride, pin);
    CHECK_THAT(outcome, Equals(expected));
}

TEST_CASE("Extracting a dimension-2 slice from a fragment") {
    const auto expected = [=] {
        auto tmp = std::vector< unsigned char >();
        for (unsigned char i = 0; i < exdims[0]; ++i) {
            for (unsigned char k = 0; k < exdims[1]; ++k) {
                unsigned char t[] = { 0x0, 0x0, 0x1, 0x0, };
                t[0] = i;
                t[1] = k;
                tmp.insert(tmp.end(), std::begin(t), std::end(t));
            }
        }
        return tmp;
    }();

    const auto pin = 1;
    const auto stride = exdims.slice_stride(one::dimension< 3 >(2));
    const auto outcome = slice(stride, pin);
    CHECK_THAT(outcome, Equals(expected));
}

TEST_CASE("Put a fragment slice into a cube slice (dimension 0)") {
    const auto expected = std::vector< unsigned char > {
        0x1, 0x0, 0x0, 0x0,
        0x1, 0x0, 0x1, 0x0,
        0x1, 0x0, 0x2, 0x0,
        0x1, 0x0, 0x3, 0x0,
        0x1, 0x0, 0x4, 0x0,
        0x1, 0x0, 0x5, 0x0,
        0x1, 0x0, 0x6, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,

        0x1, 0x1, 0x0, 0x0,
        0x1, 0x1, 0x1, 0x0,
        0x1, 0x1, 0x2, 0x0,
        0x1, 0x1, 0x3, 0x0,
        0x1, 0x1, 0x4, 0x0,
        0x1, 0x1, 0x5, 0x0,
        0x1, 0x1, 0x6, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,

        0x1, 0x2, 0x0, 0x0,
        0x1, 0x2, 0x1, 0x0,
        0x1, 0x2, 0x2, 0x0,
        0x1, 0x2, 0x3, 0x0,
        0x1, 0x2, 0x4, 0x0,
        0x1, 0x2, 0x5, 0x0,
        0x1, 0x2, 0x6, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,

        0x1, 0x3, 0x0, 0x0,
        0x1, 0x3, 0x1, 0x0,
        0x1, 0x3, 0x2, 0x0,
        0x1, 0x3, 0x3, 0x0,
        0x1, 0x3, 0x4, 0x0,
        0x1, 0x3, 0x5, 0x0,
        0x1, 0x3, 0x6, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,

        0x1, 0x4, 0x0, 0x0,
        0x1, 0x4, 0x1, 0x0,
        0x1, 0x4, 0x2, 0x0,
        0x1, 0x4, 0x3, 0x0,
        0x1, 0x4, 0x4, 0x0,
        0x1, 0x4, 0x5, 0x0,
        0x1, 0x4, 0x6, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
    };

    const auto dim0 = one::dimension< 3 >(0);
    const auto slice_frag_dim = one::FS< 3 > { 1, 5,  7 };
    const auto slice_dim      = one::CS< 3 > { 1, 5, 14 };
    auto gvt = one::gvt< 2 >(slice_dim.squeeze(dim0),
                             slice_frag_dim.squeeze(dim0));
    REQUIRE(expected.size() == gvt.global_size() * sizeof(float));

    /* extract a slice from a fragment */
    const auto pin = 1;
    const auto source_stride = exdims.slice_stride(dim0);
    const auto source = slice(source_stride, pin);

    auto out = expected;
    out.assign(expected.size(), 0);

    /* Put the slice tile at the right place in the output array */
    const auto id = one::FID< 3 > { 0, 0, 0 };
    auto layout = gvt.injection_stride(id.squeeze(dim0));
    auto src = source.begin();
    auto dst = out.begin() + layout.initial_skip * sizeof(float);
    for (auto i = 0; i < layout.iterations; ++i) {
        std::copy_n(src, layout.chunk_size * sizeof(float), dst);
        src += layout.substride * sizeof(float);
        dst += layout.superstride * sizeof(float);
    }

    CHECK_THAT(out, Equals(expected));
}

TEST_CASE("Put a fragment slice into a cube slice (dimension 1)") {
    const auto expected = std::vector< unsigned char > {
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x1, 0x0, 0x0,
        0x0, 0x1, 0x1, 0x0,
        0x0, 0x1, 0x2, 0x0,
        0x0, 0x1, 0x3, 0x0,
        0x0, 0x1, 0x4, 0x0,
        0x0, 0x1, 0x5, 0x0,
        0x0, 0x1, 0x6, 0x0,

        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x1, 0x1, 0x0, 0x0,
        0x1, 0x1, 0x1, 0x0,
        0x1, 0x1, 0x2, 0x0,
        0x1, 0x1, 0x3, 0x0,
        0x1, 0x1, 0x4, 0x0,
        0x1, 0x1, 0x5, 0x0,
        0x1, 0x1, 0x6, 0x0,

        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x2, 0x1, 0x0, 0x0,
        0x2, 0x1, 0x1, 0x0,
        0x2, 0x1, 0x2, 0x0,
        0x2, 0x1, 0x3, 0x0,
        0x2, 0x1, 0x4, 0x0,
        0x2, 0x1, 0x5, 0x0,
        0x2, 0x1, 0x6, 0x0,
    };

    const auto dim1 = one::dimension< 3 >(1);
    const auto slice_frag_dim = one::FS< 3 > { 3, 1,  7 };
    const auto slice_dim      = one::CS< 3 > { 3, 1, 14 };
    auto gvt = one::gvt< 2 >(slice_dim.squeeze(dim1),
                             slice_frag_dim.squeeze(dim1));
    REQUIRE(expected.size() == gvt.global_size() * sizeof(float));

    /* extract a slice from a fragment */
    const auto pin = 1;
    const auto source_stride = exdims.slice_stride(dim1);
    const auto source = slice(source_stride, pin);

    auto out = expected;
    out.assign(expected.size(), 0);

    /* Put the slice tile at the right place in the output array */
    const auto id = one::FID< 3 > { 0, 0, 1 };
    auto layout = gvt.injection_stride(id.squeeze(dim1));
    auto src = source.begin();
    auto dst = out.begin() + layout.initial_skip * sizeof(float);
    for (auto i = 0; i < layout.iterations; ++i) {
        std::copy_n(src, layout.chunk_size * sizeof(float), dst);
        src += layout.substride * sizeof(float);
        dst += layout.superstride * sizeof(float);
    }

    CHECK_THAT(out, Equals(expected));
}

TEST_CASE("Put a fragment slice into a cube slice (dimension 1, lateral)") {
    const auto expected = std::vector< unsigned char > {
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,

        0x0, 0x1, 0x0, 0x0,
        0x0, 0x1, 0x1, 0x0,
        0x0, 0x1, 0x2, 0x0,
        0x0, 0x1, 0x3, 0x0,
        0x0, 0x1, 0x4, 0x0,
        0x0, 0x1, 0x5, 0x0,
        0x0, 0x1, 0x6, 0x0,
        0x1, 0x1, 0x0, 0x0,
        0x1, 0x1, 0x1, 0x0,
        0x1, 0x1, 0x2, 0x0,
        0x1, 0x1, 0x3, 0x0,
        0x1, 0x1, 0x4, 0x0,
        0x1, 0x1, 0x5, 0x0,
        0x1, 0x1, 0x6, 0x0,
        0x2, 0x1, 0x0, 0x0,
        0x2, 0x1, 0x1, 0x0,
        0x2, 0x1, 0x2, 0x0,
        0x2, 0x1, 0x3, 0x0,
        0x2, 0x1, 0x4, 0x0,
        0x2, 0x1, 0x5, 0x0,
        0x2, 0x1, 0x6, 0x0,
    };

    const auto dim1 = one::dimension< 3 >(1);
    const auto slice_frag_dim = one::FS< 3 > { 3, 1, 7 };
    const auto slice_dim      = one::CS< 3 > { 6, 1, 7 };
    auto gvt = one::gvt< 2 >(slice_dim.squeeze(dim1),
                             slice_frag_dim.squeeze(dim1));
    REQUIRE(expected.size() == gvt.global_size() * sizeof(float));

    /* extract a slice from a fragment */
    const auto pin = 1;
    const auto source_stride = exdims.slice_stride(dim1);
    const auto source = slice(source_stride, pin);

    auto out = expected;
    out.assign(expected.size(), 0);

    /* Put the slice tile at the right place in the output array */
    const auto id = one::FID< 3 > { 1, 0, 0 };
    auto layout = gvt.injection_stride(id.squeeze(dim1));
    auto src = source.begin();
    auto dst = out.begin() + layout.initial_skip * sizeof(float);
    for (auto i = 0; i < layout.iterations; ++i) {
        std::copy_n(src, layout.chunk_size * sizeof(float), dst);
        src += layout.substride * sizeof(float);
        dst += layout.superstride * sizeof(float);
    }

    CHECK_THAT(out, Equals(expected));
}

TEST_CASE("Put a fragment slice into a cube slice (dimension 2)") {
    const auto expected = std::vector< unsigned char > {
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,
        0x0, 0x0, 0x0, 0x0,

        0x0, 0x0, 0x1, 0x0,
        0x0, 0x1, 0x1, 0x0,
        0x0, 0x2, 0x1, 0x0,
        0x0, 0x3, 0x1, 0x0,
        0x0, 0x4, 0x1, 0x0,
        0x1, 0x0, 0x1, 0x0,
        0x1, 0x1, 0x1, 0x0,
        0x1, 0x2, 0x1, 0x0,
        0x1, 0x3, 0x1, 0x0,
        0x1, 0x4, 0x1, 0x0,
        0x2, 0x0, 0x1, 0x0,
        0x2, 0x1, 0x1, 0x0,
        0x2, 0x2, 0x1, 0x0,
        0x2, 0x3, 0x1, 0x0,
        0x2, 0x4, 0x1, 0x0,
    };

    const auto dim2 = one::dimension< 3 >(2);
    const auto slice_frag_dim = one::FS< 3 > { 3, 5, 1 };
    const auto slice_dim      = one::CS< 3 > { 6, 5, 1 };
    auto gvt = one::gvt< 2 >(slice_dim.squeeze(dim2),
                             slice_frag_dim.squeeze(dim2));
    REQUIRE(expected.size() == gvt.global_size() * sizeof(float));

    /* extract a slice from a fragment */
    const auto pin = 1;
    const auto source_stride = exdims.slice_stride(dim2);
    const auto source = slice(source_stride, pin);

    auto out = expected;
    out.assign(expected.size(), 0);

    /* Put the slice tile at the right place in the output array */
    const auto id = one::FID< 3 > { 1, 0, 0 };
    auto layout = gvt.injection_stride(id.squeeze(dim2));
    auto src = source.begin();
    auto dst = out.begin() + layout.initial_skip * sizeof(float);
    for (auto i = 0; i < layout.iterations; ++i) {
        std::copy_n(src, layout.chunk_size * sizeof(float), dst);
        src += layout.substride * sizeof(float);
        dst += layout.superstride * sizeof(float);
    }

    CHECK_THAT(out, Equals(expected));
}
