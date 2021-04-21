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

plan mkschedule(const char* doc, int len, int task_size) {
    plan p {};
    std::vector< std::string > packed;
    try {
        packed = one::mkschedule(doc, len, task_size);
    } catch (one::not_found& e) {
        p.status_code = 404;
        auto* err = new char[std::strlen(e.what()) + 1];
        std::strcpy(err, e.what());
        p.err = err;
        return p;
    } catch (std::exception& e) {
        p.status_code = 500;
        auto* err = new char[std::strlen(e.what()) + 1];
        std::strcpy(err, e.what());
        p.err = err;
        return p;
    }

    const auto flat_tasksize = std::accumulate(
        packed.begin(),
        packed.end(),
        0,
        [](const int acc, const std::string& x) noexcept (true) {
            return acc + x.size();
        }
    );

    p.status_code = 200;
    p.sizes = new int [packed.size()];
    p.tasks = new char[flat_tasksize];
    p.len   = packed.size();
    char* dst = p.tasks;

    p.sizes[0] = packed.back().size();
    dst = copy(dst, packed.back());
    for (std::size_t i = 0; i < packed.size() - 1; ++i) {
        p.sizes[i + 1] = packed[i].size();
        dst = copy(dst, packed[i]);
    }

    return p;
}

void cleanup(plan* p) {
    if (!p) return;

    delete p->err;
    delete p->sizes;
    delete p->tasks;
    *p = plan {};
}
