#include "scheduler.h"

#include <algorithm>
#include <cassert>
#include <cstring>
#include <memory>
#include <numeric>
#include <string>

#include <oneseismic/plan.hpp>

plan mkschedule(const char* doc, int len, int task_size) try {
    const auto taskset = one::mkschedule(doc, len, task_size);
    if (taskset.empty()) {
        throw one::bad_message("task-set should not be empty");
    }

    plan p {};
    p.status_code = 200;
    p.len = taskset.count();
    p.tasks = new char[taskset.size()];
    p.sizes = new int [p.len];
    std::copy_n(taskset.sizes.begin(),  taskset.count(), p.sizes);
    std::copy_n(taskset.packed.begin(), taskset.size(),  p.tasks);
    return p;
} catch (std::exception& e) {
    plan p {};
    p.status_code = 500;
    auto* err = new char[std::strlen(e.what()) + 1];
    std::strcpy(err, e.what());
    p.err = err;
    return p;
}

void cleanup(plan* p) {
    if (!p) return;

    delete p->err;
    delete p->sizes;
    delete p->tasks;
    *p = plan {};
}
