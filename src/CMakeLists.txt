cmake_minimum_required(VERSION 3.5)
project(js LANGUAGES CXX)

include(CheckIncludeFile)
include(CTest)
include(GNUInstallDirs)
include(TestBigEndian)

add_library(json INTERFACE)
target_include_directories(json INTERFACE external/nlohmann)

add_library(clara INTERFACE)
target_include_directories(clara INTERFACE external/clara)

find_package(segyio REQUIRED)
find_package(OpenMP REQUIRED)

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

add_executable(generate generate.cpp)
target_link_libraries(generate json clara)

add_executable(stitch stitch.cpp)
target_link_libraries(stitch json clara mio)
target_include_directories(stitch
    PRIVATE include
)

add_executable(verifyer verifyer.cpp)
target_link_libraries(verifyer json clara)

add_executable(check_shatters check_shatters.cpp)
target_link_libraries(check_shatters json clara segyio
                      ${OpenMP_EXE_LINKER_FLAGS} ${OpenMP_CXX_FLAGS})
target_compile_options(check_shatters PRIVATE ${OpenMP_CXX_FLAGS})

add_executable(shatter shatter)
target_link_libraries(shatter json clara segyio)

add_executable(generate_surfaces generate_surfaces.cpp)
target_link_libraries(generate_surfaces clara)
