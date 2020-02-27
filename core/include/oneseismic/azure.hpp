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

private:
    std::string storage_account;
    std::string key;
};

}

#endif // ONESEISMIC_AZURE_HPP
