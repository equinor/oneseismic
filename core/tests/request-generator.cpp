#include <catch/catch.hpp>

#include <seismic-cloud/url.hpp>

using namespace Catch::Matchers;

TEST_CASE("Azure generator respects the storage account parameter") {
    auto gen = sc::azure_request_generator(
        "storage-account",
        "1",
        "src",
        "1-1-1"
    );
    gen.timestamp("Wed, 14 Feb 2018 16:51:03 GMT");
    gen.shared_key("key");

    CHECK_THAT(gen.resource(""), StartsWith("/storage-account"));
    CHECK_THAT(gen.url(""), StartsWith("https://storage-account"));
    CHECK_THAT(
        gen.authorization(""),
        StartsWith("Authorization: SharedKey storage-account:")
    );
}

TEST_CASE("Azure generator creates the correct signature") {
    auto gen = sc::azure_request_generator(
        "storage-account",
        "1",
        "src",
        "1-1-1"
    );
    gen.timestamp("Wed, 14 Feb 2018 16:51:03 GMT");
    gen.shared_key("key");

    /*
     * This is a regression test - the key was obtained by querying blob store,
     * and getting a response. Then, the program was run against this known
     * key+timestamp, and the authorization key recorded.
     */
    const auto expected = "Authorization: SharedKey storage-account:"
                          "ONjimjNyIRGxOc3S7SAHu5Tuoj3p7zusBR3h1bPJka8=";
    CHECK(gen.authorization("0-0-0") == expected);
}

TEST_CASE("Azure generator respects the cube-id parameter") {
    auto gen = sc::azure_request_generator(
        "storage-account",
        "cube!",
        "src",
        "1-1-1"
    );
    gen.timestamp("Wed, 14 Feb 2018 16:51:03 GMT");
    gen.shared_key("key");

    const auto resource = gen.resource("");
    const auto resource_cube = resource.substr(resource.find('/', 1));
    CHECK_THAT(resource_cube, StartsWith("/cube!/"));
    CHECK_THAT(gen.url("1-1-1"), Contains("/cube!/"));
}

TEST_CASE("Azure generator respects the resolution parameter") {
    auto gen = sc::azure_request_generator(
        "storage-account",
        "1",
        "crs",
        "1-1-1"
    );
    gen.timestamp("Wed, 14 Feb 2018 16:51:03 GMT");
    gen.shared_key("key");

    CHECK_THAT(gen.resource(""), Contains("/crs/"));
    CHECK_THAT(gen.url("1-1-1"), Contains("/crs/"));
}

TEST_CASE("Azure generator respects the fragment dimension parameter") {
    auto gen = sc::azure_request_generator(
        "storage-account",
        "1",
        "src",
        "I-J-K"
    );
    gen.timestamp("Wed, 14 Feb 2018 16:51:03 GMT");
    gen.shared_key("key");

    CHECK_THAT(gen.resource(""), Contains("/I-J-K/"));
    CHECK_THAT(gen.url("1-1-1"), Contains("/I-J-K/"));
}

TEST_CASE("Azure generator creates the expected resource and url") {
    auto gen = sc::azure_request_generator(
        "storage-account",
        "cube-id",
        "src",
        "10-10-10"
    );
    gen.timestamp("Wed, 14 Feb 2018 16:51:03 GMT");
    gen.shared_key("key");

    const auto resource = "/storage-account/cube-id/src/10-10-10/2-3-5.f32";
    const auto url = "https://storage-account.blob.core.windows.net/cube-id/src/10-10-10/2-3-5.f32";
    CHECK(gen.resource("2-3-5") == resource);
    CHECK(gen.url("2-3-5") == url);
}
