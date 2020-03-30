#ifndef ONESEISMIC_TEST_HTTPD_HPP
#define ONESEISMIC_TEST_HTTPD_HPP

#include <string>
// microhttpd requires the headers for sockaddr_in etc to be included before
// microhttpd.h
#include <inttypes.h>
#include <arpa/inet.h>
#include <netinet/in.h>
#include <microhttpd.h>

#include <oneseismic/transfer.hpp>

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

    std::string url(
            const one::batch&,
            const std::string&) const override {
        return "http://127.0.0.1:" + std::to_string(this->port);
    }

    action onstatus(
            const one::buffer&,
            const one::batch&,
            const std::string&,
            long http_code) override {
        CHECK(http_code == 200);
        return action::done;
    }

private:
    int port = 10000;
};


#endif //ONESEISMIC_TEST_HTTPD_HPP
