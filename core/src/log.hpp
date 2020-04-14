#ifndef ONESEISMIC_SRC_LOG_HPP
#define ONESEISMIC_SRC_LOG_HPP

// make gmtime_r available from time.h
#ifndef _POSIX_C_SOURCE
    #define _POSIX_C_SOURCE 1
#endif
#ifndef _XOPEN_SOURCE
    #define _XOPEN_SOURCE
#endif
#include <time.h>

#include <chrono>
#include <utility>

#include <fmt/format.h>
#include <fmt/chrono.h>

namespace one {

template < typename Module >
class basic_log {
public:
    using system_clock = std::chrono::system_clock;

    template < typename... T >
    static void log(const std::string& fmt, T&& ... ts) noexcept (false) {
        const auto now = system_clock::to_time_t(system_clock::now());
        tm gmt_storage;
        const auto* gmt = gmtime_r(&now, &gmt_storage);

        if (!gmt) {
            const auto msg = "unable to convert from system time to GMT";
            throw std::runtime_error(msg);
        }

        fmt::print(stderr, "[{:%F %T}] {}: " + fmt,
            *gmt,
            Module::name(),
            std::forward< T >(ts)...
        );
    }
};

}

#endif //ONESEISMIC_SRC_LOG_HPP
