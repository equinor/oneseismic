cmake_minimum_required(VERSION 3.5.0)
project(oneseismic-python-extension LANGUAGES C CXX)

set(CMAKE_CXX_STANDARD 11)
set(CMAKE_CXX_VISIBILITY_PRESET "hidden")
set(CMAKE_C_VISIBILITY_PRESET   "hidden")

find_package(PythonExtensions REQUIRED)
find_package(oneseismic REQUIRED)
find_package(fmt REQUIRED)
find_package(Protobuf REQUIRED)
find_package(PkgConfig REQUIRED)
pkg_check_modules(zmq IMPORTED_TARGET REQUIRED libzmq)
pkg_check_modules(hiredis IMPORTED_TARGET REQUIRED hiredis)

add_library(geometry MODULE geometry.cpp)
target_include_directories(geometry
    PRIVATE
        ${PYBIND11_INCLUDE_DIRS}
)
python_extension_module(geometry)
target_link_libraries(geometry oneseismic::oneseismic)

if (MSVC)
    target_compile_options(geometry
        BEFORE
        PRIVATE
            /EHsc
    )
endif ()

install(TARGETS geometry LIBRARY DESTINATION oneseismic)
