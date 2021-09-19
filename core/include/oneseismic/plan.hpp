#ifndef ONESEISMIC_PLAN_HPP
#define ONESEISMIC_PLAN_HPP

#include <exception>
#include <memory>
#include <string>
#include <vector>

#include <oneseismic/messages.hpp>

namespace one {

struct taskset {
    std::vector< int >  sizes;
    std::vector< char > packed;

    bool empty() const noexcept (true) {
        return this->sizes.empty();
    }

    std::size_t count() const noexcept (true) {
        return this->sizes.size();
    }

    std::size_t size() const noexcept (true) {
        return this->packed.size();
    }

    void reserve(int ntasks) {
        // rough guess that all tasks are less or approx 12kb, to reduce the
        // number of reallocs happening
        constexpr const auto estimated_task_size = 12000;
        this->sizes.reserve(ntasks);
        this->packed.reserve(ntasks * estimated_task_size);
    }

    template < typename Seq >
    void append(const Seq& task) noexcept (false) {
        this->sizes.push_back(task.size());
        this->packed.insert(this->packed.end(), task.begin(), task.end());
    }
};

class session {
public:
    session();
    session(session&&);
    session(const session&) = delete;
    session& operator = (session&&);
    ~session();

    void init(const char* doc, int len) noexcept (false);
    taskset plan_query(
        const char* doc,
        int len,
        int task_size)
    noexcept (false);

    std::string query_manifest(const std::string& path) noexcept (false);

private:
    class impl;
    std::unique_ptr< impl > self;
};

}

#endif //ONESEISMIC_PLAN_HPP
