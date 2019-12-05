cmake_minimum_required(VERSION 3.5)
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
    src/core.cpp
    src/url.cpp
)
add_library(seismic-cloud::seismic-cloud ALIAS seismic-cloud)
target_include_directories(seismic-cloud
    PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:include>
    PRIVATE
        include
)
target_compile_options(seismic-cloud
    PUBLIC
        ${GNUTLS_DEFINITIONS}
)
target_link_libraries(seismic-cloud
    PUBLIC
        fmt::fmt
        ${GNUTLS_LIBRARIES}
)

install(TARGETS seismic-cloud
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
)
export(
    TARGETS seismic-cloud
    FILE seismic-cloud-config.cmake
    NAMESPACE seismic-cloud::
)

add_subdirectory(external/catch2)

add_executable(tests
    tests/testsuite.cpp
    tests/seismic-cloud.cpp
    tests/request-generator.cpp
)
target_link_libraries(tests seismic-cloud catch2)

add_test(NAME test COMMAND tests)
