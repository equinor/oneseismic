# Define a pretty big base build image, with C++ build-time dependencies
FROM debian:buster-slim AS buildimg
ENV DEBIAN_FRONTEND=noninteractive

# fmt and cppzmq are compiled from source, since buster's versions are slightly
# off
ENV FMT_VERSION 6.1.2
ENV ZMQ_VERSION 4.6.0

RUN apt-get update && apt-get install --no-install-recommends -y \
    build-essential \
    cmake \
    libgnutls28-dev libcurl4-gnutls-dev \
    pkg-config \
    wget \
    unzip \
    libzmq5-dev \
    libspdlog-dev \
    libhiredis-dev \
    libmsgpack-dev \
    ca-certificates

WORKDIR /src
RUN wget -q https://github.com/fmtlib/fmt/releases/download/${FMT_VERSION}/fmt-${FMT_VERSION}.zip
RUN wget -q https://github.com/zeromq/cppzmq/archive/v${ZMQ_VERSION}.zip
RUN unzip fmt-${FMT_VERSION}.zip && unzip v${ZMQ_VERSION}.zip

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

WORKDIR /src/cppzmq-${ZMQ_VERSION}/build
RUN cmake \
    -DCMAKE_BUILD_TYPE=Release \
    -DCPPZMQ_BUILD_TESTS=OFF \
    -DBUILD_SHARED_LIBS=OFF \
    -DCMAKE_INSTALL_PREFIX=/usr/local \
    /src/cppzmq-${ZMQ_VERSION}
RUN make -j4 install
RUN rm -rf /src

FROM buildimg AS cppbuilder
WORKDIR /src
COPY core/ core

WORKDIR /src/build
RUN cmake \
    -DCMAKE_BUILD_TYPE=Release \
    -DBUILD_SHARED_LIBS=OFF \
    -DBUILD_TESTING=OFF \
    -DBUILD_PYTHON=OFF \
    -DCMAKE_INSTALL_PREFIX=/usr/local \
    /src/core
RUN make -j4 install

FROM golang:1.15-buster as gobuilder
COPY --from=cppbuilder /usr/local /usr/local
RUN apt-get update && apt-get install -y libzmq5-dev

WORKDIR /src
COPY api/go.mod .
COPY api/go.sum .
RUN go mod download

COPY api api
WORKDIR /src/api
RUN go test -race ./...
WORKDIR /src/api/cmd/query
RUN go build -v
WORKDIR /src/api/cmd/fetch
RUN go build -v

# The final image with only the binaries and runtime dependencies
FROM debian:buster-slim as deployimg
ENV DEBIAN_FRONTEND=noninteractive
RUN    apt-get update \
    && apt-get install -y \
        libzmq5 \
        libgnutls28-dev \
        libcurl3-gnutls \
        libhiredis0.14 \
        ca-certificates \
    && apt-get clean -y \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists

COPY --from=gobuilder /src/api/cmd/query/query /bin/oneseismic-query
COPY --from=gobuilder /src/api/cmd/fetch/fetch /bin/oneseismic-fetch

COPY --from=cppbuilder /usr/local/bin/oneseismic-manifest /bin/oneseismic-manifest
COPY --from=cppbuilder /usr/local/bin/oneseismic-fragment /bin/oneseismic-fragment
