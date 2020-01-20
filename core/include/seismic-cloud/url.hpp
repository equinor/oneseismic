#ifndef SEISMIC_CLOUD_URL_HPP
#define SEISMIC_CLOUD_URL_HPP

#include <array>
#include <string>
#include <vector>

#include <seismic-cloud/geometry.hpp>

namespace sc {

class azure_request_generator {
public:
    azure_request_generator(
        const std::string& account,
        const std::string& cube_id,
        const std::string& resolution,
        const std::string& fragment_dimension,
        const std::string& endpoint_suffix = "blob.core.windows.net"
    ) noexcept (false);

    /*
     * Update the timestamp to now()
     */
    void timestamp() noexcept (false);

    /*
     * Update the timestamp to ts - this is unsuited for production, but quite
     * useful for development, as it enables reproducible requests and
     * signatures.
     */
    void timestamp(const std::string& ts) noexcept (false);

    /*
     * Set the shared-key to the storage account
     */
    void shared_key(const std::string&) noexcept (false);

    /*
     * Get the authorization header which should be the last header added onto
     * the request before getting a blob
     */
    std::string authorization(const std::string&) const noexcept (false);

    /*
     * Obtain the resource-ID
     */
    std::string resource(const std::string&) const noexcept (false);

    /*
     * The x-ms-* headers which must be set on the request
     */
    std::vector< std::string > headers() const noexcept (true);

    /*
     * The full url to GET a fragment
     */
    std::string url(const std::string& fragmentid) const noexcept (false);

private:
    /*
     * The base URI of the resource
     * https://account.endpoint/cube/resolution/dimension
     */
    std::string baseurl;
    std::string account;
    std::string date;
    std::string resource_id;
    std::string key;
};

}

#endif //SEISMIC_CLOUD_URL_HPP
