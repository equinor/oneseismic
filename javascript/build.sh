#!/usr/bin/env bash

set -xe

BUILD_DIR="$1"

emcmake cmake \
    -S core/ -B $BUILD_DIR/core/ \
    -DBUILD_CORE=OFF \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_CXX_FLAGS=-I/msgpack \
    -DCMAKE_TOOLCHAIN_FILE=/emsdk/upstream/emscripten/cmake/Modules/Platform/Emscripten.cmake \

pushd $BUILD_DIR/core
emmake make
popd

emcmake cmake \
    -S javascript/ -B $BUILD_DIR/javascript/ \
    -Doneseismic_DIR=$BUILD_DIR/core/ \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_CXX_FLAGS=-I/msgpack \
    -DCMAKE_INTERPROCEDURAL_OPTIMIZATION=1 \
    -DCMAKE_TOOLCHAIN_FILE=/emsdk/upstream/emscripten/cmake/Modules/Platform/Emscripten.cmake \

pushd $BUILD_DIR/javascript
emmake make
popd
