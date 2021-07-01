#ifndef ONESEISMIC_PLAN_HPP
#define ONESEISMIC_PLAN_HPP

#include <exception>
#include <string>
#include <vector>

#include <oneseismic/messages.hpp>

namespace one {

std::vector< std::string >
mkschedule(const char* doc, int len, int task_size) noexcept (false);

}

#endif //ONESEISMIC_PLAN_HPP
