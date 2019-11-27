cmake_minimum_required(VERSION 3.5)
project(fetch-curl LANGUAGES C CXX)

include(GNUInstallDirs)

find_package(CURL REQUIRED)

set(CMAKE_CXX_STANDARD 14)

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

add_executable(fetch-all src/fetch.cpp)
target_link_libraries(fetch-all ${CURL_LIBRARIES})
target_include_directories(fetch-all SYSTEM PRIVATE ${CURL_INCLUDE_DIRS})
