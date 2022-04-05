FROM emscripten/emsdk AS base_image

RUN apt-get -qq -y update \
    && DEBIAN_FRONTEND="noninteractive" \
       apt-get -qq install -y --no-install-recommends \
            libmsgpack-dev

FROM emscripten/emsdk AS build_image
#Copy dependencies
COPY --from=base_image /usr/include/msgpack.* /msgpack/
COPY --from=base_image /usr/include/msgpack/  /msgpack/msgpack/

#Copy build parts
COPY  core/src core/src
COPY  core/external core/external
COPY  core/include core/include
COPY  core/CMakeLists.txt core/.

COPY  javascript/decoder.cpp javascript/decoder.cpp
COPY  javascript/decoding.js javascript/decoding.js
COPY  javascript/CMakeLists.txt javascript/CMakeLists.txt

COPY  javascript/build.sh /build.sh

RUN /build.sh /build \
    /msgpack \
    /emsdk/upstream/emscripten/cmake/Modules/Platform/Emscripten.cmake 

FROM scratch as artifact
#Copy out build artifacts 
#This also allow buildkit to be used to output artifacts to local filesystem eg.
# DOCKER_BUILDKIT=1 docker build --output type=local,dest=build -f javascript/emscripten.dockerfile .
COPY --from=build_image /build/javascript/oneseismic.js /wasm/oneseismic.js
COPY --from=build_image /build/javascript/oneseismic.wasm /wasm/oneseismic.wasm
