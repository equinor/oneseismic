#include <msgpack.hpp>

#include <oneseismic/decoder.hpp>
#include <oneseismic/messages.hpp>

#include <pybind11/pybind11.h>
#include <pybind11/stl.h>
namespace py = pybind11;
using namespace py::literals;

namespace {

void buffer(one::decoder& self, py::buffer b) {
    auto info = b.request();
    self.buffer(
        reinterpret_cast< const char* >(info.ptr),
        info.size * info.itemsize
    );
}

one::decoder::status buffer_and_process(
    one::decoder& self,
    py::buffer b)
{
    auto info = b.request();
    return self.buffer_and_process(
        reinterpret_cast< const char* >(info.ptr),
        info.size * info.itemsize
    );
}

void register_writer(
    one::decoder& self,
    const std::string& key,
    py::buffer b)
{
    auto info = b.request();
    self.register_writer(key, info.ptr);
}

}

PYBIND11_MODULE(decoder, m) {
    py::class_<one::process_header>(m, "header")
        .def_readonly("attrs",      &one::process_header::attributes)
        .def_readonly("ndims",      &one::process_header::ndims)
        .def_readonly("index",      &one::process_header::index)
        .def_readonly("function",   &one::process_header::function)
    ;

    py::enum_<one::functionid>(m, "functionid")
        .value("slice",     one::functionid::slice)
        .value("curtain",   one::functionid::curtain)
        .export_values()
    ;

    py::class_<one::decoder> decoder(m, "decoder");
    decoder
        .def(py::init<>())
        .def("reset",               &one::decoder::reset)
        .def("process",             &one::decoder::process)
        .def("buffer",              buffer)
        .def("buffer_and_process",  buffer_and_process)
        .def("header",              &one::decoder::header,
                                        py::return_value_policy::copy)
        .def("register_writer",     register_writer)
    ;

    using status = one::decoder::status;
    py::enum_<status>(decoder, "status")
        .value("paused", status::paused)
        .value("done",   status::done)
        .export_values()
    ;

}
