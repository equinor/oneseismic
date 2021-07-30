#ifndef ONESEISMIC_DECODER_HPP
#define ONESEISMIC_DECODER_HPP

#include <map>
#include <string>

#include <msgpack.hpp>

#include <oneseismic/messages.hpp>

namespace one {

/*
 * Class to parse the response message format, and extract into the intended
 * "target" structure. The main feature is the streaming support; it is built
 * for working on not-yet-complete messages, and wired directly to the download
 * stream.
 */
class decoder {
public:
    enum class status {
        paused,
        done,
    };

    /*
     * Buffer up some data, but don't process it yet. Calling this over
     * buffer_and_process can be useful if you want to free up your own
     * buffers, but not yet start processing (e.g. when you know that it's just
     * half an object anyway), or if you want to delay processing until all
     * data is read.
     */
    void buffer(const char* input, std::size_t len);

    /*
     * Process as many message parts as possible.
     *
     * This function does nothing if:
     * * no data is buffered
     * * no complete message is buffered
     * * the message completely processed
     *
     * The function will *always* pause once it parses the header, and in order
     * to parse a full message it must be called *at least* twice. It stops
     * because writers for the message must be registered, and registering them
     * requires data that is in the header.
     *
     * Process will throw an exception if the internal state machine breaks, or
     * the message is corrupted and cannot be processed further, which means
     * process() can be called repeatedly until it returns done.
     *
     * Example
     * -------
     * while (true) {
     *     stream.read();
     *     a.buffer(stream.bytes, stream.len);
     *     a.process();
     *
     *     if (a.header()) {
     *         for (auto& attr : a.header()->attributes) {
     *             auto* wr = outputs.alloc(attr);
     *             a.register_writer(attr, wr);
     *         }
     *         break;
     *     }
     * }
     *
     * while (true) {
     *     stream.read()
     *     a.buffer(stream.bytes, stream.len);
     *     status = a.process();
     *     if (status == done)
     *         break;
     * }
     */
    status process();

    /*
     * Call buffer and process in sequence. This is usually the function you
     * want to use. This function can be called until it returns status::done.
     */
    status buffer_and_process(const char* input, std::size_t len);

    /*
     * Get the process header for the current request, if available. This
     * function returns nullptr if the header has not been processed yet, in
     * which case callers must buffer and process more data. The pointer is
     * valid until the next call to reset(), or until this object is destroyed.
     */
    const process_header* header() const noexcept (true);

    /*
     * If you are re-using the decoder between messages, call reset() before
     * starting a new message.
     */
    void reset();

    /*
     * Register a writer/output arena for some attribute. *data should be a
     * buffer large enough for the output.
     *
     * Examples
     * --------
     * Pre-allocate a numpy array and register it as the writer. This is for a
     * slice, and the allocation must be aware of the shape of the output based
     * on the shape+index.
     *
     * head = a.header()
     * arrays = {}
     * for attr in head.attributes:
     *     shape = head.index[:head.ndims]
     *     npa = np.zeros(shape = shape, dtype = 'f4')
     *     a.register_writer(attr, npa)
     *     arrays[attr] = npa
     * # process-all
     */
    void register_writer(const std::string& attr, void* data);

private:
    msgpack::v2::unpacker unp;
    msgpack::v2::object_handle objhandle;

    /*
     * Processing functions
     *
     * These process and extract the parsed message components into the right
     * output structure. If you are implementing response message parsing
     * outside of the decoder, then you  can extract the algorithms from the
     * processing functions. They are kept private until exposing them is
     * actually necessary and useful, in order to make it easier to evolve the
     * messages, and keep the public API smaller.
     */
    void extract(const msgpack::v2::object&) noexcept (false);
    void slice  (const msgpack::v2::object&) noexcept (false);
    void curtain(const msgpack::v2::object&) noexcept (false);

    /*
     * Look for the writer for some attribute ('data', 'cdpx' etc.). If no such
     * writer is registered, this returns nullptr indicating that the block can
     * be skipped.
     *
     * Not registering a writer is not an error, it is choosing to ignore parts
     * of the response.
     */
    char* get_writer_for(const std::string& attr) noexcept (true);

    enum class state {
        envelope,
        header,
        nbundles,
        bundles,
        done,
    };

    state phase = state::envelope;
    int nbundles = 0;
    process_header head;
    std::map< std::string, void* > writers;
};

}

#endif //ONESEISMIC_DECODER_HPP
