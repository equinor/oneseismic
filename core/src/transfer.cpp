#include <cassert>
#include <ciso646>
#include <cstdint>
#include <stdexcept>
#include <string>
#include <vector>

#include <curl/curl.h>
#include <fmt/format.h>
#include <fmt/chrono.h>

#include <oneseismic/transfer.hpp>

namespace one {

namespace {

std::size_t append(
    std::uint8_t* data,
    std::size_t n,
    std::size_t l,
    buffer* f) {

    /*
     * A simple write/callback function that buffers up a fragment
     */

    f->insert(f->end(), data, data + (n * l));
    return n * l;
}

int check_multi_error(CURLMcode rc) noexcept (false) {
    /*
     * Check the error code, and return zero if all is well, and non-zero if
     * multi perform needs to be called again. All other errors raise an
     * appropriate exception.
     */
    switch (rc) {
        case CURLM_OK:
            return 0;

        case CURLM_CALL_MULTI_PERFORM:
            /* from the manual: CURLM_CALL_MULTI_PERFORM (-1)
             *
             * This is not really an error. It means you should call
             * curl_multi_perform again without doing select() or similar
             * in between. Before version 7.20.0 this could be returned by
             * curl_multi_perform, but in later versions this return code
             * is never used.
             */
            return 1;

        case CURLM_BAD_HANDLE:
        case CURLM_BAD_EASY_HANDLE:
            throw std::logic_error(curl_multi_strerror(rc));

        case CURLM_OUT_OF_MEMORY:
        case CURLM_INTERNAL_ERROR:
            throw std::runtime_error(curl_multi_strerror(rc));

        case CURLM_BAD_SOCKET:
        case CURLM_UNKNOWN_OPTION:
            throw std::logic_error(curl_multi_strerror(rc));

        /*
         * These enum values are added in a newer version of libcurl than the
         * one availble off-the-shelf on RHEL7. They're mostly logic errors, so
         * just defer them to the default case for newer libcurls - they're all
         * fatal anyway.
         *
         * case CURLM_WAKEUP_FAILURE:
         *     throw std::runtime_error(curl_multi_strerror(rc));

         * case CURLM_BAD_FUNCTION_ARGUMENT:
         * case CURLM_ADDED_ALREADY:
         * case CURLM_RECURSIVE_API_CALL:
         *     throw std::logic_error(curl_multi_strerror(rc));
         */

        default:
            throw std::logic_error(
                fmt::format("Unknown curl error: {}", curl_multi_strerror(rc))
            );
    }
}

class recoverable_curl_error : public std::runtime_error {
public:
    recoverable_curl_error(CURLcode rc) :
        std::runtime_error(curl_easy_strerror(rc)),
        rc(rc) {
    }

    CURLcode code() const noexcept (true) {
        return this->rc;
    }

private:
    CURLcode rc;
};

class fatal_curl_error : public std::runtime_error {
public:
    fatal_curl_error(CURLcode rc) :
        std::runtime_error(curl_easy_strerror(rc)),
        rc(rc) {
    }

    CURLcode code() const noexcept (true) {
        return this->rc;
    }

private:
    CURLcode rc;
};

void check_easy_error(CURLcode rc) noexcept (false) {
    switch (rc) {
        case CURLE_OK:
            return;

        case CURLE_FAILED_INIT:
            throw fatal_curl_error(rc);

        case CURLE_OPERATION_TIMEDOUT:
        case CURLE_COULDNT_RESOLVE_PROXY:
        case CURLE_COULDNT_RESOLVE_HOST:
        case CURLE_COULDNT_CONNECT:
        case CURLE_SSL_CONNECT_ERROR:
        case CURLE_FILE_COULDNT_READ_FILE:
            throw recoverable_curl_error(rc);

        case CURLE_HTTP_RETURNED_ERROR:
            throw std::logic_error(
                fmt::format("It is assumed CURLOPT_FAILONERROR is false. "
                            "CURL message: {}", curl_easy_strerror(rc))
            );


        default:
            throw std::logic_error(
                fmt::format("CURL error ({}): {}", rc, curl_easy_strerror(rc))
            );
    }
}

/*
 * Template this, even though it's always transfer::task, to avoid having to
 * specify it as friend, or declare it in the header, since task is a private
 * inner class in transfer.
 */
template < typename T >
T* getprivate(CURL* e) noexcept (false) {
    T* t;
    const auto rcpriv = curl_easy_getinfo(e, CURLINFO_PRIVATE, &t);
    if (rcpriv != CURLE_OK) {
        const auto errmsg = "curl_easy_getinfo(CURLINFO_PRIVATE) failed"
                            ", task* not set?";
        throw std::logic_error(errmsg);
    }
    assert(t);
    return t;
}

long response_code(CURL* e) noexcept (false) {
    long http_code;
    const auto rc = curl_easy_getinfo(e, CURLINFO_RESPONSE_CODE, &http_code);

    if (rc != CURLE_OK)
        throw fatal_curl_error(rc);

    /*
     * From the manual:
     * The stored value will be zero if no server response code has been
     * received.
     *
     * The response code should only be read when the transfer is *complete*,
     * so if this is zero something else is terribly wrong.
     */
    if (http_code == 0)
        throw std::runtime_error("No HTTP response code from server");

    return http_code;
}

}

transfer::transfer(int max_connections, transfer_configuration& xc) :
    multi(curl_multi_init()),
    config(xc) {

    if (max_connections <= 0) {
        const auto msg = "Expected max-connections (which is {}) > 0";
        throw std::invalid_argument(fmt::format(msg, max_connections));
    }

    if (!this->multi)
        throw std::runtime_error("unable to init curl multi handle");

    this->tasks.resize(max_connections);
    for (int i = 0; i < max_connections; ++i) {
        auto* e = curl_easy_init();
        if (!e)
            throw std::runtime_error("unable to init curl handle");

        const auto timeout = std::chrono::milliseconds(this->config.timeout());
        auto rc = curl_easy_setopt(e, CURLOPT_TIMEOUT_MS, timeout.count());
        if (rc != CURLE_OK) {
            const auto msg = "Unable to set timeout {}";
            throw std::invalid_argument(fmt::format(msg, timeout));
        }

        curl_easy_setopt(e, CURLOPT_PRIVATE,   &this->tasks.at(i));
        curl_easy_setopt(e, CURLOPT_WRITEDATA, &this->tasks.at(i).storage);
        curl_easy_setopt(e, CURLOPT_WRITEFUNCTION, append);
        this->connections.push_back(e);
        this->idle.push_back(e);
    }
}


transfer::~transfer() {
    for (auto e : this->connections) {
        curl_multi_remove_handle(this->multi, e);
        curl_easy_cleanup(e);
    }

    curl_multi_cleanup(this->multi);
}


void transfer::perform(batch batch) {
    /*
     * batch is copied for now, since jobs are scheduled by just popping
     * fragments to fetch from the end. In the future this might be done with
     * an iterator or something instead, in which case we can save ourselves
     * the copy, but this is an API compatible change and should be fine to do later.
     */
    while (true) {
        if (batch.fragment_ids.empty())
            break;
        if (this->idle.empty())
            break;

        assert(not batch.fragment_ids.empty());
        this->schedule(batch, batch.fragment_ids.back());
        batch.fragment_ids.pop_back();
    }

    int msgs_left = -1;
    int still_alive = 1;

    do {
        const auto call_perform = check_multi_error(
            curl_multi_perform(this->multi, &still_alive)
        );

        if (call_perform) continue;

        CURLMsg* msg;
        while ((msg = curl_multi_info_read(multi, &msgs_left))) {
            if (msg->msg != CURLMSG_DONE) {
                const auto errmsg = fmt::format("E: CURLMsg {}", msg->msg);
                throw std::runtime_error(errmsg);
            }

            try {
                check_easy_error(msg->data.result);
            } catch (const recoverable_curl_error& e) {
                /* TODO: log properly */
                throw;
            }
            auto* e = msg->easy_handle;
            /*
             * From the curl manual:
             * WARNING: The data the returned pointer points to will not
             * survive calling curl_multi_cleanup, curl_multi_remove_handle
             * or curl_easy_cleanup.
             *
             * don't access msg after this!
             */
            check_multi_error(curl_multi_remove_handle(this->multi, e));
            idle.push_back(e);

            const auto* t = getprivate< task >(e);
            const auto http_code = response_code(e);

            if (http_code == 200) {
                this->config.oncomplete(t->storage, batch, t->fragment_id);
            } else {
                this->config.onfailure(t->storage, batch, t->fragment_id);
            }

            if (not batch.fragment_ids.empty()) {
                this->schedule(batch, batch.fragment_ids.back());
                batch.fragment_ids.pop_back();
            }
        }

        if (still_alive) {
            const auto call_perform = check_multi_error(
                curl_multi_wait(this->multi, nullptr, 0, 1000, nullptr)
            );

            if (call_perform) continue;
        }

    } while (still_alive
        or not batch.fragment_ids.empty()
        or this->idle.size() < this->connections.size()
    );

    // TODO: if this happens in release, at least log properly
    assert(this->idle.size() == this->connections.size());
}

void transfer::schedule(const batch& batch, std::string fragment_id) {
    assert(not this->idle.empty());
    auto* e = this->idle.back();
    auto* t = getprivate< task >(e);
    t->fragment_id = fragment_id;
    t->storage.clear();

    const auto url = this->config.url(batch, fragment_id);
    curl_easy_setopt(e, CURLOPT_URL, url.c_str());

    auto* headers = this->config.http_headers(batch, fragment_id);
    if (headers) {
        t->headers.reset(headers);
        const auto rc = curl_easy_setopt(e, CURLOPT_HTTPHEADER, headers);

        if (rc != CURLE_OK) {
            assert(false && "CURL or handle does not support HTTP");
            throw std::logic_error("HTTP is not supported");
        }
    }

    check_multi_error(curl_multi_add_handle(this->multi, e));
    this->idle.pop_back();
}

}
