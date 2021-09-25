#include <string>
#include <vector>

#include <oneseismic/decoder.hpp>

#include <emscripten/bind.h>

using namespace emscripten;

void register_writer(
    one::decoder& self,
    const std::string& key,
    std::uintptr_t ptr)
{
    self.register_writer(key, reinterpret_cast< void* >(ptr));
}

void buffer(one::decoder& self, const std::string& buffer) {
    return self.buffer(buffer.data(), buffer.size());
}

one::decoder::status
buffer_and_process(one::decoder& self, const std::string& buffer) {
    return self.buffer_and_process(buffer.data(), buffer.size());
}

bool header_ready(const one::decoder& self) {
    return self.header();
}

one::process_header header_get(const one::decoder& self) {
    return *self.header();
}

EMSCRIPTEN_BINDINGS(decoder) {
    register_vector< int >("VectorInt");
    register_vector< std::string >("VectorString");

    class_< one::process_header >("process_header")
        .property("attrs",    &one::process_header::attributes)
        .property("ndims",    &one::process_header::ndims)
        .property("index",    &one::process_header::index)
        .property("function", &one::process_header::function)
        .property("shapes",   &one::process_header::shapes)
        .property("labels",   &one::process_header::labels)
    ;

    class_< one::decoder >("decoder")
        .constructor<>()
        .function("reset",               &one::decoder::reset)
        .function("process",             &one::decoder::process)
        .function("buffer",              buffer)
        .function("buffer_and_process",  buffer_and_process)
        .function("header_ready",        header_ready)
        .function("header_get",          header_get)
        .function("register_writer",     register_writer)
    ;

    using status = one::decoder::status;
    enum_<status>("status")
        .value("paused", status::paused)
        .value("done",   status::done)
    ;

    enum_<one::functionid>("functionid")
        .value("slice",   one::functionid::slice)
        .value("curtain", one::functionid::curtain)
    ;
}
