#define CATCH_CONFIG_RUNNER
#include <catch/catch.hpp>

#include <curl/curl.h>
#include <spdlog/spdlog.h>

#include "config.hpp"

static std::string redis_addr;

std::string redisaddr() {
    if (redis_addr.empty()) {
        const auto msg = "redis address undefined; set with --redis host:port";
        throw std::runtime_error(msg);
    }
    return redis_addr;
}

/*
 * Use own defined main, because that's catch2's way of allowing custom global
 * init [1]. spdlog is turned off because otherwise test output gets very
 * chatty, and not as useful.
 *
 * [1] https://github.com/catchorg/Catch2/blob/master/docs/own-main.md
 */
int main(int argc, char** argv) {
    spdlog::set_level(spdlog::level::off);
    curl_global_init(CURL_GLOBAL_ALL);

    Catch::Session session;
    auto cli = session.cli()
    #if defined(INTEGRATION_TESTS_REDIS)
        | Catch::clara::Opt(redis_addr, "host:port")
            ["--redis"]
            ("Address to running redis instance")
    #endif
    ;
    session.cli(cli);
    return session.run(argc, argv);
}
