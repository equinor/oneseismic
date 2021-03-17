#include "scheduler.h"

#include <cstdlib>
#include <cstring>
#include <memory>

#include <oneseismic/plan.hpp>

namespace {

template < typename T >
void* copyalloc(const T& x) {
    void* p = std::malloc(x.size() * sizeof(*x.data()));
    std::memcpy(p, x.data(), x.size());
    return p;
}

}

tasks* mkschedule(const char* doc, int len, int task_size) {
    const auto packed = one::mkschedule(doc, len, task_size);

    struct deleter {
        void operator () (tasks* t) noexcept (true) {
            cleanup(t);
        }
    };

    std::unique_ptr< tasks, deleter > cs(new tasks());
    cs->err = nullptr;
    cs->tasks = new task[packed.size()];
    cs->size = packed.size();

    auto& header = cs->tasks[0];
    header.size  = packed.back().size();
    header.task  = copyalloc(packed.back());
    for (std::size_t i = 0; i < packed.size() - 1; ++i) {
        auto& task = cs->tasks[i + 1];
        task.size = packed[i].size();
        task.task = copyalloc(packed[i]);
    }

    return cs.release();
}

void cleanup(tasks* cs) {
    if (!cs) return;

    for (int i = 0; i < cs->size; ++i) {
        std::free((void*)cs->tasks[i].task);
    }
    delete cs->err;
    delete cs->tasks;
    delete cs;
}
