cmake_minimum_required(VERSION 3.12)
project(oneseismic LANGUAGES CXX)

include(CheckIncludeFile)
include(CTest)
include(GNUInstallDirs)
include(TestBigEndian)

option(BUILD_CORE      "Build the core library"      ON)
option(BUILD_DECODER   "Build the decoder library"   ON)

if (BUILD_TESTING)
    add_subdirectory(external/catch2)
endif ()

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

add_library(oneseismic)
target_compile_features(oneseismic PUBLIC cxx_std_20)
add_library(oneseismic::oneseismic ALIAS oneseismic)
target_include_directories(oneseismic
    PUBLIC
        $<BUILD_INTERFACE:${CMAKE_CURRENT_SOURCE_DIR}/include>
        $<INSTALL_INTERFACE:include>
)

# Some users will not want the full library (e.g. language bindings for the
# decoder), so the sources and dependencies are guarded behind feature
# toggles.
if (BUILD_CORE)
    add_library(json INTERFACE)
    target_include_directories(json INTERFACE external/nlohmann)
    find_package(fmt REQUIRED)

    target_sources(oneseismic
        PRIVATE
            src/base64.cpp
            src/messages.cpp
            src/plan.cpp
            src/process.cpp
    )

    target_include_directories(oneseismic
        PRIVATE
            include
            external/nlohmann
    )
    target_link_libraries(oneseismic
        PUBLIC
            fmt::fmt
    )

    if (BUILD_TESTING)
        add_executable(core-tests
            tests/testsuite.cpp
            tests/geometry.cpp
            tests/messages.cpp
            tests/process.cpp
        )
        target_link_libraries(core-tests
            PRIVATE
                catch2
                oneseismic::oneseismic
                fmt::fmt
                json
        )
        add_test(NAME unit-tests COMMAND core-tests)
    endif ()
endif ()

if (BUILD_DECODER)
    target_sources(oneseismic PRIVATE src/decoder.cpp)
endif ()

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
