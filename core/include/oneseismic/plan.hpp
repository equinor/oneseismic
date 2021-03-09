#ifndef ONESEISMIC_PLAN_HPP
#define ONESEISMIC_PLAN_HPP

#include <vector>

namespace one {

std::vector< std::vector< char > >
mkschedule(const char* doc, int len, int task_size) noexcept (false);

}

#endif //ONESEISMIC_PLAN_HPP
