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
     * Get the list of fragment IDs for this process as a ';'-separated string.
     * This is intended for parsing and building real URLs.
     *
     * The substrings come back as '<resolution>/<shape>/<id>;...'
     *
     * Example use from python:
     *     urls = [
     *          f'{endpoint}/{guid}/{fragment}'
     *          for fragment in proc.fragments().split(';')
     *     ]
     */
    const std::string& fragments() const;

    /*
     * The add() function is responsible for taking a fragment (size len) and
     * extract the endpoint-specified shape from it, e.g. for /slice/0 it
     * should extract the slice along the inline.
     */
    virtual void add(int index, const char* chunk, int len) = 0;
    virtual std::string pack() = 0;

    virtual ~proc() = default;

    std::string errmsg;
    std::string packed;

protected:
    /*
     * Set the fragment shape. This is cleared by clear() and must be set for
     * every init(). It sets the prefix for fragment-ID generation.
     */
    void set_fragment_shape(const std::string&) noexcept (false);
    /*
     * Register a fragment id, for url generation. Duplicates will not be
     * removed, this is effectively an accumulating ';'.join([prefix + id]...)
     *
     * This will be cleared by clear(), which must be called before process
     * handles are re-used.
     */
    void add_fragment(const std::string& id) noexcept (false);
    void clear() noexcept (true);

private:
    std::string prefix;
    std::string frags;
};

class slice : public proc {
public:
    void init(const char* msg, int len) override;
    virtual void add(int, const char* chunk, int len) override;
    std::string pack() override;

private:
    one::slice_fetch input;
    one::slice_tiles output;

    one::dimension< 3 > dim = one::dimension< 3 >(0);
    int idx;
    one::slice_layout layout;
    one::gvt< 2 > gvt;
};

}

#endif //ONESEISMIC_PROC_HPP
