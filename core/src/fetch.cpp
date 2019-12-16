#include <algorithm>
#include <ciso646>
#include <cstdint>
#include <future>
#include <string>
#include <vector>

#include <curl/curl.h>
#include <fmt/core.h>

#include <seismic-cloud/fetch.hpp>

namespace sc {

namespace {

size_t write(std::uint8_t* data,
             std::size_t n,
             std::size_t l,
             std::vector< std::uint8_t >* out) {
    out->insert(out->end(), data, data + (n * l));
    return n * l;
}

}

multifetch::multifetch(int max_connections) :
    cm(curl_multi_init()),
    max_connections(max_connections) {
    if (not this->cm)
        throw curl_error("unable to init curl multi-handle");
}

multifetch::~multifetch() {
    for (const auto& e : this->transfers)
        this->remove_easy_handle(e.get());
}

std::future< std::vector< std::uint8_t > >
multifetch::enqueue(const std::string& url,
                    const std::vector< std::string >& headers) {
    /*
     * this function is a bit elaborate with how it manages resources, to
     * maintain exception safety.
     */
    auto transfer = unique_curl(curl_easy_init());
    auto* e = transfer.get();
    if (not e) {
        const auto msg = fmt::format("unable to init easy handle for {}", url);
        throw curl_init_error(msg);
    }

    headers_curl tmpheaders = nullptr;
    for (const auto& h : headers) {
        auto tmp = curl_slist_append(tmpheaders.get(), h.c_str());
        if (not tmp)
            throw curl_error(fmt::format("unable to set header '{}'", h));

        tmpheaders.release();
        tmpheaders.reset(tmp);
    }

    auto payload = std::make_unique< transfer_payload >();
    auto future = payload->promise.get_future();
    payload->url = url;

    curl_easy_setopt(e, CURLOPT_URL, url.c_str());
    curl_easy_setopt(e, CURLOPT_HTTPHEADER, tmpheaders.get());
    curl_easy_setopt(e, CURLOPT_WRITEFUNCTION, write);
    curl_easy_setopt(e, CURLOPT_WRITEDATA, &payload->buffer);
    curl_easy_setopt(e, CURLOPT_PRIVATE, payload.release());
    this->headers.push_back(std::move(tmpheaders));
    this->pending_transfers.push_back(std::move(transfer));
    return future;
}

void multifetch::run() {
    while (true) {
        const auto status = this->enqueue_pending_transfer();
        switch (status) {
            case enqueue_status::enqueued:
                continue;

            case enqueue_status::no_pending:
            case enqueue_status::max_parallel:
                break;
        }

        break;
    }

    int msgs_in_queue = -1;
    int still_alive = -1;
    do {
        const auto err = curl_multi_perform(this->cm.get(), &still_alive);
        if (err) {
            /*
             * handle error, maybe restart connection, maybe something else
             * entirely
             */
            throw curl_error(err);
        }

        CURLMsg* msg;
        while ((msg = curl_multi_info_read(this->cm.get(), &msgs_in_queue))) {
            if (msg->msg == CURLMSG_DONE) {
                auto* e = msg->easy_handle;
                assert(e);

                transfer_payload* payload;
                curl_easy_getinfo(e, CURLINFO_PRIVATE, &payload);
                payload->promise.set_value(std::move(payload->buffer));
                this->remove_easy_handle(e);
            } else {
                /* TODO: figure out the message, maybe restart */
                throw curl_error("transfer failed");
            }

            this->enqueue_pending_transfer();
        }

        if (still_alive) {
            curl_multi_wait(
                this->cm.get(),
                nullptr,
                0,
                this->timeout_ms,
                nullptr
            );
        }
    } while (still_alive
             or not this->transfers.empty()
             or not this->pending_transfers.empty());
}

multifetch::enqueue_status multifetch::enqueue_pending_transfer() {
    if (this->pending_transfers.empty())
        return enqueue_status::no_pending;

    if (int(this->transfers.size()) >= this->max_connections)
        return enqueue_status::max_parallel;

    auto* e = this->pending_transfers.back().get();
    transfer_payload* payload;
    const auto err = curl_easy_getinfo(e, CURLINFO_PRIVATE, &payload);
    if (err) {
        throw curl_error(err);
    }
    payload->start = std::chrono::high_resolution_clock::now();

    this->transfers.push_back(std::move(this->pending_transfers.back()));
    this->pending_transfers.pop_back();
    const auto errm = curl_multi_add_handle(this->cm.get(), e);
    if (errm) {
        throw curl_error(errm);
    }
    return enqueue_status::enqueued;
}

void multifetch::remove_easy_handle(CURL* e) {
    const auto cmp = [&](const auto& lhs) noexcept (true) {
        return lhs.get() == e;
    };
    auto itr = std::find_if(
        this->transfers.begin(),
        this->transfers.end(),
        cmp
    );
    assert(itr != this->transfers.end() &&
           "tried to remove non-existant handle");
    if (itr == this->transfers.end())
        return;

    const auto err = curl_multi_remove_handle(this->cm.get(), e);
    if (err)
        throw curl_cleanup_error(err);
    this->transfers.erase(itr);
}

fetch::fetch() {
    const auto err = curl_global_init(CURL_GLOBAL_ALL);
    if (err)
        throw curl_init_error(err);
}

fetch::~fetch() {
    curl_global_cleanup();
}

}
