#include <catch/catch.hpp>

#include <seismic-cloud/seismic-cloud.hpp>
#include "generators.hpp"


SCENARIO( "Converting between global and local coordinates" ) {

    GIVEN( "A point in global coordiantes divisible "
           "by the subcube dimensions" ) {
        sc::point p {100, 200, 110};
        sc::dimension d {20, 20, 10};

        WHEN( "Converting to local coordinates" ) {
            sc::point local = sc::global_to_local( p, d );

            THEN( "The point should be in origo in "
                  "the local coordinate system" ) {
                CHECK( local == sc::point {0, 0, 0} );
            }

            THEN( "The point can be converted back to global coordiantes" ) {
                sc::point root = sc::global_to_root( p, d );
                sc::point result = sc::local_to_global( local, root );
                CHECK( result == p );
            }
        }
    }

    GIVEN( "A point in global coordiantes not divisible "
           "by the subcube dimensions" ) {
        sc::point p {55, 67, 88};
        sc::dimension d {22, 20, 10};

        WHEN( "Converting to local coordinates" ) {
            const auto local = sc::global_to_local( p, d );

            THEN( "The point is correctly converted to local coordiantes" ) {
                CHECK( local == sc::point {11, 7, 8} );
            }

            THEN( "The point can be converted back to global coordiantes" ) {
                const auto root = sc::global_to_root( p, d );
                const auto result = sc::local_to_global( local, root );
                CHECK( result == p );
            }
        }
    }

    GIVEN( "Points that should be mapped to the subcube (upper) corners" ) {
        const sc::point p1 {98, 59, 54};
        const sc::point p2 {65, 79, 109};

        const sc::dimension d1 {33, 20, 11};
        const sc::dimension d2 {22, 20, 10};

        WHEN( "Converting to local coordinates" ) {
            const auto local1 = sc::global_to_local( p1, d1 );
            const auto local2 = sc::global_to_local( p2, d2 );

            THEN( "The point is mapped to the subcubes (upper) corner" ) {
                CHECK( local1 == sc::point {32, 19, 10} );
                CHECK( local2 == sc::point {21, 19, 9} );
            }

            THEN( "The point can be converted back to global coordiantes" ) {
                const auto root1 = sc::global_to_root( p1, d1 );
                const auto root2 = sc::global_to_root( p2, d2 );

                const auto result1 = sc::local_to_global( local1, root1 );
                const auto result2 = sc::local_to_global( local2, root2 );

                CHECK( result1 == p1 );
                CHECK( result2 == p2 );
            }
        }
    }

    GIVEN( "Some randomly generated points and subcube dimensions" ) {
        const auto d = GENERATE( take( 10, random_dimensions() ) );
        const auto p = GENERATE( take( 100, random_points() ) );

        WHEN( "Converting to local coordinates" ) {
            const auto local = sc::global_to_local( p, d );

            THEN( "The converted point is within the subcube boarders" ) {
                CHECK( 0 <= local.x );
                CHECK( 0 <= local.y );
                CHECK( 0 <= local.z );

                CHECK( local.x < d.x );
                CHECK( local.y < d.y );
                CHECK( local.z < d.z );
            }

            const auto root = sc::global_to_root( p, d );

            THEN( "The subcube corner components are <= the "
                  "corresponding point components" ) {
                CHECK( root.x <= p.x );
                CHECK( root.y <= p.y );
                CHECK( root.z <= p.z );
            }

            THEN( "The point can be converted back to global coordiantes" ) {
                const auto result = sc::local_to_global( local, root );
                CHECK( result == p );
            }
        }
    }
}

SCENARIO( "Converting between points and offsets" ) {

    GIVEN( "Points inside the cube" ) {
        sc::point p1{   0,  6, 21 };
        sc::point p2{ 100,  7, 32 };

        sc::dimension d{101, 60, 63};

        WHEN( "Converting to offsets" ) {
            const auto off1 = sc::point_to_offset( p1, d );
            const auto off2 = sc::point_to_offset( p2, d );

            THEN( "The points are correctly converted" ) {
                CHECK( off1 ==    399 );
                CHECK( off2 == 378473 );
            }

            THEN( "The offset can be converted back" ) {
                CHECK( p1 == sc::offset_to_point( off1, d ) );
                CHECK( p2 == sc::offset_to_point( off2, d ) );
            }
        }
    }

    GIVEN( "Points on the cube boarders" ) {
        sc::point p1{   0,  6, 21 };
        sc::point p2{ 100,  7, 32 };
        sc::point p3{   6,  0, 60 };
        sc::point p4{  99, 59, 52 };
        sc::point p5{  55, 52, 0  };
        sc::point p6{  21, 59, 62 };

        sc::dimension d{101, 60, 63};

        WHEN( "Converting to offsets" ) {
            const auto off1 = sc::point_to_offset( p1, d );
            const auto off2 = sc::point_to_offset( p2, d );
            const auto off3 = sc::point_to_offset( p3, d );
            const auto off4 = sc::point_to_offset( p4, d );
            const auto off5 = sc::point_to_offset( p5, d );
            const auto off6 = sc::point_to_offset( p6, d );

            THEN( "The points are correctly converted" ) {
                CHECK( off1 ==    399 );
                CHECK( off2 == 378473 );
                CHECK( off3 ==  22740 );
                CHECK( off4 == 377989 );
                CHECK( off5 == 211176 );
                CHECK( off6 ==  83159 );
            }

            THEN( "The offset can be converted back" ) {
                CHECK( p1 == sc::offset_to_point( off1, d ) );
                CHECK( p2 == sc::offset_to_point( off2, d ) );
                CHECK( p3 == sc::offset_to_point( off3, d ) );
                CHECK( p4 == sc::offset_to_point( off4, d ) );
                CHECK( p5 == sc::offset_to_point( off5, d ) );
                CHECK( p6 == sc::offset_to_point( off6, d ) );
            }
        }
    }

    GIVEN( "Points on the cube corners" ) {
        const sc::point p1{   0,  0,  0 };
        const sc::point p2{   0,  0, 62 };
        const sc::point p3{   0, 59,  0 };
        const sc::point p4{   0, 59, 62 };
        const sc::point p5{ 100,  0,  0 };
        const sc::point p6{ 100,  0, 62 };
        const sc::point p7{ 100, 59,  0 };
        const sc::point p8{ 100, 59, 62 };

        const sc::dimension d{101, 60, 63};

        WHEN( "Converting to offsets" ) {
            const auto off1 = sc::point_to_offset( p1, d );
            const auto off2 = sc::point_to_offset( p2, d );
            const auto off3 = sc::point_to_offset( p3, d );
            const auto off4 = sc::point_to_offset( p4, d );
            const auto off5 = sc::point_to_offset( p5, d );
            const auto off6 = sc::point_to_offset( p6, d );
            const auto off7 = sc::point_to_offset( p7, d );
            const auto off8 = sc::point_to_offset( p8, d );

            THEN( "The points are correctly converted" ) {
                CHECK( off1 ==      0 );
                CHECK( off2 ==     62 );
                CHECK( off3 ==   3717 );
                CHECK( off4 ==   3779 );
                CHECK( off5 == 378000 );
                CHECK( off6 == 378062 );
                CHECK( off7 == 381717 );
                CHECK( off8 == 381779 );
            }

            THEN( "The offset can be converted back" ) {
                CHECK( p1 == sc::offset_to_point( off1, d ) );
                CHECK( p2 == sc::offset_to_point( off2, d ) );
                CHECK( p3 == sc::offset_to_point( off3, d ) );
                CHECK( p4 == sc::offset_to_point( off4, d ) );
                CHECK( p5 == sc::offset_to_point( off5, d ) );
                CHECK( p6 == sc::offset_to_point( off6, d ) );
            }
        }
    }

    GIVEN( "Some randomly generated cube dimensions and "
           "points within these cubes" ) {

        using Integral = decltype( sc::point_to_offset( {0,0,0}, {0,0,0} ) );
        const auto m = std::cbrt( std::numeric_limits< Integral >::max() );

        const auto d = GENERATE_COPY( take( 10, random_dimensions(m, m, m) ) );
        const auto p =
            GENERATE_COPY( take( 100, random_points( d.x-1, d.y-1, d.z-1 ) ) );

        WHEN( "Converting to offsets" ) {
            const auto off = sc::point_to_offset( p, d );

            THEN( "The offset is less than the cube size (x * y * z)" ) {
                const auto cube_size = d.x * d.y * d.z;
                CHECK( off < cube_size );
            }

            THEN( "The offset can be converted back" ) {
                CHECK( p == sc::offset_to_point( off, d ) );
            }
        }
    }
}

SCENARIO( "Converting from local to global offset" ) {

    GIVEN( "Some local offsets" ) {
        const std::size_t local_offset1 = 0;
        const std::size_t local_offset2 = 400;
        const std::size_t local_offset3 = 28337;
        const std::size_t local_offset4 = 4002;

        const sc::dimension fragment_size {22, 30, 43};
        const sc::dimension cube_size {603, 300, 533};
        const sc::point root {109, 300, 473};

        WHEN( "Converting to global offset" ) {
            const auto global_offset1 = sc::local_to_global( local_offset1,
                                                             fragment_size,
                                                             cube_size,
                                                             root );
            const auto global_offset2 = sc::local_to_global( local_offset2,
                                                             fragment_size,
                                                             cube_size,
                                                             root );
            const auto global_offset3 = sc::local_to_global( local_offset3,
                                                             fragment_size,
                                                             cube_size,
                                                             root );
            const auto global_offset4 = sc::local_to_global( local_offset4,
                                                             fragment_size,
                                                             cube_size,
                                                             root );

            THEN( "The offsets are converted correctly" ) {
                const std::size_t root_offset = 109*300*533 + 300*533 + 473;
                CHECK( global_offset1 == root_offset );
                CHECK( global_offset2 == root_offset + 9*533 + 13 );
                CHECK( global_offset3 == root_offset + 21*300*533 + 29*533 );
                CHECK( global_offset4 == root_offset + 3*300*533 + 3*533 + 3 );
            }
        }
    }
}
