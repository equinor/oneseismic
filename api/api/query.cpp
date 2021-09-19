#include "query.h"

#include <algorithm>
#include <cassert>
#include <cstdlib>
#include <cstring>
#include <memory>
#include <new>
#include <numeric>
#include <string>

#include <oneseismic/plan.hpp>

void cleanup(plan* p) {
    if (!p) return;

    delete[] p->err;
    delete[] p->sizes;
    delete[] p->tasks;
    *p = plan {};
}

struct session : public one::session {};

session* session_new() {
    return new (std::nothrow) session;
}

const char* session_init(session* self, const char* doc, int len) {
    try {
        self->init(doc, len);
        return nullptr;
    } catch (std::exception& e) {
        // using malloc is important; the str will be free'd by go
        char* msg = (char*)std::malloc(std::strlen(e.what()) + 1);
        std::strcpy(msg, e.what());
        return msg;
    }
}

plan session_plan_query(
    session* self,
    const char* doc,
    int len,
    int task_size)
try {
    const auto taskset = self->plan_query(doc, len, task_size);
    if (taskset.empty()) {
        throw one::bad_message("task-set should not be empty");
    }

    plan p {};
    p.len = taskset.count();
    p.tasks = new char[taskset.size()];
    p.sizes = new int [p.len];
    std::copy_n(taskset.sizes.begin(),  taskset.count(), p.sizes);
    std::copy_n(taskset.packed.begin(), taskset.size(),  p.tasks);
    return p;
} catch (std::exception& e) {
    plan p {};
    auto* err = new char[std::strlen(e.what()) + 1];
    std::strcpy(err, e.what());
    p.err = err;
    return p;
}
