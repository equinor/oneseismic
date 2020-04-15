cmake_minimum_required(VERSION 3.9)
project(oneseismic LANGUAGES CXX)

include(CheckIncludeFile)
include(CTest)
include(GNUInstallDirs)
include(TestBigEndian)

option(BUILD_PYTHON "Build Python library" ON)

add_library(json INTERFACE)
target_include_directories(json INTERFACE external/nlohmann)

add_library(clara INTERFACE)
target_include_directories(clara INTERFACE external/clara)

find_package(fmt REQUIRED)
find_package(GnuTLS REQUIRED)
find_package(CURL REQUIRED)
find_package(Threads REQUIRED)
find_package(Protobuf REQUIRED)

find_package(PkgConfig REQUIRED)
pkg_check_modules(microhttpd IMPORTED_TARGET libmicrohttpd)
pkg_check_modules(zmq IMPORTED_TARGET REQUIRED libzmq)

protobuf_generate_cpp(
    PROTO_SRCS
    PROTO_HDRS
    ${CMAKE_CURRENT_SOURCE_DIR}/../protos/core.proto
)

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

add_library(oneseismic
    src/base64.cpp
    src/geometry.cpp
    src/transfer.cpp
    src/azure.cpp
    src/worker.cpp
    src/manifest.cpp
    ${PROTO_SRCS}
    ${PROTO_HDRS}
)
add_library(oneseismic::oneseismic ALIAS oneseismic)
target_include_directories(oneseismic
    PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:include>
        ${CURL_INCLUDE_DIRS}
    PRIVATE
        include
        ${GNUTLS_INCLUDE_DIR}
        ${CMAKE_CURRENT_BINARY_DIR}
        external/nlohmann
)
target_compile_options(oneseismic
    PUBLIC
        ${GNUTLS_DEFINITIONS}
)
target_link_libraries(oneseismic
    PUBLIC
        fmt::fmt
        ${GNUTLS_LIBRARIES}
        ${CURL_LIBRARIES}
        PkgConfig::zmq
        protobuf::libprotobuf
)

add_executable(oneseismic-manifest
    src/server-manifest.cpp
)

target_link_libraries(oneseismic-manifest
    oneseismic::oneseismic
    fmt::fmt
    PkgConfig::zmq
    clara
)

install(
    TARGETS
        oneseismic
        oneseismic-manifest
    EXPORT
        oneseismic-export
    ARCHIVE DESTINATION
        ${CMAKE_INSTALL_LIBDIR}
    LIBRARY DESTINATION
        ${CMAKE_INSTALL_LIBDIR}
    RUNTIME DESTINATION
        ${CMAKE_INSTALL_BINDIR}
)
install(DIRECTORY include/ DESTINATION include)
install(
    EXPORT
        oneseismic-export
    DESTINATION
        ${CMAKE_INSTALL_DATAROOTDIR}/oneseismic/cmake
    FILE
        oneseismic-config.cmake
    NAMESPACE
        oneseismic::
)
export(
    TARGETS
        oneseismic
    FILE
        oneseismic-config.cmake
    NAMESPACE
        oneseismic::
)
set(ONESEISMIC_LIB_CMAKECONFIG_DIR ${CMAKE_CURRENT_BINARY_DIR} CACHE PATH "")

add_subdirectory(external/catch2)

add_executable(tests
    tests/testsuite.cpp
    tests/geometry.cpp
    tests/azure-transfer-config.cpp
)
target_link_libraries(tests
    PRIVATE
        catch2
        oneseismic::oneseismic
        fmt::fmt
        Threads::Threads
)
target_include_directories(tests
    PRIVATE
        ${CMAKE_CURRENT_BINARY_DIR}
)

if (TARGET PkgConfig::microhttpd)
    target_sources(tests PRIVATE tests/transfer-http.cpp)
    target_sources(tests PRIVATE tests/manifest-task.cpp)
    target_sources(tests PRIVATE tests/fragment-task.cpp)
    target_link_libraries(tests PRIVATE PkgConfig::microhttpd)
else ()
    message(WARNING
        "microhttpd not found - not building transfer tests "
        "https://www.gnu.org/software/libmicrohttpd/"
    )
endif()

add_test(NAME test COMMAND tests)

add_subdirectory(python)
