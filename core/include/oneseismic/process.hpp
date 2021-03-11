#ifndef ONESEISMIC_PROC_HPP
#define ONESEISMIC_PROC_HPP

#include <string>
#include <vector>

#include <oneseismic/messages.hpp>

namespace one {

class proc {
public:
    virtual void init(const char* msg, int len) = 0;

    /*
     * A vector-of-fragment IDs for this process.
     */
    virtual std::vector< std::string > fragments() const = 0;

    /*
     * The add() function is responsible for taking a fragment (size len) and
     * extract the endpoint-specified shape from it, e.g. for /slice/0 it
     * should extract the slice along the inline.
     */
    virtual void add(int index, const char* chunk, int len) = 0;
    virtual std::string pack() = 0;

    virtual ~proc() = default;

    std::string errmsg;
    std::string frags;
    std::string packed;
};

class slice : public proc {
public:
    void init(const char* msg, int len) override;
    std::vector< std::string > fragments() const override;
    virtual void add(int, const char* chunk, int len) override;
    std::string pack() override;

private:
    one::slice_fetch req;
    one::slice_tiles tiles;

    one::dimension< 3 > dim = one::dimension< 3 >(0);
    int idx;
    one::slice_layout layout;
    one::gvt< 2 > gvt;
};

}

#endif //ONESEISMIC_PROC_HPP
