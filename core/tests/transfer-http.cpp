#include <chrono>
#include <cstring>
#include <memory>
#include <stdexcept>
#include <thread>

// microhttpd requires the headers for sockaddr_in etc to be included before
// microhttpd.h
#include <inttypes.h>
#include <arpa/inet.h>
#include <netinet/in.h>

#include <catch/catch.hpp>
#include <fmt/format.h>
#include <microhttpd.h>

#include <oneseismic/transfer.hpp>

using namespace Catch::Matchers;

TEST_CASE(
    "transfer rejects nonsensical max-connections",
    "[transfer]") {

    struct config : one::storage_configuration {
        std::string url(const one::batch&, const std::string&) const override {
            return "";
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
 * These tests work by spinning up an in-process http server, and doing
 * requests against it.
 */

struct mhttpd_stop {
    void operator () (MHD_Daemon* d) {
        if (d) MHD_stop_daemon(d);
    }
};

class mhttpd {
public:
    mhttpd(MHD_AccessHandlerCallback request, void* request_arg = nullptr) {
        /*
         * Randomly pick a port to use. Collision with running services is
         * quite unlikely.
         *
         * TODO: Allow setting a specific port through env vars or something.
         */
        const auto port = GENERATE(take(1, random(10000, 30000)));

        std::memset(&this->address, 0, sizeof(this->address));
        this->address.sin_family = AF_INET;
        this->address.sin_port = htons(port);
        this->address.sin_addr.s_addr = htonl(INADDR_LOOPBACK);

        constexpr const int ignored_port = 0;
        this->httpd.reset(
            MHD_start_daemon(
                  MHD_USE_THREAD_PER_CONNECTION
                | MHD_USE_INTERNAL_POLLING_THREAD
                ,
                ignored_port,
                nullptr,      /* access callback (ignored, see SOCK_ADDR) */
                nullptr,      /* access callback arg (ignored) */
                request,      /* request handler */
                request_arg,  /* request handler args */
                MHD_OPTION_SOCK_ADDR, (&this->address),
                MHD_OPTION_CONNECTION_TIMEOUT, (unsigned int) 2,
                MHD_OPTION_END
            )
        );

        INFO("Unable to bind httpd to localhost:" << port);
        REQUIRE(this->httpd);
    }

    std::uint16_t port() const {
        return ntohs(this->address.sin_port);
    }

private:
    sockaddr_in address;
    std::unique_ptr< MHD_Daemon, mhttpd_stop > httpd;
};

/*
 * Base-config for testing that generates loopback addresses. Set port to
 * whatever mhttpd is listening to.
 */
struct loopback_cfg : public one::storage_configuration {
    explicit loopback_cfg(int port) : port(port) {}

    virtual std::string url(
            const one::batch&,
            const std::string&) const override {
        return fmt::format("http://127.0.0.1:{}/", this->port);
    }

private:
    int port = 10000;
};

/*
 * The default implementations of oncomplete and onfailure is to fail the test,
 * so little extra code is needed when either is expected.
 */
struct always_fail : public one::transfer_configuration {
    void oncomplete(
            const one::buffer&,
            const one::batch& b,
            const std::string& id) override {
        FAIL(fmt::format("HTTP request for '{}' succeeded", id));
    }

    void onfailure(
            const one::buffer&,
            const one::batch& b,
            const std::string& id) override {
        FAIL(fmt::format("HTTP request for '{}' failed", id));
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
    "Transfer calls onfailure on HTTP error",
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
    struct count_failure : public always_fail {
        int called = 0;

        void onfailure(
                const one::buffer&,
                const one::batch&,
                const std::string&) override {
            this->called += 1;
        }
    } config;

    for (auto x : statuscodes) {
        SECTION(fmt::format("HTTP {}", x)) {
            mhttpd httpd(empty_response, &x);
            loopback_cfg storage(httpd.port());
            one::transfer xfer(max_connections, storage);

            one::batch batch;
            batch.fragment_ids.resize(1);
            xfer.perform(batch, config);
            CHECK(config.called == 1);
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
