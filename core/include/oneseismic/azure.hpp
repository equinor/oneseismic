#ifndef ONESEISMIC_AZURE_HPP
#define ONESEISMIC_AZURE_HPP

#include <string>

#include <oneseismic/transfer.hpp>

namespace one {

std::string x_ms_date() noexcept (false);

inline constexpr const char* x_ms_version() noexcept (true) {
    return "x-ms-version:2018-11-09";
}

class az : public one::storage_configuration {
public:
    az(std::string acc, std::string k);

    std::string sign(
        const std::string& date,
        const std::string& version,
        const one::batch&,
        const std::string&)
    const noexcept (false);

    curl_slist* http_headers(
            const one::batch&,
            const std::string&)
        const override;

    std::string url(
            const one::batch&,
            const std::string&)
        const override;

    /*
     * A reasonable default azure configuration - anything but 200/OK gives a
     * runtime error and aborts the transfer. Retrying and other fancy stuff
     * can come later.
     */
    action onstatus(
            const buffer&,
            const batch&,
            const std::string& fragment_id,
            long status_code)
        override;

private:
    std::string storage_account;
    std::string key;
};

}

#endif // ONESEISMIC_AZURE_HPP
