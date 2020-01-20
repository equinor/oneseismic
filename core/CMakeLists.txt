cmake_minimum_required(VERSION 3.9)
project(js LANGUAGES CXX)

include(CheckIncludeFile)
include(CTest)
include(GNUInstallDirs)
include(TestBigEndian)

# The libraries will be linked against from python extension module. This requires
# position independent code.
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

add_library(json INTERFACE)
target_include_directories(json INTERFACE external/nlohmann)

add_library(clara INTERFACE)
target_include_directories(clara INTERFACE external/clara)

find_package(fmt REQUIRED)
find_package(GnuTLS REQUIRED)
find_package(CURL REQUIRED)
find_package(Threads REQUIRED)
find_package(Protobuf REQUIRED)

# gRPC sometimes provide cmake config files, and sometimes provides pkg-config
# files, depending on how it was built. On debian/ubuntu, the grpc-dev package
# only includes the pkg-config. When the cmake config does not work out, fall
# back to pkg-config. The GLOBAL flag, requried for aliasing PkgConfig::grpc++
# to gRPC::grpc++, is not availble until CMake 3.13 [1], so until that's the
# minimum required cmake version, emulate it by putting the link-target in a
# variable.
#
# [1] https://cmake.org/cmake/help/v3.13/module/FindPkgConfig.html
find_package(gRPC QUIET)
if (gRPC_FOUND)
    set(ONESEISMIC_GRPC_TARGET gRPC::grpc++)
elseif (NOT gRPC_FOUND)
    message(
        STATUS
        "gRPC not found with find_package, falling back to pkg-config"
    )
    find_package(PkgConfig REQUIRED)
    pkg_check_modules(grpc++ REQUIRED IMPORTED_TARGET grpc++)
    # add_library(gRPC::grpc++ ALIAS PkgConfig::grpc++)
    set(ONESEISMIC_GRPC_TARGET PkgConfig::grpc++)
endif ()

# cmake ships a protobuf module, but not one that handles the protobuf-grpc
# plugin for compiling .proto files into grpc, so set it up manually
find_program(
    PROTOC_GRPC_CPP_EXECUTABLE
    grpc_cpp_plugin
)
if (NOT PROTOC_GRPC_CPP_EXECUTABLE)
    message(FATAL_ERROR "Could not find grpc_cpp_plugin (protoc-grpc)")
endif ()

# The custom command is pretty much the what is run by protobuf_generate_cpp
protobuf_generate_cpp(
    PROTO_SRCS
    PROTO_HDRS
    ${CMAKE_CURRENT_SOURCE_DIR}/../protos/core.proto
)
add_custom_command(
    OUTPUT
        core.grpc.pb.h
        core.grpc.pb.cc
    COMMAND
        ${PROTOBUF_PROTOC_EXECUTABLE}
            --grpc_out=${CMAKE_CURRENT_BINARY_DIR}
            --plugin=protoc-gen-grpc=${PROTOC_GRPC_CPP_EXECUTABLE}
            -I ${CMAKE_CURRENT_SOURCE_DIR}/../protos
            ${CMAKE_CURRENT_SOURCE_DIR}/../protos/core.proto
    DEPENDS
        ${CMAKE_CURRENT_SOURCE_DIR}/../protos/core.proto
)

add_subdirectory(external/mio)

if (NOT MSVC)
    # assuming gcc-style options
    # add warnings in debug mode
    list(APPEND warnings-c++ -Wall
                             -Wextra
                             -pedantic
                             -Wformat-nonliteral
                             -Wcast-align
                             -Wpointer-arith
                             -Wmissing-declarations
                             -Wcast-qual
                             -Wwrite-strings
                             -Wchar-subscripts
                             -Wredundant-decls
    )
endif ()

set(CMAKE_CXX_STANDARD 14)

add_library(seismic-cloud
    src/base64.cpp
    src/geometry.cpp
    src/url.cpp
    src/fetch.cpp
)
add_library(seismic-cloud::seismic-cloud ALIAS seismic-cloud)
target_include_directories(seismic-cloud
    PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:include>
        ${CURL_INCLUDE_DIRS}
    PRIVATE
        include
        ${GNUTLS_INCLUDE_DIR}
)
target_compile_options(seismic-cloud
    PUBLIC
        ${GNUTLS_DEFINITIONS}
)
target_link_libraries(seismic-cloud
    PUBLIC
        fmt::fmt
        ${GNUTLS_LIBRARIES}
        ${CURL_LIBRARIES}
)

add_executable(one-server
    src/server.cpp
    ${PROTO_SRCS}
    ${PROTO_HDRS}
    ${CMAKE_CURRENT_BINARY_DIR}/core.grpc.pb.h
    ${CMAKE_CURRENT_BINARY_DIR}/core.grpc.pb.cc
)
target_include_directories(one-server
    PRIVATE
        ${CMAKE_CURRENT_BINARY_DIR}
)
target_link_libraries(one-server
    PRIVATE
        clara
        ${ONESEISMIC_GRPC_TARGET}
        protobuf::libprotobuf
        seismic-cloud::seismic-cloud
        # explicitly link to Threads - the libprotobuf target does not add the
        # dependency to its target on threads until cmake 3.13
        Threads::Threads
)

install(TARGETS seismic-cloud one-server
        EXPORT seismic-cloud-export
        ARCHIVE DESTINATION ${CMAKE_INSTALL_LIBDIR}
        LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
        RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
)
install(DIRECTORY include/ DESTINATION include)
install(
    EXPORT seismic-cloud-export
    DESTINATION ${CMAKE_INSTALL_DATAROOTDIR}/seismic-cloud/cmake
    FILE seismic-cloud-config.cmake
    NAMESPACE seismic-cloud::
)
export(
    TARGETS seismic-cloud
    FILE seismic-cloud-config.cmake
    NAMESPACE seismic-cloud::
)

add_subdirectory(external/catch2)

add_executable(tests
    tests/testsuite.cpp
    tests/geometry.cpp
    tests/request-generator.cpp
)
target_link_libraries(tests seismic-cloud catch2)

add_test(NAME test COMMAND tests)
