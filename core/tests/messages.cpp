#include <algorithm>
#include <cstring>
#include <string>

#include <catch/catch.hpp>
#include <fmt/format.h>

#include <oneseismic/messages.hpp>

using namespace Catch::Matchers;

namespace one {

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

    if (!std::equal(lhs.vol.begin(), lhs.vol.end(), rhs.vol.begin()))
        return false;
    if (!std::equal(lhs.attr.begin(), lhs.attr.end(), rhs.attr.begin()))
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
        "url-query": "",
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

SCENARIO("Converting from UTM coordinates to cartesian grid") {
    const std::vector< int > inlines{1, 2, 3, 5, 6};
    const std::vector< int > crosslines{11, 12, 13, 14, 16, 17};
    float offsetx = 1;
    float offsety = 10;
    // We align the inline with the x-axis and crossline with the y-axis and add
    // a rotation 0.52 radians (approx pi/5). Before adding rotation we have:
    //
    //    x = inline * ilinc, y = crossline * xlinc
    //
    // Applying the rotation matrix:
    //
    //    cos(rot) -sin(rot)  *  inline * ilinc
    //    sin(rot) cos(rot)      crossline * xlinc
    //
    //    =  inline * ilinc * cos(rot) - crossline * xlinc * sin(rot)
    //       inline * ilinc * sin(rot) + crossline * xlinc * cos(rot)
    //
    //    =  inline * ilincx + crossline * xlincx
    //       inline * ilincy + crossline * xlincy
    //
    // Including the initial offset at (inline, crossline) = (0, 0) we get:
    //
    //    x = inline * ilincx + crossline * xlincx + offsetx,
    //    y = inline * ilincy + crossline * xlincy + offsety

    float ilincx = 1 * cos(0.52);
    float ilincy = 1 * sin(0.52);
    float xlincx = - 2 * sin(0.52);
    float xlincy = 2 * cos(0.52);

    // The above equations can be written in matrix notation:
    //
    //    ilincx xlincx offsetx     inline        x
    //    ilincy xlincy offsety  x  crossline  =  y
    //         0     0       1      1             1
    //
    // The utm_to_lineno transformation matrix is derived by computing the
    // inverse of the matrix to the left in the above expression and removing
    // the last row.
    const std::vector< std::vector< double > > utm_to_lineno {
        {0.86781918,  0.49688014, -5.83662056},
        {-0.24844007, 0.43390959, -4.09065583}
    };

    GIVEN("A point that falls on a missing line") {
        float x = 3.99 * ilincx + 15.01 * xlincx + offsetx;
        float y = 3.99 * ilincy + 15.01 * xlincy + offsety;

        WHEN("Converting to cartesian coordinate") {
            auto result = one::detail::utm_to_cartesian(
                inlines,
                crosslines,
                utm_to_lineno,
                x,
                y
            );

            THEN("The cartesian coordinate for the nearest line is found"){
                CHECK(result.first == 2);
                CHECK(result.second == 4);
            }
        }
    }
}

SCENARIO("Unpacking a curtain request") {
    std::string doc = R"({
        "pid": "some-pid",
        "token": "on-behalf-of-token",
        "url-query": "",
        "guid": "object-id",
        "storage_endpoint": "https://storage.com",
        "manifest": {
            "format-version": 1,
            "data": [
                {
                    "file-extension": "f32",
                    "shapes": [[2, 2, 2]],
                    "prefix": "prefix",
                    "resolution": "source"
                }
            ],
            "attributes": [],
            "line-numbers": [[10, 11], [1, 2], [0, 1]],
            "line-labels": ["dim-0"],
            "utm-to-lineno": [[1, 0, 10], [0, 1, 1]]
        },
        "function": "curtain",)";

    one::curtain_query query;

    GIVEN("Index coordinates") {
        doc += R"(
            "args": {
                "kind": "index",
                "coords": [[0, 1], [1, 0]]
            }
        })";

        WHEN("Unpacking the request") {
            query.unpack(doc.c_str(), doc.c_str() + doc.size());

            THEN("The coordinates are unpacked correctly") {
                CHECK(query.dim0s == std::vector{0, 1});
                CHECK(query.dim1s == std::vector{1, 0});
            }
        }
    }

    GIVEN("Lineno coordinates") {
        doc += R"(
            "args": {
                "kind": "lineno",
                "coords": [[10, 2], [11, 1]]
            }
        })";

        WHEN("Unpacking the request") {
            query.unpack(doc.c_str(), doc.c_str() + doc.size());

            THEN("The coordinates are unpacked correctly") {
                CHECK(query.dim0s == std::vector{0, 1});
                CHECK(query.dim1s == std::vector{1, 0});
            }
        }
    }

    GIVEN("UTM coordinates") {
        doc += R"(
            "args": {
                "kind": "utm",
                "coords": [[0.1, 1.1], [0.9, -0.1]]
            }
        })";

        WHEN("Unpacking the request") {
            query.unpack(doc.c_str(), doc.c_str() + doc.size());

            THEN("The coordinates are unpacked correctly") {
                CHECK(query.dim0s == std::vector{0, 1});
                CHECK(query.dim1s == std::vector{1, 0});
            }
        }
    }
}