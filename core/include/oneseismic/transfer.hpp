#ifndef ONESEISMIC_TRANSFER_HPP
#define ONESEISMIC_TRANSFER_HPP

#include <chrono>
#include <cstdint>
#include <memory>
#include <stdexcept>
#include <string>
#include <vector>

#include <curl/curl.h>
#include <hiredis/hiredis.h>

namespace one {

/*
 * The batch is a pre-grouped set of fragments to be scheduled for a single
 * transfer task. A job partitioner parses a request for a slice, surface, or
 * some other structure, and figures out the fragments that needs fetching.
 *
 * With this design comes the fundamental restriction that all fragments in a
 * single batch must belong to the same cube, given by guid and shape.
 */
struct batch {
    std::string storage_endpoint; /* url including storage-account */
    /*
     * Authorization. For HTTP, this is typically 'Bearer $token', i.e. without
     * the Authorization: key, but with authorization type
     */
    std::string auth;
    std::string guid;
    std::string fragment_shape; /* src/64-64-64 */

    /* IDs of the fragments to fetch */
    std::vector< std::string > fragment_ids;
};

using buffer = std::vector< std::uint8_t >;

class aborted : public std::runtime_error {
public:
    aborted(const std::string& reason) : std::runtime_error(reason) {}
    aborted(const char*        reason) : std::runtime_error(reason) {}
};

class notfound : public std::runtime_error {
public:
    notfound(const std::string& reason) : std::runtime_error(reason) {}
    notfound(const char*        reason) : std::runtime_error(reason) {}
};

class unauthorized : public std::runtime_error {
public:
    unauthorized(const std::string& reason) : std::runtime_error(reason) {}
    unauthorized(const char*        reason) : std::runtime_error(reason) {}
};

class storage_error : public std::runtime_error {
public:
    storage_error(const std::string& reason) : std::runtime_error(reason) {}
    storage_error(const char*        reason) : std::runtime_error(reason) {}
};

/*
 * Different backends (e.g. azure, local file system) need to configure
 * transfers differently: set different headers, generate different urls,
 * timeouts.
 *
 * The methods of the configuration class correspond to customization points in
 * the transfer operation.
 */
class storage_configuration {
public:
    /*
     * Transfer timeout, which signals when a task has failed, and a batch is
     * aborted. Note the timeout is set for every transfer and not total
     * transfer time for a batch.
     *
     * Defaults to 0 (= no timeout)
     */
    virtual std::chrono::milliseconds timeout() const noexcept (true) {
        return std::chrono::seconds(0);
    }

    /*
     * Return a curl_slist which will be given to curl CURLOPT_HTTPHEADER [1].
     * If this returns nullptr, no headers are changed for the connection.
     *
     * [1] https://curl.haxx.se/libcurl/c/CURLOPT_HTTPHEADER.html
     */
    virtual curl_slist* http_headers(const std::string& authorization) const {
        return nullptr;
    }

    /*
     * Create a url for the fragment-id
     */
    virtual std::string url(
            const batch&,
            const std::string& fragment_id) const = 0;

    /*
     * Create a url for an object at path. This give storage configurations the
     * opportunity to intercept, decorate or otherwise play modify the URL
     * before an actual request is made.
     *
     * URL hijacking is particularly useful for logging and testing, so all
     * URLs should preferably go through this function, even if it is the
     * identity
     */
    virtual std::string url(const std::string& path) const {
        return path;
    }

    /*
     * Check the status code and decide what to do for the in-progress
     * transfer.  What exactly is the right choice depends both on back-end,
     * responsibility, and run-time config. For example, simple retrying is
     * pointless for file systems, usually the right choice a couple of times
     * for cloud storage.
     *
     * To abort a transfer, throw the aborted exception.
     */

    enum class action {
        done,
        retry,
    };

    virtual action onstatus(
            const buffer&,
            const batch&,
            const std::string& fragment_id,
            long status_code) = 0;

    /*
     * onstatus for PUT/writes
     */
    virtual action onstatus(long status_code) = 0;

    virtual ~storage_configuration() = default;
};

class transfer_configuration {
public:
    /*
     * Called on successful transfer, if onstatus returns done. This function
     * is called before the handle is released. Buffer data is still owned by
     * transfer, so copy it if you need to keep it around.
     */
    virtual void oncomplete(
            const buffer&,
            const batch&,
            const std::string& fragment_id) {
    }

    virtual ~transfer_configuration() = default;
};

/*
 * Thin RAII layer on top of the curl_slist
 */
class curl_headers {
public:
    curl_headers() = default;
    explicit curl_headers(curl_slist* l);

    void append(const std::string& header) noexcept (false);
    void append(const char* header) noexcept (false);

    const curl_slist* get() const noexcept (true);
    void set(curl_slist* l) noexcept (true);

    curl_slist* release() noexcept (true);

private:
    struct slist_deleter {
        void operator () (curl_slist* l) noexcept (true) {
            curl_slist_free_all(l);
        }
    };

    std::unique_ptr< curl_slist, slist_deleter > headers;
};

/*
 */
class transfer {
public:
    /*
     * Max concurrent connections can only be set during construction, and
     * controls the number of curl_easy_handle created. Transfer does *not*
     * make a copy of the transfer_configuration, so it must be kept alive for
     * the lifetime of the transfer object.
     */
    transfer(int max_connections, storage_configuration&);
    ~transfer();

    transfer(const transfer&) = delete;
    transfer& operator = (      transfer&) = delete;
    transfer& operator = (const transfer&) = delete;

    transfer(transfer&&) = default;
    transfer& operator = (transfer&&) = default;

    /*
     * perform is blocking, and fetches all jobs described by batch in a
     * concurrent manner. No guarantees are given for order of fetches or
     * completion, except that transfer_configuration.oncomplete will be
     * invoked as soon as possible.
     *
     * oncomplete will be called in the same thread as transfer.perform. If it
     * is an expensive operation, or you want future-like semantics, this is
     * the place to do wake-up.
     */
    void perform(batch, transfer_configuration&);

    void put(
            const std::string& path,
            const char* data,
            std::size_t size,
            const std::string& auth,
            const std::string& type = "application/json"
    );

private:
    struct task {
        buffer storage;
        std::string fragment_id;
        curl_headers headers;
    };

    CURLM* multi;
    std::vector< CURL* > idle;
    std::vector< CURL* > connections;
    std::vector< task >  tasks;
    storage_configuration& config;

    void schedule(const batch&, std::string);
};

class working_storage {
public:
    /*
     * Connect to the storage. This must be called before put().
     */
    void connect(const std::string& addr)           noexcept (false);
    void connect(const std::string& host, int port) noexcept (false);

    /*
     * Put { key: binary-string } in storage
     */
    void put(const std::string& key, const std::string&) noexcept (false);
    void put(const char*        key, const std::string&) noexcept (false);

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
