#ifndef ONESEISMIC_TRANSFER_HPP
#define ONESEISMIC_TRANSFER_HPP

#include <chrono>
#include <cstdint>
#include <memory>
#include <stdexcept>
#include <string>
#include <vector>

#include <curl/curl.h>

namespace one {

/*
 * The batch is a pre-grouped set of fragments to be scheduled for a single
 * transfer task. A job partitioner parses a request for a slice, surface, or
 * some other structure, and figures out the fragments that needs fetching.
 *
 * With this design comes the fundamental restriction that all fragments in a
 * single batch must belong to the same cube, given by root, guid, and shape.
 */
struct batch {
    std::string root; /* storage-account in azure */
    std::string storage_endpoint; /* url including storage-account */
    std::string authorization;    /* For azure this is "Bearer $token" */
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
    virtual curl_slist* http_headers(
            const batch&,
            const std::string& fragment_id) const {
        return nullptr;
    }

    /*
     * Create a url for the fragment-id
     */
    virtual std::string url(
            const batch&,
            const std::string& fragment_id) const = 0;

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

private:
    struct slist_free {
        void operator () (curl_slist* list) {
            curl_slist_free_all(list);
        }
    };

    struct task {
        buffer storage;
        std::string fragment_id;
        std::unique_ptr< curl_slist, slist_free > headers;
    };

    CURLM* multi;
    std::vector< CURL* > idle;
    std::vector< CURL* > connections;
    std::vector< task >  tasks;
    storage_configuration& config;

    void schedule(const batch&, std::string);
};

}

#endif //ONESEISMIC_TRANSFER_HPP
