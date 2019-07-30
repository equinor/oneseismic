#!/bin/bash
# Build seismic-cloud core and create sample cubes and surfaces

rm -rf core/build
rm -rf tmp/testfiles
mkdir -p core/build
mkdir -p tmp/testfiles/cubes
mkdir -p tmp/testfiles/surfaces
pushd core/build
cmake .. -DCMAKE_BUILD_TYPE=Release
make  -j4
popd
./core/build/generate_surfaces tmp/testfiles/surfaces/
./core/build/generate 1000 1000 1000 100 100 100 -o tmp/testfiles/cubes/
