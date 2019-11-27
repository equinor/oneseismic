#include <curl/curl.h>

#include <cstdio>
#include <string>
#include <vector>
#include <memory>

namespace {
 
std::size_t write_cb(char* data, size_t n, size_t l, void* userp) {
    /* take care of the data here, ignored in this example */ 
    (void)data;
    (void)userp;
    //std::puts("downloaded a blob");
    return n*l;
}
 
void add_transfer(CURLM *cm, const char* url) noexcept (true) {
    CURL *eh = curl_easy_init();
    curl_easy_setopt(eh, CURLOPT_WRITEFUNCTION, write_cb);
    curl_easy_setopt(eh, CURLOPT_URL, url);
    curl_easy_setopt(eh, CURLOPT_PRIVATE, url);
    curl_multi_add_handle(cm, eh);
}

struct globalcurl {
    globalcurl() noexcept (true) {
        ::curl_global_init(CURL_GLOBAL_ALL);
    };

    ~globalcurl() noexcept (true) {
        curl_global_cleanup();
    }
};

struct multi_clean {
    void operator () (CURLM* ptr) {
        curl_multi_cleanup(ptr);
    }
};

std::unique_ptr< CURLM, multi_clean > multicurl() noexcept (true) {
    return std::unique_ptr< CURLM, multi_clean >{ ::curl_multi_init() };
}

}
 
int main(int args, char** argv) {
    CURLMsg *msg;
    unsigned int transfers = 0;
    int msgs_left = -1;
    int still_alive = 1;

    const auto max_parallel = std::stol(argv[1]);
 
    globalcurl gl;
    auto ucm = multicurl();
    auto* cm = ucm.get();
    
    /* Limit the amount of simultaneous connections curl should allow: */ 
    curl_multi_setopt(cm, CURLMOPT_MAXCONNECTS, max_parallel);
 
    const auto urls = std::vector< std::string > {
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-0-0.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-0-1.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-0-2.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-0-3.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-0-4.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-0-5.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-0-6.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-0-7.f32",

        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-1-0.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-1-1.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-1-2.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-1-3.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-1-4.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-1-5.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-1-6.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-1-7.f32",

        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-2-0.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-2-1.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-2-2.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-2-3.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-2-4.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-2-5.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-2-6.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-2-7.f32",

        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-3-0.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-3-1.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-3-2.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-3-3.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-3-4.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-3-5.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-3-6.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-3-7.f32",

        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-4-0.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-4-1.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-4-2.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-4-3.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-4-4.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-4-5.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-4-6.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-4-7.f32",

        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-5-0.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-5-1.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-5-2.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-5-3.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-5-4.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-5-5.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-5-6.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/0-5-7.f32",


        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-0-0.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-0-1.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-0-2.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-0-3.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-0-4.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-0-5.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-0-6.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-0-7.f32",

        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-1-0.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-1-1.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-1-2.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-1-3.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-1-4.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-1-5.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-1-6.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-1-7.f32",

        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-2-0.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-2-1.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-2-2.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-2-3.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-2-4.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-2-5.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-2-6.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-2-7.f32",

        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-3-0.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-3-1.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-3-2.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-3-3.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-3-4.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-3-5.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-3-6.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-3-7.f32",

        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-4-0.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-4-1.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-4-2.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-4-3.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-4-4.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-4-5.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-4-6.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-4-7.f32",

        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-5-0.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-5-1.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-5-2.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-5-3.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-5-4.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-5-5.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-5-6.f32",
        "https://segyngrmdfdev.blob.core.windows.net/fragments/999f3ee0a7fb9889aa4d00d7a9d660caab9dc24851aa686cfe9754b28604cf67/src/120-120-120/1-5-7.f32",
    };

    for(transfers = 0; transfers < max_parallel; transfers++)
        add_transfer(cm, urls.at(transfers).c_str());
 
    do {
        curl_multi_perform(cm, &still_alive);

        while ((msg = curl_multi_info_read(cm, &msgs_left))) {
            if (msg->msg == CURLMSG_DONE) {
                char *url;
                CURL *e = msg->easy_handle;
                curl_easy_getinfo(msg->easy_handle, CURLINFO_PRIVATE, &url);
                fprintf(stderr, "R: %d - %s <%s>\n",
                        msg->data.result, curl_easy_strerror(msg->data.result), url);
                curl_multi_remove_handle(cm, e);
                curl_easy_cleanup(e);
            }
            else {
                fprintf(stderr, "E: CURLMsg (%d)\n", msg->msg);
            }
            if(transfers < urls.size())
                add_transfer(cm, urls.at(transfers++).c_str());
        }
        if (still_alive)
            curl_multi_wait(cm, NULL, 0, 1000, NULL);

    } while(still_alive || (transfers < urls.size()));
 
  return EXIT_SUCCESS;
}
