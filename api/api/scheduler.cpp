#include "scheduler.h"

#include <algorithm>
#include <cassert>
#include <cstring>
#include <memory>
#include <numeric>
#include <string>

#include <oneseismic/plan.hpp>

namespace {

template < typename T >
char* copy(char* dst, const T& x) noexcept (true) {
    const auto len = x.size();
    std::memcpy(dst, x.data(), len);
    return dst + len;
}

void find_msg_sizes(
    const std::string& packed,
    int elems,
    int* dst
) noexcept (false) {
    /*
     * Compute the individual sizes of the messages by recording where the
     * \0 separators are
     *
     * The approach is most easily demonstrated with an example:
     *
     * aa0bbb0c0dd0
     *
     * pos: [2, 6, 8, 11]
     * len: [2, 3, 1, 2]
     *
     * adj-diff: [2 - 0, 6 - 2, 8 - 6, 11 - 8]
     * adj-diff: [2, 4, 2, 3]
     *
     * All elements but the first have an off-by-one because they count the
     * separator too, which is fixed by making the diff function (lhs-rhs)-1
     */

    assert(!packed.empty());
    const auto bad_count = [](auto i, auto elems) {
        return std::logic_error(
            "count() found "
            + std::to_string(elems) + "messages, "
            + "but find_msg_sizes got npos at elem = " + std::to_string(i)
        );
    };

    std::string::size_type pos = 0;
    for (int i = 0; i < elems; ++i) {
        pos = packed.find('\0', pos + 1);
        if (pos == std::string::npos)
            throw bad_count(i, elems);
        dst[i] = int(pos);
    }

    if (pos != packed.size() - 1)
        throw std::logic_error("find_msg_sizes did not exhaust input");
    const auto diff_strlen = [](auto x, auto y) noexcept (true) {
        return (x - y) - 1;
    };
    std::adjacent_difference(dst, dst + elems, dst, diff_strlen);
}

}

plan mkschedule(const char* doc, int len, int task_size) try {
    const auto packed = one::mkschedule(doc, len, task_size);
    if (packed.empty()) {
        throw one::bad_message("packed query should not be empty");
    }

    plan p {};
    p.status_code = 200;
    p.len = std::count(packed.begin(), packed.end(), '\0');
    p.tasks = new char[packed.size()];
    p.sizes = new int [p.len];
    std::remove_copy(packed.begin(), packed.end(), p.tasks, '\0');
    find_msg_sizes(packed, p.len, p.sizes);
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
