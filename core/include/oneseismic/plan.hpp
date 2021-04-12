#ifndef ONESEISMIC_PLAN_HPP
#define ONESEISMIC_PLAN_HPP

#include <exception>
#include <string>
#include <vector>

namespace one {

struct not_found : public std::out_of_range {
    using std::out_of_range::out_of_range;
};

std::vector< std::string >
mkschedule(const char* doc, int len, int task_size) noexcept (false);

}

#endif //ONESEISMIC_PLAN_HPP
