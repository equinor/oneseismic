#!/usr/bin/env bash
# Script to build emscripten library

set -xe

BUILD_DIR=$(realpath $1)
INCLUDE_DIR="$2"
TOOLCHAIN_FILE="$3"
shift 3

#First build core lib
emcmake cmake \
    -S core/ -B $BUILD_DIR/core/ \
    -DBUILD_CORE=OFF \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_CXX_FLAGS=-I$INCLUDE_DIR \
    -DCMAKE_TOOLCHAIN_FILE=$TOOLCHAIN_FILE 

pushd $BUILD_DIR/core
emmake make
popd

#Then build our target
emcmake cmake \
    -S javascript/ -B $BUILD_DIR/javascript/ \
    -Doneseismic_DIR=$BUILD_DIR/core/ \
    -DONESEISMIC_MODULARIZE=1  \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_CXX_FLAGS=-I$INCLUDE_DIR \
    -DCMAKE_INTERPROCEDURAL_OPTIMIZATION=1 \
    -DCMAKE_TOOLCHAIN_FILE=$TOOLCHAIN_FILE  \
    "$@" \

pushd $BUILD_DIR/javascript
emmake make
popd
