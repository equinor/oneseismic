#ifndef ONESEISMIC_SRC_LOG_HPP
#define ONESEISMIC_SRC_LOG_HPP

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
        fmt::print(stderr, "[{%F %T}] {}: " + fmt,
            system_clock::to_time_t(system_clock::now()),
            Module::name(),
            std::forward< T >(ts)...
        );
    }
};

}

#endif //ONESEISMIC_SRC_LOG_HPP
