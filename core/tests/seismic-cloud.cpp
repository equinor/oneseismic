#include <catch/catch.hpp>

#include <seismic-cloud/seismic-cloud.hpp>
#include "generators.hpp"


SCENARIO( "Converting between global and local coordinates" ) {

    GIVEN("A point in global grid is divisible "
          "by the subcube dimensions") {
        sc::cube_point p {100, 200, 110};
        sc::cube_dim cube_size {2000, 2000, 1000};
        sc::frag_dim frag_size {20, 20, 10};

        const auto co = sc::cubecoords(cube_size, frag_size);

        WHEN("Converting to local coordinates") {
            const auto local = co.to_local(p);

            THEN("The point should be in origo in "
                 "the local coordinate system") {
                CHECK(local == sc::frag_point {0, 0, 0});
            }

            THEN("The point can be converted back to global coordinates") {
                auto root = co.frag_id(p);
                auto result = co.to_global(root, local);
                CHECK(result == p);
            }
        }
    }

    GIVEN( "A point in global grid not divisible "
           "by the fragment dimensions" ) {
        sc::cube_point p {55, 67, 88};
        sc::cube_dim cube {220, 200, 100};
        sc::frag_dim frag {22, 20, 10};

        const auto co = sc::cubecoords(cube, frag);

        WHEN("Converting to local coordinates") {
            const auto local = co.to_local(p);

            THEN("The point is correctly converted to local coordiantes") {
                CHECK(local == sc::frag_point {11, 7, 8});
            }

            THEN("The point can be converted back to global coordiantes") {
                const auto root = co.frag_id(p);
                const auto result = co.to_global(root, local);
                CHECK(result == p);
            }
        }
    }

    GIVEN("Points that should be mapped to the fragment (upper) corners") {
        const sc::cube_point p1 {98, 59, 54};
        const sc::cube_point p2 {65, 79, 109};

        const sc::frag_dim frag1 {33, 20, 11};
        const sc::frag_dim frag2 {22, 20, 10};

        const sc::cube_dim cube {220, 200, 1000};

        const auto co1 = sc::cubecoords(cube, frag1);
        const auto co2 = sc::cubecoords(cube, frag2);

        WHEN("Converting to local coordinates") {
            const auto local1 = co1.to_local(p1);
            const auto local2 = co2.to_local(p2);

            THEN("The point is mapped to the subcubes (upper) corner") {
                CHECK(local1 == sc::frag_point {32, 19, 10});
                CHECK(local2 == sc::frag_point {21, 19, 9});
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
