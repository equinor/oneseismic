FROM emscripten/emsdk AS emscripten

RUN apt-get -qq -y update \
    && DEBIAN_FRONTEND="noninteractive" \
       apt-get -qq install -y --no-install-recommends \
            libmsgpack-dev

FROM emscripten AS cppbuild
COPY --from=emscripten /usr/include/msgpack.* /msgpack/
COPY --from=emscripten /usr/include/msgpack/  /msgpack/msgpack/

RUN apt-get update && apt-get install --no-install-recommends -y \
    cmake
COPY ./core ./core

RUN emcmake cmake \
    -S core/ -B /src/js/core/ \
    -DBUILD_CORE=OFF \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_CXX_FLAGS=-I/msgpack \
    -DCMAKE_TOOLCHAIN_FILE=/emsdk/upstream/emscripten/cmake/Modules/Platform/Emscripten.cmake
WORKDIR /src/js/core/
RUN emmake make


FROM cppbuild AS jsbuild
COPY ./javascript /javascript

WORKDIR /
RUN emcmake cmake \
    -S javascript/ -B /src/js/javascript/ \
    -Doneseismic_DIR=/src/js/core/ \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_CXX_FLAGS=-I/msgpack \
    -DCMAKE_INTERPROCEDURAL_OPTIMIZATION=1 \
    -DCMAKE_TOOLCHAIN_FILE=/emsdk/upstream/emscripten/cmake/Modules/Platform/Emscripten.cmake \
    -DONESEISMIC_MODULARIZE=ON \
    -DCMAKE_INTERPROCEDURAL_OPTIMIZATION=OFF
WORKDIR /src/js/javascript/
RUN emmake make


FROM node:16-buster AS build
COPY /python /python
RUN apt-get update && apt-get install --no-install-recommends -y \
    python3-venv
RUN python3 -m venv /pyenv/upload_with

RUN /pyenv/upload_with/bin/python -m pip install --upgrade pip
RUN /pyenv/upload_with/bin/pip install -r /python/requirements-dev.txt

ARG UPLOAD_WITH_CLIENT_VERSION
ENV UPLOAD_WITH_CLIENT_VERSION=$UPLOAD_WITH_CLIENT_VERSION
RUN if test -z "$UPLOAD_WITH_CLIENT_VERSION"; \
    then echo /python > /pyenv/upload_with/lib/python3.7/site-packages/oneseismic.pth; \
    else /pyenv/upload_with/bin/pip install oneseismic=="$UPLOAD_WITH_CLIENT_VERSION"; \
    fi


FROM build AS jstest
COPY --from=jsbuild /src/js/javascript/ /build
COPY /tests /tests

WORKDIR /tests/javascript
RUN npm install

ENV UPLOAD_WITH_PYTHON /pyenv/upload_with/bin/python
ENV NODE_PATH /build:$NODE_PATH


FROM jstest AS localtest
ENTRYPOINT ["npm","run", "testintegration"]
