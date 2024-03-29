#include <algorithm>
#include <cstring>
#include <string>
#include <regex>
#include <variant>

#include <catch/catch.hpp>
#include <nlohmann/json.hpp>
#include <fmt/format.h>
#include <fmt/printf.h>

#include <oneseismic/messages.hpp>

using namespace Catch::Matchers;

namespace one {

bool operator == (const one::basic_task& lhs, const one::basic_task& rhs) {
    return lhs.pid              == rhs.pid
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

namespace {
using jsonvalue = std::variant<
    int,
    std::string,
    std::vector< int >,
    std::vector< std::vector<int> >
>;

struct badjson {
    std::string keypath;
    jsonvalue value;
    std::string error;
};

std::string update_json(const std::string& qs, const std::string& keypath,
                        const jsonvalue& value) {
    auto doc = nlohmann::json::parse(qs.begin(), qs.end());
    const auto update = [&doc, &keypath](auto&& v) {
        const auto keypointer = nlohmann::json::json_pointer(keypath);
        doc.at(keypointer) = v;
    };

    std::visit(update, value);
    return nlohmann::json(doc).dump();
}
}

/***
 * Template for main fields and manifest
 */
const std::string query_base_template = R"(
    "pid": "some-pid",
    "url-query": "original query",
    "guid": "object-id",
    "storage_endpoint": "https://storage.com",
    "manifest": {
        "data": [
            {
                "file-extension": "f32",
                "shapes": [[2, 3, 15]],
                "prefix": "prefix"
            }
        ],
        "attributes": [
            {
                "type": "cdpx",
                "layout": "tiled",
                "file-extension": "f32",
                "labels": ["CDP X"],
                "shapes": [[512, 512, 1]],
                "prefix": "attributes/cdpx"
            }
        ],
        "line-numbers": %s,
        "line-labels": ["dim-0", "dim-1", "dim-2"]
        %s
    }
)";

/***
 * All possible required basic query fields
 */
const std::string query_required =
    fmt::sprintf(query_base_template,
                 "[[1, 2, 3, 4, 5], [6, 7, 8, 9, 10, 69], [12, 34, 560]]", "");

/***
 * All possible optional base query fields
 */
const std::string query_optional =  R"(
    "opts": {
        "attributes": ["attribute1", "attribute2"]
    }
)";

/***
 * Unexpected base query fields that are disregarded
 */
const std::string query_unexpected =  R"(
    "unexpected": "value"
)";

/***
 * All required query slice fields
 */
const std::string query_slice_specific = R"(
    "function": "slice",
    "args": {
        "kind": "lineno",
        "dim": 1,
        "val": 9
    }
)";

/***
 * All required query curtain fields
 */
const std::string query_curtain_specific = R"(
    "function": "curtain",
    "args": {
        "kind": "lineno",
        "coords": [[3, 9], [1, 7], [5, 9], [1, 9], [2, 7]]
    }
)";

const std::vector<std::string> query_specific = {
    query_slice_specific,
    query_curtain_specific
};

TEST_CASE("well-formed query is unpacked correctly") {
    constexpr auto templ = "{{ {}, {}, {}, {} }}";
    const auto qs = fmt::format(templ, query_required, query_slice_specific,
                                query_optional, query_unexpected);

    one::slice_query query;
    query.unpack(qs.c_str(), qs.c_str() + qs.length());

    one::volumedesc vol;
    vol.prefix = "prefix";
    vol.ext = "f32";
    vol.shapes = { { 2, 3, 15 } };

    one::attributedesc attr;
    attr.prefix = "attributes/cdpx";
    attr.ext = "f32";
    attr.type = "cdpx";
    attr.layout = "tiled";
    attr.labels = { "CDP X" };
    attr.shapes = {{512, 512, 1}};

    one::manifestdoc manifest;
    manifest.vol.push_back(vol);
    manifest.attr.push_back(attr);
    manifest.line_numbers = {
        {1, 2, 3, 4, 5}, {6, 7, 8, 9, 10, 69}, {12, 34, 560}};
    manifest.line_labels = {"dim-0", "dim-1", "dim-2"};

    const std::vector<std::string> attributes = {"attribute1", "attribute2"};

    CHECK(query.pid   == "some-pid");
    CHECK(query.url_query == "original query");
    CHECK(query.guid  == "object-id");
    CHECK(query.manifest == manifest);
    CHECK(query.storage_endpoint == "https://storage.com");
    CHECK(query.attributes == attributes);

    CHECK(query.function == "slice");
    CHECK(query.dim == 1);
    CHECK(query.idx == 3);
}

TEMPLATE_TEST_CASE_SIG("unpacking a query with missing field fails", "",
                       ((typename T, int i), T, i),
                       (one::slice_query, 0),
                       (one::curtain_query, 1)) {
    const auto qs =
        fmt::format("{{ {}, {} }}", query_required, query_specific[i]);

    const std::regex r("\"(.+)\"\\s*:"); // matches any key
    auto it = std::sregex_iterator(qs.begin(), qs.end(), r);

    for (it; it != std::sregex_iterator(); ++it) {
        auto scopy = qs;

        const auto key = it->str(1);
        const auto pos = it->position(1);
        const auto doc = scopy.replace(pos, key.length(), "dummy").c_str();

        T query;
        INFO(fmt::format("when key '{}' is missing", key));
        CHECK_THROWS_WITH(query.unpack(doc, doc + std::strlen(doc)),
                          Contains(fmt::format("key '{}' not found", key)));
    }
}


std::initializer_list<badjson> badslice = {
    {
        "/function",
        "dummy",
        "expected query 'slice', got dummy"
    },
    {
        "/args/dim",
        8,
        "args.dim (= 8) not in [0, 3)"
    },
    {
        "/args/val",
        11,
        "line (= 11) not found in index"
    },
    {
        "/args/kind",
        "dummy",
        "expected kind 'index' or 'lineno', got dummy"
    },
    {
        "/args/kind",
        "index",
        "index (= 9) not in [0, 6)"
    },
};

std::initializer_list<badjson> badcurtain = {
    {
        "/function",
        "dummy",
        "expected query 'curtain', got dummy"
    },
    {
        "/args/kind",
        "dummy",
        "expected kind 'index' or 'lineno' or 'utm', got dummy"
    },
    {
        "/args/kind",
        "index",
        "dimension 0: coordinate (= 5) of type index is out of cube boundaries [0, 5)"
    },
    {
        "/args/coords",
        std::vector<std::vector<int>>{{1, 6}, {3, 200}},
        "dimension 1: coordinate (= 200) of type lineno is not found"
    },
    {
        "/args/coords",
        std::vector<std::vector<int>>{{1}, {2}, {3}},
        "bad coords arg: expected list-of-pairs"
    },
    {
        "/args/coords",
        std::vector<int>{1, 2, 3},
        "bad coords arg: expected list-of-pairs"
    },
    {
        "/args/coords",
        std::vector<std::vector<int>>{{1, 2, 10}, {3, 4, 11}},
        "bad coords arg: expected list-of-pairs"
    },
    {
        "/manifest/line-numbers",
        std::vector<std::vector<int>>{{1, 2, 3, 4, 5}, {7, 9}, {100}, {0}},
        "operation requires 3-dimensional cube, but dimension was 4"
    },
    {
        "/manifest/data/0/shapes",
        std::vector<std::vector<int>>{{1}},
        "operation requires 3-dimensional fragments, but dimension was 1"
    }
};

TEMPLATE_TEST_CASE_SIG("unpacking a query with wrong key value fails", "",
                       ((typename T, int i), T, i),
                       (one::slice_query, 0),
                       (one::curtain_query, 1)) {

    const std::vector<std::initializer_list<badjson>> badquery = {
        badslice,
        badcurtain
    };

    auto [keypath, value, error] = GENERATE_REF(values<badjson>(badquery[i]));

    SECTION(fmt::format("when value for key '{}' is unexpected", keypath)) {
        const auto qs =
            fmt::format("{{ {}, {} }}", query_required, query_specific[i]);
        const auto bad_qs = update_json(qs, keypath, value);
        const auto doc = bad_qs.c_str();

        T query;
        CHECK_THROWS_WITH(query.unpack(doc, doc + std::strlen(doc)),
                          Contains(error));
    }
}

SCENARIO("Requests of different kinds return the same result for slice") {
    std::string query_specific = R"(
            "function": "slice",
    )";

    one::slice_query query;
    const auto verify = [&]() {
        WHEN("Unpacking the request") {
            const auto doc =
                fmt::format("{{ {}, {} }}", query_required, query_specific);
            query.unpack(doc.c_str(), doc.c_str() + doc.size());

            THEN("The values are unpacked correctly") {
                CHECK(query.dim == 2);
                CHECK(query.idx == 1);
            }
        }
    };

    GIVEN("Index value") {
        query_specific += R"(
            "args": {
                "kind": "index",
                "dim": 2,
                "val": 1
            }
        )";
        verify();
    }

    GIVEN("Lineno value") {
        query_specific += R"(
            "args": {
                "kind": "lineno",
                "dim": 2,
                "val": 34
            }
        )";
        verify();
    }
}

SCENARIO("Requests of different kinds return the same result for curtain") {
    const std::string query_main =
        fmt::sprintf(query_base_template, "[[10, 11], [1, 2], [0, 1]]",
                     R"(,"utm-to-lineno": [[1, 0, 10], [0, 1, 1]])");
    std::string query_specific = R"(
            "function": "curtain",
    )";

    one::curtain_query query;
    const auto verify = [&]() {
        WHEN("Unpacking the request") {
            const auto doc =
                fmt::format("{{ {}, {} }}", query_main, query_specific);
            query.unpack(doc.c_str(), doc.c_str() + doc.size());

            THEN("The coordinates are unpacked correctly") {
                CHECK(query.dim0s == std::vector{0, 1});
                CHECK(query.dim1s == std::vector{1, 0});
            }
        }
    };

    GIVEN("Index coordinates") {
        query_specific += R"(
            "args": {
                "kind": "index",
                "coords": [[0, 1], [1, 0]]
            }
        )";
        verify();
    }

    GIVEN("Lineno coordinates") {
        query_specific += R"(
            "args": {
                "kind": "lineno",
                "coords": [[10, 2], [11, 1]]
            }
        )";
        verify();
    }

    GIVEN("UTM coordinates") {
        query_specific += R"(
            "args": {
                "kind": "utm",
                "coords": [[0.1, 1.1], [0.9, -0.1]]
            }
        )";
        verify();
    }
}

TEST_CASE("packing a query is not supported") {
    const auto msg = "Packing is not implemented for query";
    one::slice_query slice_query;
    CHECK_THROWS_WITH(slice_query.pack(), Contains(msg));

    one::curtain_query curtain_query;
    CHECK_THROWS_WITH(curtain_query.pack(), Contains(msg));
}


TEST_CASE("slice-task can round trip packing") {
    one::slice_task task;
    task.pid = "pid";
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
