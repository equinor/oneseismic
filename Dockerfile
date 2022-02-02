# Define a pretty big base build image, with C++ build-time dependencies
FROM debian:buster-slim AS buildimg
ENV DEBIAN_FRONTEND=noninteractive

# fmt is compiled from source, since buster's versions are slightly off
ENV FMT_VERSION 6.1.2

RUN apt-get update && apt-get install --no-install-recommends -y \
    build-essential \
    cmake \
    pkg-config \
    wget \
    unzip \
    libspdlog-dev \
    libmsgpack-dev \
    ca-certificates

WORKDIR /src
RUN wget -q https://github.com/fmtlib/fmt/releases/download/${FMT_VERSION}/fmt-${FMT_VERSION}.zip
RUN unzip fmt-${FMT_VERSION}.zip

# Since this is a docker build, just build the dependencies statically. It
# makes copying a tad easier, and it makes go happy. This is in no way a
# requirement though, and oneseismic is perfectly happy to dynamically link.
WORKDIR /src/fmt-${FMT_VERSION}/build
RUN cmake \
    -DCMAKE_BUILD_TYPE=Release \
    -DBUILD_SHARED_LIBS=OFF \
    -DFMT_TEST=OFF \
    -DCMAKE_INSTALL_PREFIX=/usr/local \
    /src/fmt-${FMT_VERSION}
RUN make -j4 install

FROM buildimg AS cppbuilder
WORKDIR /src
COPY core/ core

WORKDIR /src/build
RUN cmake \
    -DCMAKE_BUILD_TYPE=Release \
    -DBUILD_SHARED_LIBS=OFF \
    -DBUILD_TESTING=OFF \
    -DCMAKE_CXX_FLAGS=-DFMT_HEADER_ONLY=1 \
    -DCMAKE_INSTALL_PREFIX=/usr/local \
    /src/core
RUN make -j4 install

FROM golang:1.16-buster as gobuilder
COPY --from=cppbuilder /usr/local /usr/local

WORKDIR /src
COPY api/go.mod .
COPY api/go.sum .
RUN go mod download

COPY api api
WORKDIR /src/api
RUN go test -race ./...
RUN go install ./...

# The final image with only the binaries and runtime dependencies
FROM debian:buster-slim as deployimg
ENV DEBIAN_FRONTEND=noninteractive
RUN    apt-get update \
    && apt-get install -y \
        ca-certificates \
    && apt-get clean -y \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists

COPY --from=gobuilder /go/bin/query  /bin/oneseismic-query
COPY --from=gobuilder /go/bin/result /bin/oneseismic-result
COPY --from=gobuilder /go/bin/fetch  /bin/oneseismic-fetch
COPY --from=gobuilder /go/bin/gc     /bin/oneseismic-gc
