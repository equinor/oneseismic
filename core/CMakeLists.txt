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

find_package(segyio REQUIRED)

if (NOT MSVC)
    # openmp is pretty broken on MSVC with only super old versions. Accept the
    # performance loss, and don't bother searching for it
    find_package(OpenMP REQUIRED)
endif ()

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

add_library(seismic-cloud src/core.cpp)
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
        ${OpenMP_CXX_FLAGS}
)
target_link_libraries(seismic-cloud
    PUBLIC
        ${OpenMP_EXE_LINKER_FLAGS}
        ${OpenMP_CXX_FLAGS}
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

add_subdirectory(core-server)

add_executable(generate src/generate.cpp)
target_link_libraries(generate seismic-cloud json clara)

add_executable(stitch src/stitch.cpp)
target_link_libraries(stitch
    seismic-cloud
    json
    clara
    mio
    ${OpenMP_EXE_LINKER_FLAGS}
    ${OpenMP_CXX_FLAGS}
)
target_compile_definitions(stitch
    PRIVATE
        # cmake >= 3.9 uses OpenMP_<lang>_FOUND, but older cmake sets
        # OPENMP_FOUND if the library is found and properly configured. We only
        # set the preprocessor directive to only include the header on
        # non-MSVC, so just set the same directive on both
        $<$<BOOL:${OPENMP_FOUND}>:HAVE_OPENMP>
        $<$<BOOL:${OpenMP_CXX_FOUND}>:HAVE_OPENMP>
)
target_compile_options(stitch
    PRIVATE
        ${OpenMP_CXX_FLAGS}
)

add_executable(verifyer src/verifyer.cpp)
target_link_libraries(verifyer json clara seismic-cloud)

add_executable(check_shatters src/check_shatters.cpp)
target_link_libraries(check_shatters
    json
    clara
    segyio::segyio
    ${OpenMP_EXE_LINKER_FLAGS}
    ${OpenMP_CXX_FLAGS}
)
target_compile_options(check_shatters
    PRIVATE
        ${OpenMP_CXX_FLAGS}
)

add_executable(shatter src/shatter.cpp)
target_link_libraries(shatter seismic-cloud json clara segyio::segyio)

add_executable(generate_surfaces src/generate_surfaces.cpp)
target_link_libraries(generate_surfaces clara)

add_subdirectory(external/catch2)

add_executable(tests tests/testsuite.cpp tests/seismic-cloud.cpp)
target_link_libraries(tests seismic-cloud catch2)

add_test(NAME test COMMAND tests)
