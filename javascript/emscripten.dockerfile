FROM emscripten/emsdk AS build_image

RUN apt-get -qq -y update \
    && DEBIAN_FRONTEND="noninteractive" \
       apt-get -qq install -y --no-install-recommends \
            libmsgpack-dev

FROM emscripten/emsdk AS final_image
COPY --from=build_image /usr/include/msgpack.* /msgpack/
COPY --from=build_image /usr/include/msgpack/  /msgpack/msgpack/
