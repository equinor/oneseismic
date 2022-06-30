project (oneseismic-javascript LANGUAGES CXX)
cmake_minimum_required(VERSION 3.13)

find_package(oneseismic REQUIRED)

option(ONESEISMIC_ASMJS_ONLY "Target asm.js (no WebAssembly)"        OFF)
option(ONESEISMIC_MODULARIZE "Modularize module; useful for node.js" OFF)

add_executable(oneseismic decoder.cpp)
target_link_libraries  (oneseismic PRIVATE oneseismic::oneseismic)
target_compile_features(oneseismic PRIVATE cxx_std_17)
target_link_options    (oneseismic
    PRIVATE
        --bind
	-s ALLOW_MEMORY_GROWTH=1
        --pre-js ${CMAKE_CURRENT_SOURCE_DIR}/decoding.js
)

if (ONESEISMIC_MODULARIZE)
  target_link_options(oneseismic PRIVATE "SHELL:-s MODULARIZE=1 ")
  target_link_options(oneseismic PRIVATE "SHELL:-s EXPORT_NAME='CreateOneSeismic' ") 
endif ()

if (ONESEISMIC_ASMJS_ONLY)
    target_link_options(oneseismic PRIVATE "SHELL:-s WASM=0")
else ()
    target_link_options(oneseismic PRIVATE "SHELL:-s WASM=1")
endif ()
