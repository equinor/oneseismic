#ifndef SEISMIC_CLOUD_FETCH_HPP
#define SEISMIC_CLOUD_FETCH_HPP

#include <chrono>
#include <exception>
#include <future>
#include <memory>
#include <vector>

#include <curl/curl.h>

namespace sc {

/*
 * curl error code -> C++ exception with automatic string-reporting in
 * exception message. Later it might be a good idea to augment this with local
 * context and information from where the exception was thrown, but for now
 * it's probably fine to just echo whatever curl says.
 */
struct curl_error : public std::runtime_error {
    using std::runtime_error::runtime_error;
    curl_error(CURLcode err)  : std::runtime_error(curl_easy_strerror(err)) {}
    curl_error(CURLMcode err) : std::runtime_error(curl_multi_strerror(err)) {}
};

struct curl_init_error : public curl_error {
    using curl_error::curl_error;
};

struct curl_cleanup_error : public curl_error {
    using curl_error::curl_error;
};

/*
 * Buffer and transfer metadata - this must be kept alive for as long as the
 * handle is alive, but this teardown is handled by the custom easy-handle
 * destructor.
 */
struct transfer_payload {
    std::vector< std::uint8_t > buffer;
    std::promise< decltype(buffer) > promise;
    std::string url;

    std::chrono::time_point<std::chrono::high_resolution_clock> start;
};


struct CURL_deleter {
    void operator () (CURL* e) const noexcept (true) {
        if (not e) return;

        transfer_payload* p;
        curl_easy_getinfo(e, CURLINFO_PRIVATE, &p);
        delete p;
        curl_easy_cleanup(e);
    }
};

struct CURLM_deleter {
    void operator () (CURLM* m) const noexcept (true) {
        if (m) curl_multi_cleanup(m);
    }
};

struct slist_deleter {
    void operator () (curl_slist* l) const noexcept (true) {
        curl_slist_free_all(l);
    }
};

using unique_curl = std::unique_ptr< CURL, CURL_deleter >;
using headers_curl = std::unique_ptr< curl_slist, slist_deleter >;

/*
 * A fairly straight-forward c++ification of the curl_multi_handle, but with
 * some extra features for lifetime management and maybe easier names to work
 * with.
 */
class multifetch {
public:
    multifetch(int max_connections);
    ~multifetch();

    std::future< std::vector< std::uint8_t > >
    enqueue(const std::string& url, const std::vector< std::string >& headers);

    void run();

private:
    std::vector< unique_curl > transfers;
    std::vector< unique_curl > pending_transfers;
    std::vector< headers_curl > headers;
    /*
     * the curl multi handle must be deleted *after* transfers and pending
     * transfers, so keep it after in the members-list
     */
    std::unique_ptr< CURLM, CURLM_deleter > cm;
    int max_connections;
    /*
     * not externally configurable yet, as it's unclear how useful it is to
     * set, instead of just using curl's default timeouts
     */
    int timeout_ms = 1000;

    enum class enqueue_status {
        enqueued,
        no_pending,
        max_parallel,
    };

    enqueue_status enqueue_pending_transfer();
    void remove_easy_handle(CURL* e);
};

class fetch {
public:
    fetch();
    ~fetch();
};

}

#endif //SEISMIC_CLOUD_FETCH_HPP
