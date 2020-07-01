// make gmtime_r available from time.h
#define _POSIX_C_SOURCE 1
#define _XOPEN_SOURCE
#include <time.h>

#include <string>
#include <chrono>
#include <cassert>

#include <curl/curl.h>
#include <fmt/format.h>
#include <fmt/chrono.h>
#include <gnutls/gnutls.h>
#include <gnutls/crypto.h>

#include <oneseismic/azure.hpp>
#include "base64.h"

namespace one {

std::string x_ms_date() noexcept (false) {
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

    /* gmtime is not thread safe, so use the posix gtime_r */
    tm gmt_storage;
    const auto* gmt = gmtime_r(&now, &gmt_storage);
    if (!gmt) {
        const auto msg = "unable to convert from system time to GMT";
        throw std::runtime_error(msg);
    }

    constexpr const char* fmtstr = "x-ms-date:{:%a, %d %b %Y %T} GMT";
    /*
     * TODO: micro performance can be gained by writing *only* the date, and
     * not the constant x-ms-date stuff
     */
    return fmt::format(fmtstr, *gmt);
}

az::az(std::string acc, std::string key) :
    storage_account(std::move(acc)),
    key(base64_decode(key))
{}

std::string az::sign(
        const std::string& date,
        const std::string& version,
        const one::batch& batch,
        const std::string& fragment_id)
const noexcept (false) {

    const auto canonical_resource = this->canonicalized_resource(
        batch.root,
        batch.guid,
        batch.fragment_shape,
        fragment_id
    );

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
        "{}\n"  /* x-ms-date */
        "{}\n"  /* x-ms-version */
        "{}",   /* resource to get, i.e. the blob */
        date,
        version,
        canonical_resource
    );

    assert(!this->key.empty() && "az.key is empty");
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
        batch.root,
        base64_encode(digest.data(), digest.size())
    );
}

curl_slist* az::http_headers(
        const one::batch& batch,
        const std::string& job) const {
    const auto date = one::x_ms_date();
    const auto version = one::x_ms_version();
    const auto auth = this->sign(date, version, batch, job);

    // TODO: address leak and flag errors here
    curl_slist* headers = nullptr;
    headers = curl_slist_append(headers, date.c_str());
    headers = curl_slist_append(headers, version);
    headers = curl_slist_append(headers, auth.c_str());
    return headers;
}

std::string az::url(const one::batch& batch, const std::string& id) const {
    return fmt::format(
        "{}/{}/{}/{}.f32",
        batch.storage_endpoint,
        batch.guid,
        batch.fragment_shape,
        id
    );
}

std::string az::canonicalized_resource(
        const std::string& root,
        const std::string& guid,
        const std::string& fragment_shape,
        const std::string& fragment_id)
const noexcept (false) {
    /*
     * TODO: this could be and URLs be a dynamic config instead of virtual
     * dispatch?
     */
    return fmt::format(
        "/{}/{}/{}/{}.f32",
        root,
        guid,
        fragment_shape,
        fragment_id
    );
}

storage_configuration::action az::onstatus(
        const buffer&,
        const batch&,
        const std::string& fragment_id,
        long status_code) {

    if (status_code != 200)
        throw aborted("az: status code was not 200/OK");

    return action::done;
}

}
