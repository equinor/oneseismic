#include <memory>
#include <string>

#include <oneseismic/geometry.hpp>
#include <oneseismic/messages.hpp>
#include <oneseismic/process.hpp>

#include "tasks.h"

struct proc {
    std::unique_ptr< one::proc > p;
    std::string errmsg;
};

proc* newproc(const char* kind) try {
    auto up = std::make_unique< proc >();
    up->p = one::proc::make(kind);

    // Unknown kind, bail out
    if (!up->p) return nullptr;

    return up.release();
} catch (...) {
    return nullptr;
}

void cleanup(proc* p) {
    if (p) delete p;
}

const char* errmsg(proc* p) {
    return p->errmsg.c_str();
}

bool init(proc* p, const void* msg, int len) {
    try {
        p->p->init(static_cast< const char* >(msg), len);
        return true;
    } catch (std::exception& e) {
        p->errmsg = e.what();
        return false;
    }
}

const char* fragments(proc* p) {
    return p->p->fragments().c_str();
}

bool add(proc* p, int index, const void* chunk, int len) {
    try {
        p->p->add(index, static_cast< const char* >(chunk), len);
        return true;
    } catch (std::exception& e) {
        p->errmsg = e.what();
        return false;
    }
}

packed pack(proc* p) {
    packed pd;
    auto* pp = p->p.get();
    try {
        pp->packed = pp->pack();
        pd.err = false;
        pd.size = pp->packed.size();
        pd.body = pp->packed.data();
    } catch (std::exception& e) {
        p->errmsg = e.what();
        pd.err = true;
    }
    return pd;
}
