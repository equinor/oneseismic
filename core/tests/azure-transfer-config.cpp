#include <chrono>
#include <ciso646>
#include <ctime>
#include <iomanip>
#include <sstream>
#include <string>

#include <catch/catch.hpp>

#include <oneseismic/azure.hpp>

using namespace Catch::Matchers;

TEST_CASE(
        "Configuration makes the correct URL",
        "[azure][az]") {
    const auto expected =
        "https://acc.blob.core.windows.net/guid/src/64-64-64/0-1-2.f32";

    one::batch batch;
    batch.guid = "guid";
    batch.storage_endpoint = "https://acc.blob.core.windows.net";
    batch.fragment_shape = "src/64-64-64";
    one::az az("");
    const auto url = az.url(batch, "0-1-2");
    CHECK(url == expected);
}

TEST_CASE(
        "x-ms-date starts with x-ms-date",
        "[azure][az]") {
    CHECK_THAT(one::x_ms_date(), StartsWith("x-ms-date:"));
}

TEST_CASE(
        "x-ms-version starts with x-ms-version",
        "[azure][az]") {
    CHECK_THAT(one::x_ms_version(), StartsWith("x-ms-version:"));
}

TEST_CASE(
        "x_ms_date creates RFC1123 formatted HTTP header",
        "[azure][az]") {
    std::tm tm = {};
    std::stringstream ss(one::x_ms_date());
    INFO("Was:             '" << ss.str() << "'");
    INFO("Expected format: 'x-ms-date:Mon, 24 Feb 2020 11:43:51 GMT'");
    ss >> std::get_time(&tm, "x-ms-date:%a, %d %b %Y %H:%M:%S GMT");
    CHECK(not ss.fail());
}

std::vector< std::string > to_vector(curl_slist* slist) {
    auto cur = slist;
    std::vector< std::string > headers;
    while (cur) {
        headers.push_back(cur->data);
        cur = cur->next;
    }
    curl_slist_free_all(slist);
    return headers;
}

TEST_CASE(
        "Bearer authorization is added as a http header",
        "[azure][az][http]") {
    one::az az("key");
    one::batch batch;
    batch.token = "$token";
    const auto headers = to_vector(az.http_headers(batch, "pid"));
    const auto expected = std::string("Authorization: Bearer $token");
    CHECK_THAT(headers, VectorContains(expected));
}
