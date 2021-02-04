#ifndef ONESEISMIC_TRANSFER_HPP
#define ONESEISMIC_TRANSFER_HPP

#include <chrono>
#include <cstdint>
#include <memory>
#include <stdexcept>
#include <string>
#include <vector>

#include <hiredis/hiredis.h>

namespace one {

class storage_error : public std::runtime_error {
public:
    storage_error(const std::string& reason) : std::runtime_error(reason) {}
    storage_error(const char*        reason) : std::runtime_error(reason) {}
};

class working_storage {
public:
    /*
     * Connect to the storage. This function should be called before any read
     * or write operations.
     *
     * The connect(addr) function will parse the string as host[:port],
     * defaulting the port to redis' default 6379 if omitted.
     */
    void connect(const std::string& addr)           noexcept (false);
    void connect(const std::string& host, int port) noexcept (false);

    /*
     * Put { key: binary-string } in storage
     */
    void put(const std::string& key, const std::string&) noexcept (false);
    void put(const char*        key, const std::string&) noexcept (false);

    std::string get(const std::string& key) noexcept (false);
    std::string get(const char*        key) noexcept (false);

    /*
     * Set expiration for all objects. It sets the (minimum) time a result is
     * available after it was requested. A reasonably short expiration is
     * needed to ensure that permissions changes are detected, to reduce memory
     * pressure on the storage, and to reduce the chance for data leaks.
     */
    static constexpr std::chrono::seconds
    default_expiration = std::chrono::minutes(10);

    void expiration(std::chrono::seconds) noexcept (false);

private:
    struct deleter {
        void operator () (redisContext* p) {
            if (p) redisFree(p);
        }
    };

    std::unique_ptr< redisContext, deleter > ctx;
    std::chrono::seconds exp = default_expiration;
    std::string host;
    int port;
};

}

#endif //ONESEISMIC_TRANSFER_HPP
