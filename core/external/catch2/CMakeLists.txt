cmake_minimum_required(VERSION 3.2)
project(catch2 LANGUAGES CXX VERSION 2)

add_library(catch2 INTERFACE)
target_include_directories(catch2 SYSTEM INTERFACE ${CMAKE_CURRENT_SOURCE_DIR})
