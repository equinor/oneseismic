#include <chrono>
#include <cstring>
#include <memory>
#include <stdexcept>
#include <thread>

#include <catch/catch.hpp>
#include <fmt/format.h>

#include <oneseismic/transfer.hpp>

#include "mhttpd.hpp"

using namespace Catch::Matchers;

TEST_CASE(
    "transfer rejects nonsensical max-connections",
    "[transfer]") {

    struct config : one::storage_configuration {
        std::string url(const one::batch&, const std::string&) const override {
            return "";
        }

        action onstatus(
                const one::buffer&,
                const one::batch& b,
                const std::string& id,
                long http_code) override {
            const auto msg = "HTTP request with status {} for '{}'";
            FAIL(fmt::format(msg, http_code, id));
            throw std::logic_error("FAIL() not invoked as it should");
        }

        action onstatus(long http_code) override {
            const auto msg = "HTTP request with status {}";
            FAIL(fmt::format(msg, http_code));
            throw std::logic_error("FAIL() not invoked as it should");
        }
    } cfg;

    CHECK_THROWS_AS(one::transfer( 0, cfg), std::invalid_argument);
    CHECK_THROWS_AS(one::transfer(-1, cfg), std::invalid_argument);
}

namespace {

int empty_response(
        void* statusc,
        struct MHD_Connection* connection,
        const char*,
        const char*,
        const char*,
        const char*,
        size_t*,
        void** ptr) {

    /*
     * Even empty responses must be explicitly crafted with microhttpd
     */
    char empty[] = "";
    auto* response = MHD_create_response_from_buffer(
            0,
            empty,
            MHD_RESPMEM_MUST_COPY
    );

    const auto* status = static_cast< unsigned int* >(statusc);
    auto ret = MHD_queue_response(connection, *status, response);
    MHD_destroy_response(response);
    return ret;
}

/*
 * The default implementation of oncomplete is to fail the test. This to avoid
 * accidental passes when the wrong handler is being invoked.
 */
struct always_fail : public one::transfer_configuration {

    void oncomplete(
            const one::buffer&,
            const one::batch& b,
            const std::string& id) override {
        FAIL("oncomplete() called unexpectedly");
    }
};

}

TEST_CASE(
    "Transfer calls oncomplete on HTTP 200/OK",
    "[transfer][http]") {

    struct count_complete : public always_fail {
        int called = 0;

        void oncomplete(
                const one::buffer&,
                const one::batch&,
                const std::string&) override {
            this->called += 1;
        }
    } action;

    const auto max_connections = GENERATE(1, 2, 3, 5);
    unsigned int OK = MHD_HTTP_OK;
    mhttpd httpd(empty_response, &OK);
    loopback_cfg storage(httpd.port());
    one::transfer xfer(max_connections, storage);

    one::batch batch;
    batch.fragment_ids.resize(1);
    xfer.perform(batch, action);
    CHECK(action.called == 1);
}

TEST_CASE(
    "Transfer does not call oncomplete on HTTP error",
    "[transfer][http]") {

    constexpr unsigned int statuscodes[] = {
        MHD_HTTP_BAD_REQUEST,
        MHD_HTTP_UNAUTHORIZED,
        MHD_HTTP_PAYMENT_REQUIRED,
        MHD_HTTP_FORBIDDEN,
        MHD_HTTP_NOT_FOUND,
        MHD_HTTP_METHOD_NOT_ALLOWED,
        MHD_HTTP_NOT_ACCEPTABLE,
        MHD_HTTP_PROXY_AUTHENTICATION_REQUIRED,
        MHD_HTTP_REQUEST_TIMEOUT,
        MHD_HTTP_CONFLICT,
        MHD_HTTP_GONE,
        MHD_HTTP_LENGTH_REQUIRED,
        MHD_HTTP_PRECONDITION_FAILED,
        MHD_HTTP_PAYLOAD_TOO_LARGE,
        MHD_HTTP_URI_TOO_LONG,
        MHD_HTTP_UNSUPPORTED_MEDIA_TYPE,
        MHD_HTTP_RANGE_NOT_SATISFIABLE,
        MHD_HTTP_EXPECTATION_FAILED,
        MHD_HTTP_UNPROCESSABLE_ENTITY,
        MHD_HTTP_LOCKED,
        MHD_HTTP_FAILED_DEPENDENCY,
        MHD_HTTP_UNORDERED_COLLECTION,
        MHD_HTTP_UPGRADE_REQUIRED,
        MHD_HTTP_NO_RESPONSE,
        MHD_HTTP_RETRY_WITH,
        MHD_HTTP_BLOCKED_BY_WINDOWS_PARENTAL_CONTROLS,
        MHD_HTTP_UNAVAILABLE_FOR_LEGAL_REASONS,

        MHD_HTTP_INTERNAL_SERVER_ERROR,
        MHD_HTTP_NOT_IMPLEMENTED,
        MHD_HTTP_BAD_GATEWAY,
        MHD_HTTP_SERVICE_UNAVAILABLE,
        MHD_HTTP_GATEWAY_TIMEOUT,
        MHD_HTTP_HTTP_VERSION_NOT_SUPPORTED,
        MHD_HTTP_VARIANT_ALSO_NEGOTIATES,
        MHD_HTTP_INSUFFICIENT_STORAGE,
        MHD_HTTP_BANDWIDTH_LIMIT_EXCEEDED,
        MHD_HTTP_NOT_EXTENDED,
    };

    const auto max_connections = GENERATE(1, 2, 3, 5);
    struct count_failure : public loopback_cfg {
        using loopback_cfg::loopback_cfg;

        int called = 0;

        action onstatus(
                const one::buffer&,
                const one::batch&,
                const std::string&,
                long http_code) override {
            CHECK(http_code != 200);
            this->called += 1;
            throw one::aborted("Error code != 200/OK");
        }
    };

    for (auto x : statuscodes) {
        SECTION(fmt::format("HTTP {}", x)) {
            mhttpd httpd(empty_response, &x);
            count_failure storage(httpd.port());
            one::transfer xfer(max_connections, storage);

            one::batch batch;
            batch.fragment_ids.resize(1);
            always_fail config;
            CHECK_THROWS_AS(xfer.perform(batch, config), one::aborted);
            CHECK(storage.called == 1);
        }
    }
}

TEST_CASE(
    "Transfer accepts and completes more jobs than max-connections",
    "[transfer][http]") {
    struct count_complete : public always_fail {
        int called = 0;

        void oncomplete(
                const one::buffer&,
                const one::batch&,
                const std::string&) override {
            this->called += 1;
        }
    } config;

    const auto connections = GENERATE(1, 2, 3, 5);
    unsigned int OK = MHD_HTTP_OK;
    mhttpd httpd(empty_response, &OK);
    loopback_cfg storage(httpd.port());
    one::transfer xfer(connections, storage);

    const auto jobs = GENERATE(take(5, random(7, 21)));
    one::batch batch;
    batch.fragment_ids.resize(jobs);
    INFO(fmt::format("Running {} jobs on {} connections", jobs, connections));
    xfer.perform(batch, config);
    CHECK(config.called == jobs);
}

using namespace std::chrono_literals;

TEST_CASE(
    "Timed-out requests fails the batch",
    "[transfer][http]") {
    const auto slow_response = [] (
        void*,
        struct MHD_Connection*,
        const char*,
        const char*,
        const char*,
        const char*,
        size_t*,
        void** ptr) {

        std::this_thread::sleep_for(20ms);
        unsigned int OK = MHD_HTTP_OK;
        return empty_response(
            &OK,
            nullptr,
            nullptr,
            nullptr,
            nullptr,
            nullptr,
            nullptr,
            nullptr
        );

        return 0;
    };

    /*
     * Set a 10ms timeout - this should make the tests go fast enough, while
     * give the http server plenty of time to make the test behave like a real system.
     */
    struct timeoutcfg : public loopback_cfg {
        using loopback_cfg::loopback_cfg;

        std::chrono::milliseconds timeout() const noexcept (true) override {
            return 10ms;
        }
    };

    const auto connections = GENERATE(1, 2, 3, 5);
    mhttpd httpd(slow_response);
    timeoutcfg storage(httpd.port());
    one::transfer xfer(connections, storage);

    one::batch batch;
    batch.fragment_ids.resize(2);
    always_fail action;
    CHECK_THROWS_WITH(xfer.perform(batch, action), Contains("Timeout"));
}

TEST_CASE(
    "Custom headers are added to the fetch request",
    "[transfer][http]") {

    const auto header_accumulate_response = [] (
        void*,
        struct MHD_Connection* conn,
        const char*,
        const char*,
        const char*,
        const char*,
        size_t*,
        void** ptr) {

        auto gather = [] (
                void* heads,
                MHD_ValueKind kind,
                const char* key,
                const char* value) {
            REQUIRE(kind == MHD_HEADER_KIND);
            auto headers = static_cast< std::vector< std::string >* >(heads);
            headers->push_back(fmt::format("{}: {}", key, value));
            return MHD_YES;
        };

        std::vector< std::string > headers;
        MHD_get_connection_values(conn, MHD_HEADER_KIND, gather, &headers);
        const auto expected = std::string("Custom: Expected");
        CHECK_THAT(headers, VectorContains(expected));

        unsigned int OK = MHD_HTTP_OK;
        return empty_response(
            &OK,
            conn,
            nullptr,
            nullptr,
            nullptr,
            nullptr,
            nullptr,
            nullptr
        );
    };

    struct succeed : public always_fail {
        void oncomplete(
                const one::buffer&,
                const one::batch&,
                const std::string&) override {
            /* no-op */
        }
    } action;

    mhttpd httpd(header_accumulate_response);
    struct loopback_with_headers : loopback_cfg {
        using loopback_cfg::loopback_cfg;
        curl_slist* http_headers(const std::string&) const override {
            return curl_slist_append(nullptr, "Custom: Expected");
        }
    } config(httpd.port());

    one::batch batch;
    batch.fragment_ids.resize(2);
    one::transfer(1, config).perform(batch, action);
}
