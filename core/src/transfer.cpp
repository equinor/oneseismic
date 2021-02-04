#include <cassert>
#include <ciso646>
#include <cstdint>
#include <stdexcept>
#include <string>
#include <vector>

#include <fmt/format.h>
#include <fmt/chrono.h>
#include <spdlog/spdlog.h>

#include <oneseismic/transfer.hpp>

namespace one {

void working_storage::connect(const std::string& address) noexcept (false) {
    /*
     * TODO: parse addr -> (host, port) in separate, testable function
     */
    const auto default_port = 6379;
    auto colonpos = address.rfind(':');
    int port = default_port;
    if (colonpos != std::string::npos) {
        auto portstr = address.substr(colonpos + 1, std::string::npos);
        port = std::stoi(portstr);
    }
    const auto host = address.substr(0, colonpos);
    this->connect(host, port);
}

void working_storage::connect(
        const std::string& host,
        int port)
noexcept (false) {
    std::unique_ptr< redisContext, deleter > new_ctx(
        redisConnect(host.c_str(), port)
    );

    if (not new_ctx) {
        const auto msg = "unable to create redis context in working_storage";
        throw std::runtime_error(msg);
    }

    if (new_ctx->err) {
        const auto msg = "unable to connect to redis {}:{}: {}";
        throw std::runtime_error(
            fmt::format(msg, host, port, new_ctx->errstr)
        );
    }

    this->host = host;
    this->port = port;
    this->ctx.swap(new_ctx);
}

struct redis_reply_deleter {
    /*
     * It's not documented if freeReplyObject() accepts nullptr. In the
     * implementation (HEAD 2. nov 2020) it does, but until this is specified
     * it's worth guarding against nullptr.
     */
    void operator () (void* reply) {
        if (reply) freeReplyObject(reply);
    }
};

using unique_reply = std::unique_ptr< void, redis_reply_deleter >;

void working_storage::put(
        const std::string& key,
        const std::string& val)
noexcept (false) {
    this->put(key.c_str(), val);
}

void working_storage::put(
        const char* key,
        const std::string& val)
noexcept (false) {
    if (not this->ctx) {
        /*
         * If ctx isn't set then either some previous request failed, e.g.
         * because of network outage, or connect() hasn't been called. If
         * connect() has been called, then host should be set.
         */
        if (this->host.empty()) {
            const auto msg = "put() called before connect()";
            throw std::logic_error(msg);
        }

        this->connect(this->host, this->port);
    }

    /*
     * Read count as long long to make sure the count isn't truncated. While
     * hiredis doesn't specify it in the docs, it supports the %lld formatter
     * [1]. It's unlikely that any operator has given malicios expirations that
     * overflow int32, but there's little harm in widening to long long anyway.
     *
     * [1] in HEAD nov 2. 2020
     * https://github.com/redis/hiredis/blob/e3f88ebcf830323db32a33b908d67a617caf83e4/hiredis.c#L428
     */
    const long long exp = this->exp.count();
    auto* c = this->ctx.get();
    auto reply = unique_reply(
        redisCommand(c, "SET %s %b EX %lld", key, val.data(), val.size(), exp)
    );

    /*
     * hiredis docs
     * ------------
     * The return value of redisCommand holds a reply when the command was
     * successfully executed. When an error occurs, the return value is NULL
     * and the err field in the context will be set (see section on Errors).
     * Once an error is returned the context cannot be reused and you should
     * set up a new connection.
     */
    if (reply) {
        /*
         * hiredis docs
         * ------------
         *  The standard replies that redisCommand are of the type redisReply.
         *  The type field in the redisReply should be used to test what kind
         *  of reply was received.
         */
        auto* r = static_cast< redisReply* >(reply.get());
        switch (r->type) {
            case REDIS_REPLY_ERROR:
                throw storage_error(r->str);

            /*
             * Unless it's an error, we don't actually care what the reply was,
             * so just ignore it.
             *
             * It's unclear if REPLY_RETURN also means the redisConnection
             * should be re-initialised, but for now operate under the
             * assumption that a successful error reply means the connection is
             * fine.
             *
             * Switching on the set of known replies is super paranoid, but
             * should catch (in debug) if new return types are added to
             * hiredis. This should only matter in the case of new error codes,
             * but that way new error signals should not go through unhandled.
             */
            case REDIS_REPLY_STATUS:
            case REDIS_REPLY_INTEGER:
            case REDIS_REPLY_NIL:
            case REDIS_REPLY_STRING:
            case REDIS_REPLY_ARRAY:

            /*
             * RESP3, since v1.0.0
             */
        #if (HIREDIS_MAJOR >= 1)
            case REDIS_REPLY_DOUBLE:
            case REDIS_REPLY_BOOL:
            case REDIS_REPLY_MAP:
            case REDIS_REPLY_SET:
            case REDIS_REPLY_PUSH:
            case REDIS_REPLY_ATTR:
            case REDIS_REPLY_BIGNUM:
            case REDIS_REPLY_VERB:
        #endif
                return;

            default:
                assert(false && "Unhandled redis reply type");
                return;
        }

        return;
    }

    /*
     * hiredis docs
     * ------------
     * Once an error is returned the context cannot be reused and you should
     * set up a new connection.
     */
    const std::string errmsg = ctx->errstr;
    this->ctx.release();
    throw storage_error(errmsg);
}

std::string working_storage::get(const std::string& key) noexcept (false) {
    return this->get(key.c_str());
}

std::string working_storage::get(const char* key) noexcept (false) {
    if (not this->ctx) {
        /*
         * If ctx isn't set then either some previous request failed, e.g.
         * because of network outage, or connect() hasn't been called. If
         * connect() has been called, then host should be set.
         */
        if (this->host.empty()) {
            const auto msg = "get() called before connect()";
            throw std::logic_error(msg);
        }

        this->connect(this->host, this->port);
    }

    auto* c = this->ctx.get();
    auto reply = unique_reply(redisCommand(c, "GET %s", key));

    if (reply) {
        auto* r = static_cast< redisReply* >(reply.get());
        switch (r->type) {
            case REDIS_REPLY_ERROR:
                throw storage_error(r->str);

            /*
             * GET only supports getting strings
             */
            case REDIS_REPLY_STRING:
                return std::string(r->str, r->len);

            /*
             * Unless it's an error, we don't actually care what the reply was,
             * so just ignore it.
             *
             * It's unclear if REPLY_RETURN also means the redisConnection
             * should be re-initialised, but for now operate under the
             * assumption that a successful error reply means the connection is
             * fine.
             *
             * Switching on the set of known replies is super paranoid, but
             * should catch (in debug) if new return types are added to
             * hiredis. This should only matter in the case of new error codes,
             * but that way new error signals should not go through unhandled.
             */
            case REDIS_REPLY_STATUS:
            case REDIS_REPLY_INTEGER:
            case REDIS_REPLY_NIL:
            case REDIS_REPLY_ARRAY:

            /*
             * RESP3, since v1.0.0
             */
        #if (HIREDIS_MAJOR >= 1)
            case REDIS_REPLY_DOUBLE:
            case REDIS_REPLY_BOOL:
            case REDIS_REPLY_MAP:
            case REDIS_REPLY_SET:
            case REDIS_REPLY_PUSH:
            case REDIS_REPLY_ATTR:
            case REDIS_REPLY_BIGNUM:
            case REDIS_REPLY_VERB:
        #endif
                throw std::runtime_error(
                    "unexpected GET reply type; only STRING supported"
                );

            default:
                assert(false && "Unhandled redis reply type");
                throw std::logic_error("Unexpected reply type");
        }
    }

    const std::string errmsg = ctx->errstr;
    this->ctx.release();
    throw storage_error(errmsg);
}

}
