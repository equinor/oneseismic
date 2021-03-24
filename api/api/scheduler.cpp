#include "scheduler.h"

#include <cstring>
#include <memory>
#include <numeric>

#include <oneseismic/plan.hpp>

namespace {

template < typename T >
char* copy(char* dst, const T& x) noexcept (true) {
    const auto len = x.size();
    std::memcpy(dst, x.data(), len);
    return dst + len;
}

}

tasks* mkschedule(const char* doc, int len, int task_size) {
    const auto packed = one::mkschedule(doc, len, task_size);

    struct deleter {
        void operator () (tasks* t) noexcept (true) {
            cleanup(t);
        }
    };

    const auto flat_tasksize = std::accumulate(
        packed.begin(),
        packed.end(),
        0,
        [](const int acc, const std::string& x) noexcept (true) {
            return acc + x.size();
        }
    );

    std::unique_ptr< tasks, deleter > cs(new tasks());
    cs->err = nullptr;
    cs->sizes = new int [packed.size()];
    cs->tasks = new char[flat_tasksize];
    cs->len   = packed.size();
    char* dst = cs->tasks;

    cs->sizes[0] = packed.back().size();
    dst = copy(dst, packed.back());
    for (std::size_t i = 0; i < packed.size() - 1; ++i) {
        cs->sizes[i + 1] = packed[i].size();
        dst = copy(dst, packed[i]);
    }

    return cs.release();
}

void cleanup(tasks* cs) {
    if (!cs) return;

    delete cs->err;
    delete cs->sizes;
    delete cs->tasks;
    delete cs;
}
