#include <catch/catch.hpp>

#include <seismic-cloud/seismic-cloud.hpp>
#include "generators.hpp"

using namespace Catch::Matchers;

SCENARIO( "Converting between global and local coordinates" ) {

    GIVEN("A point in global grid is divisible "
          "by the subcube dimensions") {
        sc::cube_point< 3 > p {100, 200, 110};
        sc::cube_dimension< 3 > cube_size {2000, 2000, 1000};
        sc::frag_dimension< 3 > frag_size {20, 20, 10};

        const auto co = sc::cubecoords< 3 >(cube_size, frag_size);

        WHEN("Converting to local coordinates") {
            const auto local = co.to_local(p);

            THEN("The point should be in origo in "
                 "the local coordinate system") {
                CHECK(local == sc::frag_point< 3 > {0, 0, 0});
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
        sc::cube_point< 3 > p {55, 67, 88};
        sc::cube_dimension< 3 > cube {220, 200, 100};
        sc::frag_dimension< 3 > frag {22, 20, 10};

        const auto co = sc::cubecoords< 3 >(cube, frag);

        WHEN("Converting to local coordinates") {
            const auto local = co.to_local(p);

            THEN("The point is correctly converted to local coordiantes") {
                CHECK(local == sc::frag_point< 3 > {11, 7, 8});
            }

            THEN("The point can be converted back to global coordiantes") {
                const auto root = co.frag_id(p);
                const auto result = co.to_global(root, local);
                CHECK(result == p);
            }
        }
    }

    GIVEN("Points that should be mapped to the fragment (upper) corners") {
        const sc::cube_point< 3 > p1 {98, 59, 54};
        const sc::cube_point< 3 > p2 {65, 79, 109};

        const sc::frag_dimension< 3 > frag1 {33, 20, 11};
        const sc::frag_dimension< 3 > frag2 {22, 20, 10};

        const sc::cube_dimension< 3 > cube {220, 200, 1000};

        const auto co1 = sc::cubecoords< 3 >(cube, frag1);
        const auto co2 = sc::cubecoords< 3 >(cube, frag2);

        WHEN("Converting to local coordinates") {
            const auto local1 = co1.to_local(p1);
            const auto local2 = co2.to_local(p2);

            THEN("The point is mapped to the subcubes (upper) corner") {
                CHECK(local1 == sc::frag_point< 3 > {32, 19, 10});
                CHECK(local2 == sc::frag_point< 3 > {21, 19, 9});
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

TEST_CASE("Generate the fragments capturing an inline") {
    auto cube = sc::cubecoords< 3  >(
        { 9, 15, 23 },
        { 3,  9,  5 }
    );

    CHECK(cube.size(sc::dimension< 3 >{0}) == 3);
    CHECK(cube.size(sc::dimension< 3 >{1}) == 2);
    CHECK(cube.size(sc::dimension< 3 >{2}) == 5);

    const auto result = cube.slice(sc::dimension< 3 >{0}, 0);
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
    auto cube = sc::cubecoords< 3 > {
        { 9, 15, 23 },
        { 3,  9,  5 },
    };

    CHECK(cube.size(sc::dimension< 3 >{0}) == 3);
    CHECK(cube.size(sc::dimension< 3 >{1}) == 2);
    CHECK(cube.size(sc::dimension< 3 >{2}) == 5);

    const auto result = cube.slice(sc::dimension< 3 >{1}, 1);
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
    auto cube = sc::cubecoords< 3 > {
        { 9, 15, 23 },
        { 3,  9,  5 },
    };

    CHECK(cube.size(sc::dimension< 3 >{0}) == 3);
    CHECK(cube.size(sc::dimension< 3 >{1}) == 2);
    CHECK(cube.size(sc::dimension< 3 >{2}) == 5);

    const auto result = cube.slice(sc::dimension< 3 >{2}, 3);
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
    const auto cube = sc::cube_dimension< 3 >(9, 15, 23);
    const auto expected = 2495;
    const auto p = sc::cube_point< 3 >(7, 3, 11);
    auto result = cube.to_offset(p);
    CHECK(result == expected);
}
