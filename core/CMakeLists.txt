cmake_minimum_required(VERSION 3.9)
project(oneseismic LANGUAGES CXX)

include(CheckIncludeFile)
include(CTest)
include(GNUInstallDirs)
include(TestBigEndian)

option(BUILD_PYTHON  "Build Python library"                ON)

add_library(json INTERFACE)
target_include_directories(json INTERFACE external/nlohmann)

find_package(fmt        REQUIRED)
find_package(spdlog     REQUIRED)

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
    src/messages.cpp
    src/plan.cpp
)
add_library(oneseismic::oneseismic ALIAS oneseismic)
target_include_directories(oneseismic
    PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:include>
        ${CURL_INCLUDE_DIRS}
    PRIVATE
        include
        external/nlohmann
)
target_link_libraries(oneseismic
    PUBLIC
        fmt::fmt
)

install(
    TARGETS
        oneseismic
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

if (NOT BUILD_TESTING)
    return()
endif ()

add_subdirectory(external/catch2)

add_executable(tests
    tests/testsuite.cpp
    tests/geometry.cpp
    tests/messages.cpp
)
target_link_libraries(tests
    PRIVATE
        catch2
        oneseismic::oneseismic
        fmt::fmt
        json
)
add_test(NAME unit-tests COMMAND tests)
