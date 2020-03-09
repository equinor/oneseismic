#include <array>

#include <pybind11/pybind11.h>
#include <pybind11/stl.h>

#include <oneseismic/geometry.hpp>

namespace py = pybind11;

PYBIND11_MODULE(geometry, m) {

    py::class_< one::CS< 3 > >(m, "CS3")
      .def(py::init< const std::array< std::size_t, 3 >& >())
      .def(py::init< std::size_t, std::size_t, std::size_t >())
      .def("__getitem__", [](const one::CS< 3 >& cs, std::size_t i) {
        if (i > 2)
          throw py::index_error();
        return cs[i];
      }
      )
    ;
 py::class_< one::FS< 3 > >(m, "FS3")
     .def(py::init< const std::array< std::size_t, 3 >& >())
     .def(py::init< std::size_t, std::size_t, std::size_t >())
     .def(py::init< const one::FS< 3 >& >())
     .def("__getitem__", [](const one::FS< 3 >& fs, std::size_t i) {
       if (i > 2)
         throw py::index_error();
       return fs[i];
     }
     )
   ;

   py::class_< one::gvt< 3 > >(m, "GVT3")
     .def(py::init< const one::CS< 3 >&, const one::FS< 3 >& >())
     .def("fragment_shape", [](const one::gvt< 3 >& gvt){
        return gvt.fragment_shape();
      }
     )
     .def("fragment_count", [](const one::gvt< 3 >& gvt, std::size_t d) {
        return gvt.fragment_count(one::dimension< 3 >(d));
     }
    )
   ;
}
