#include <vector>
#include <string>
#include <map>
#include <exception>

#include <msgpack.hpp>

#include <oneseismic/decoder.hpp>
#include <oneseismic/messages.hpp>

namespace msgpack {
namespace v2 {
namespace adaptor {

template <>
struct convert< one::functionid > {
    const v2::object& operator () (const v2::object& o, one::functionid& id)
    const noexcept (false) {
        const auto v = o.as< int >();
        switch (static_cast< one::functionid >(v)) {
            case one::functionid::slice:
            case one::functionid::curtain:
                break;

            default: {
                const auto msg = "Invalid function; was " + std::to_string(v);
                throw one::bad_value(msg);
            }
        }

        id = static_cast< one::functionid >(v);
        return o;
    }
};

template <>
struct convert< one::process_header > {
    const v2::object& operator () (const v2::object& o, one::process_header& head)
    const noexcept (false) {
        if (o.type != type::MAP)
            throw type_error();

        const auto& kvs = o.via.map;
        std::string key;
        for (int i = 0; i < kvs.size; ++i) {
            const auto& kv = kvs.ptr[i];
            kv.key >> key;
                 if (key == "pid")        kv.val >> head.pid;
            else if (key == "function")   kv.val >> head.function;
            else if (key == "nbundles")   kv.val >> head.nbundles;
            else if (key == "ndims")      kv.val >> head.ndims;
            else if (key == "labels")     kv.val >> head.labels;
            else if (key == "index")      kv.val >> head.index;
            else if (key == "shapes")     kv.val >> head.shapes;
            else if (key == "attributes") kv.val >> head.attributes;
            else {
                throw one::bad_message("Unknown key '" + key + "' in header");
            }
        }
        return o;
    }
};

}}}

namespace one {

namespace {

auto asarray(const msgpack::v2::object& obj)
noexcept (false)
-> decltype(obj.via.array)
{
    if (obj.type == msgpack::v2::type::ARRAY)
        return obj.via.array;

    throw msgpack::v2::type_error();
}

auto astuple(const msgpack::v2::object& obj, int len)
noexcept (false)
-> decltype(obj.via.array.ptr)
{
    const auto a = asarray(obj);
    if (a.size != len) {
        const auto msg = "expected " + std::to_string(len)
                       + " slots, was "
                       + std::to_string(a.size)
                       ;
        throw bad_message(msg);
    }

    return a.ptr;
}

auto asbinarray(const msgpack::v2::object& obj)
noexcept (false)
-> decltype(obj.via.bin)
{
    if (obj.type == msgpack::v2::type::BIN)
        return obj.via.bin;

    throw msgpack::v2::type_error();
}

std::uint32_t net32(const char* net) noexcept (true) {
    std::uint8_t data[sizeof(std::uint32_t)] = {};
    std::memcpy(&data, net, sizeof(data));
    return ((std::uint32_t) data[3] << 0)
         | ((std::uint32_t) data[2] << 8)
         | ((std::uint32_t) data[1] << 16)
         | ((std::uint32_t) data[0] << 24)
    ;
}

std::uint16_t net16(const char* net) noexcept (true) {
    std::uint8_t data[sizeof(std::uint16_t)] = {};
    std::memcpy(&data, net, sizeof(data));
    return ((std::uint16_t) data[0] << 8)
         | ((std::uint16_t) data[1] << 0)
    ;
}

struct insufficient_bytes : public std::exception {};
int parse_array_len(const char* input, std::size_t size, int& len)
noexcept (false) {
    if (size < 1) throw insufficient_bytes();
    const auto tag = std::uint8_t(input[0]);

    if ((tag & 0xF0) == 0x90) {
        len = tag & 0x0F;
        return 1;
    }
    else if (tag == 0xDC) {
        if (size < 3) throw insufficient_bytes();
        len = net16(input + 1);
        return sizeof(std::uint16_t) + 1;
    }
    else if (tag == 0xDD) {
        if (size < 5) throw insufficient_bytes();
        len = net32(input + 1);
        return sizeof(std::uint32_t) + 1;
    } else {
        const auto msg = "expected array tag; was " + std::to_string(int(tag));
        throw bad_message(msg);
    }
}

int unpack_array_len(msgpack::v2::unpacker& unp) {
    int len;
    const auto* unparsed = unp.nonparsed_buffer();
    const auto remaining = unp.nonparsed_size();
    const auto nread = parse_array_len(unparsed, remaining, len);
    unp.skip_nonparsed_buffer(nread);
    return len;
}

void read_check_envelope(msgpack::v2::unpacker& unp) noexcept (false) {
    const auto len = unpack_array_len(unp);
    if (len == 2)
        return;

    const auto msg = "bad envelope; expected array(2), was ";
    throw bad_message(msg + std::to_string(len));
}

}

void decoder::reset() {
    // TODO: soft reset where phase, nbundles are reset, but the buffer and
    // optionally writers are not?
    this->unp.remove_nonparsed_buffer();
    this->phase = state::envelope;
    this->nbundles = 0;
    this->writers.clear();
}

const process_header* decoder::header() const noexcept (true) {
    switch (this->phase) {
        case state::envelope:
        case state::header:
            return nullptr;

        default:
            return &this->head;
    }
}

void decoder::register_writer(const std::string& attr, void* data) {
    this->writers[attr] = data;
}

void decoder::buffer(const char* input, std::size_t size) {
    if (size > 0) {
        this->unp.reserve_buffer(size);
        std::copy_n(input, size, this->unp.buffer());
        this->unp.buffer_consumed(size);
    }
}

/*
 * process() implements a fairly simple state machine that parses and processes
 * the response message, and is aware that the message may be incomplete. This
 * is a rough overview of the message structure:
 *
 * [0] envelope
 * [1] header
 * [2] nbundles
 * [3] bundles*
 *
 * The message format is built on top of msgpack, and all messages are complete
 * and valid msgpack messages. The goal for this function is to *stream* the
 * data, which means extracting the individual values as they are parsed,
 * rather than parsing into an intermediary msgpack structure, and then
 * performing extraction on the result of the parsing.
 *
 * The message is packaged as an array of 3 elements:
 *  - The [1] header
 *  - The [2] nbundles
 *  - The [3] bundles
 *
 * This makes the envelope simply the array-type + length (2). The header [1]
 * is the process_header in messages.hpp. The nbundles [2] tells the
 * parse-and-extract how many bundles the payload is made up of (which should
 * correspond to the nbundles field in the process header), and is just the
 * array type + length. Finally, the bundles is the n blobs that make up the
 * response body.
 *
 * The process() function will *always* give back control after parsing the
 * header, even if a full message is buffered. Since extraction is done by
 * looking up writers which must be registered on a per-message basis, it is
 * vital that no payload is touched before the caller has had the oportunity to
 * register writers, but the caller cannot know what writers to register
 * without reading the header.
 */
decoder::status decoder::process() {
    switch (this->phase) {
        case state::envelope: {
            try {
                read_check_envelope(this->unp);
            } catch (insufficient_bytes&) {
                return status::paused;
            }
            this->phase = state::header;
        }
        [[fallthrough]];

        case state::header:
            if (!this->unp.next(this->objhandle))
                return status::paused;

            this->objhandle.get() >> this->head;
            this->phase = state::nbundles;
            return status::paused;

        case state::nbundles: {
            try {
                this->nbundles = unpack_array_len(this->unp);
            } catch (insufficient_bytes&) {
                return status::paused;
            }
            if (nbundles != this->head.nbundles) {
                const auto msg = "nbundles inconsistent; header.nbundles = "
                    + std::to_string(this->head.nbundles)
                    + ", envelope.nbundles = "
                    + std::to_string(nbundles)
                ;
                throw bad_message(msg);
            }
            this->phase = state::bundles;
        }
        [[fallthrough]];

        case state::bundles:
            while (this->nbundles > 0) {
                if (!this->unp.next(this->objhandle))
                    return status::paused;

                this->extract(this->objhandle.get());
                this->nbundles -= 1;
            }
            this->phase = state::done;
            return status::done;

        case state::done:
            return status::done;

        default:
            throw std::logic_error("void phase; should be unreachable");
    }
}

decoder::status decoder::buffer_and_process(
    const char* input,
    std::size_t size)
{
    this->buffer(input, size);
    return this->process();
}

void decoder::extract(const msgpack::v2::object& obj) {
    switch (this->head.function) {
        case functionid::slice:
            this->slice(obj);
            return;

        case functionid::curtain:
            this->curtain(obj);
            return;

        default:
            break;
    }

    const auto msg = "void function; message poorly sanitized";
    throw std::logic_error(msg);
}

void decoder::slice(const msgpack::v2::object& obj)
noexcept (false) {
    const auto root = astuple(obj, 2);

    const auto attribute = root[0].as< std::string >();
    auto* dst = this->get_writer_for(attribute);
    if (!dst)
        return;

    const auto atiles = asarray(root[1]);
    const auto ntiles = atiles.size;
    const auto tiles  = atiles.ptr;

    for (int i = 0; i < ntiles; ++i) {
        const auto slots = astuple(tiles[i], 6);

        const auto iterations   = slots[0].as< int >();
        const auto chunk_size   = slots[1].as< int >();
        const auto initial_skip = slots[2].as< int >();
        const auto superstride  = slots[3].as< int >();
        const auto substride    = slots[4].as< int >();
        const auto v            = asbinarray(slots[5]);

        const auto* src = v.ptr;
        for (int i = 0; i < iterations; ++i) {
            std::memcpy(
                dst + sizeof(float) * (i * superstride + initial_skip),
                src + sizeof(float) * (i * substride),
                sizeof(float) * chunk_size
            );
        }
    }
}

void decoder::curtain(const msgpack::v2::object& obj)
noexcept (false) {
    const auto& slots = astuple(obj, 5);

    const auto attribute = slots[0].as< std::string >();
    auto* dst = this->get_writer_for(attribute);
    if (!dst)
        return;

    // TODO: check message integrity (array sizes)
    // TODO: cache vectors
    const auto size  = slots[1].as< int >();
    const auto major = slots[2].as< std::vector< int > >();
    const auto minor = slots[3].as< std::vector< int > >();
    const auto v     = asbinarray(slots[4]);

    const auto* src = v.ptr;
    const auto zlen = this->head.index[2];
    for (int n = 0; n < size; ++n) {
        const auto ifst = major[n*2 + 0];
        const auto ilst = major[n*2 + 1];
        const auto zfst = minor[n*2 + 0];
        const auto zlst = minor[n*2 + 1];

        const auto elemsize = sizeof(float);
        const auto chunksize = elemsize * (zlst - zfst);
        for (int i = ifst; i < ilst; ++i) {
            const auto chunk = i - ifst;
            std::memcpy(
                dst + elemsize  * (i*zlen + zfst),
                src + chunksize * chunk,
                chunksize
            );
        }
        src += chunksize * (ilst - ifst);
    }
}

char* decoder::get_writer_for(const std::string& attr) noexcept (true) {
    auto itr = this->writers.find(attr);
    if (itr == this->writers.end())
        return nullptr;
    return reinterpret_cast< char* >(itr->second);
}

}
