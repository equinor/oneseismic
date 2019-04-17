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

add_executable(generate src/generate.cpp)
target_link_libraries(generate json clara)

add_executable(stitch src/stitch.cpp)
target_link_libraries(stitch
    PRIVATE json
            clara
            mio
            ${OpenMP_EXE_LINKER_FLAGS}
            ${OpenMP_CXX_FLAGS}
)
target_include_directories(stitch
    PRIVATE include
)
target_compile_definitions(stitch
    PRIVATE _GLIBCXX_PARALLEL
)
target_compile_options(stitch
    PUBLIC ${OpenMP_CXX_FLAGS}

)

add_executable(verifyer src/verifyer.cpp)
target_link_libraries(verifyer json clara)

add_executable(check_shatters src/check_shatters.cpp)
target_link_libraries(check_shatters json clara segyio
                      ${OpenMP_EXE_LINKER_FLAGS} ${OpenMP_CXX_FLAGS})
target_compile_options(check_shatters PRIVATE ${OpenMP_CXX_FLAGS})

add_executable(shatter src/shatter.cpp)
target_link_libraries(shatter json clara segyio)

add_executable(generate_surfaces src/generate_surfaces.cpp)
target_link_libraries(generate_surfaces clara)

add_subdirectory(external/catch2)

add_executable(tests tests/testsuite.cpp)
target_include_directories(tests PRIVATE src)
target_link_libraries(tests catch2)

add_test(NAME test COMMAND tests)
