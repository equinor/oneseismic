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

curl_slist* az::http_headers(const std::string& authorization) const {
    const auto date    = one::x_ms_date();
    const auto version = one::x_ms_version();

    curl_headers headers;
    headers.append(date);
    headers.append(version);
    if (not authorization.empty()) {
        const auto auth = fmt::format("Authorization: {}", authorization);
        headers.append(auth);
    }
    return headers.release();
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

storage_configuration::action az::onstatus(
        const buffer& b,
        const batch&,
        const std::string& fragment_id,
        long status_code) {

    // https://docs.microsoft.com/en-us/rest/api/storageservices/blob-service-error-codes

    if (status_code == 200)
        return action::done;

    const auto response = std::string(b.begin(), b.end());
    switch (status_code) {
        case 403:
            throw unauthorized(response);

        case 404: {
            const auto msg = "Error fetching fragment {}, message: {}";
            throw notfound(fmt::format(msg, fragment_id, response));
        }

        case 500:
            /*
             * TODO: 500 means a problem with the blob store, and should be
             * reported as such
             */
        default: {
            const auto msg = "unhandled status code {}: {}";
            throw aborted(fmt::format(msg, status_code, response));
        }
    }
}

storage_configuration::action az::onstatus(long status_code) {
    if (status_code == 201)
        return action::done;

    switch (status_code) {
        case 403:
            throw unauthorized("");

        case 500:
            /*
             * TODO: 500 means a problem with the blob store, and should be
             * reported as such
             */
        default: {
            const auto msg = "unhandled status code {}";
            throw aborted(fmt::format(msg, status_code));
        }
    }
}

}
