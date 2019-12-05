#include <string>
#include <chrono>
#include <cassert>

#include <fmt/core.h>
#include <fmt/chrono.h>
#include <fmt/ranges.h>
#include <gnutls/gnutls.h>
#include <gnutls/crypto.h>

#include <seismic-cloud/url.hpp>
#include "base64.h"

namespace sc {

/*
 * Some of these implementations are quick-and-dirty, just using fmtlib. They
 * can be significantly improved speed-wise by caching more, pre-allocating
 * more objects etc, as minor optimisations
 */

namespace {
constexpr const char* date_example = "x-ms-date:Day, dd Mon year HH:MM:SS GMT";
}

azure_request_generator::azure_request_generator(
    const std::string& account,
    const std::string& cube_id,
    const std::string& resolution,
    const std::string& fragment_dimension,
    const std::string& endpoint_suffix) noexcept (false) :
        baseurl(
            fmt::format("https://{}.{}/{}/{}/{}",
                account,
                endpoint_suffix,
                cube_id,
                resolution,
                fragment_dimension
            )
        ),
        account(account),
        // Use an example (but broken) timestamp to get the length right
        date(date_example),
        resource_id(
            fmt::format("{}/{}/{}/{}",
                account,
                cube_id,
                resolution,
                fragment_dimension)
        )
{
}

void azure_request_generator::timestamp() noexcept (false) {
    /*
     * https://docs.microsoft.com/en-us/rest/api/storageservices/representation-of-date-time-values-in-headers
     *
     * Azure storage services follow RFC 1123 for representation of date/time
     * values. This is the preferred format for HTTP/1.1 operations, as
     * described in section 3.3 of the HTTP/1.1 Protocol Parameters
     * specification. An example of this format is:
     *
     *      Sun, 06 Nov 1994 08:49:37 GMT
     */
    using clock = std::chrono::system_clock;
    const auto now = clock::to_time_t(clock::now());
    const auto* gmt = std::gmtime(&now);
    if (!gmt) {
        const auto msg = "unable to convert from system time to GMT";
        throw std::runtime_error(msg);
    }

    constexpr const char* fmtstr = "x-ms-date:{:%a, %d %b %Y %T} GMT";
    /*
     * date should be pre-allocated, and large enough to fit the date exactly
     *
     * TODO: micro performance can be gained by writing *only* the date, and
     * not the constant x-ms-date stuff
     */
    assert(this->date.size() == fmt::format(fmtstr, *gmt).size());
    fmt::format_to(this->date.begin(), fmtstr, *gmt);
}

void azure_request_generator::timestamp(const std::string& ts) noexcept (false) {
    this->date = "x-ms-date:" + ts;
}

void azure_request_generator::shared_key(const std::string& k) noexcept (false) {
    this->key = base64_decode(k);
}

std::string
azure_request_generator::authorization(const std::string& fragment_id) const
noexcept (false) {
    /*
     * https://docs.microsoft.com/en-us/rest/api/storageservices/authorize-with-shared-key
     *
     * The authorization must be created with new-lines for every missing header
     * (which for our GETs are most of them)
     */

    const auto request = fmt::format(
        "GET\n" /* HTTP Verb */
        "\n"    /* Content-Encoding */
        "\n"    /* Content-Language */
        "\n"    /* Content-Length */
        "\n"    /* Content-MD5 */
        "\n"    /* Content-Type */
        "\n"    /* Date */
        "\n"    /* If-Modified-Since  */
        "\n"    /* If-Match */
        "\n"    /* If-None-Match */
        "\n"    /* If-Unmodified-Since */
        "\n"    /* Range */
        "{}\n"  /* x-ms-headers */
        "{}",   /* resource to get, i.e. the blob */
        fmt::join(this->headers(), "\n"),
        this->resource(fragment_id));

    assert(
        !this->key.empty() &&
        "key not set - make sure to call shared_key()"
    );
    auto digest = std::array< unsigned char, 32 >{};
    const auto err = gnutls_hmac_fast(
        GNUTLS_MAC_SHA256,
        this->key.data(),
        this->key.size(),
        request.data(),
        request.size(),
        digest.data()
    );

    if (err) {
        throw std::runtime_error("unable to sign request...");
    }

    return fmt::format(
        "Authorization: SharedKey {}:{}",
        this->account,
        base64_encode(digest.data(), digest.size())
    );
}

std::string
azure_request_generator::resource(const std::string& fragment_id) const
noexcept (false) {
    /*
     * The resource ID for the blob is:
     *
     * /storage-account/container/blob-path
     *
     * This is *not* translatable to the URL (as the storage account goes as a
     * subdomain for for the endpoint), but is necessary to compute the
     * authorization
     */
    return fmt::format("/{}/{}.f32", this->resource_id, fragment_id);
}

std::array< const char*, 2 >
azure_request_generator::headers() const noexcept (true) {
    /*
     * timestamp() *must* be called before this - pick up on it in debug mode,
     * but save the trouble in release builds
     */
    assert(this->date != date_example);

    /*
     * The order is important - headers must be lexicographically sorted
     */
    return {
        this->date.c_str(),
        "x-ms-version:2018-11-09",
    };
}

std::string azure_request_generator::url(const std::string& fragment_id) const {
    return fmt::format("{}/{}.f32", this->baseurl, fragment_id);
}

}
