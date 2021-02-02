FROM debian:buster-slim AS baseimg
ENV DEBIAN_FRONTEND=noninteractive
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
    -DCMAKE_INSTALL_PREFIX=/usr \
    /src/cppzmq-${ZMQ_VERSION}
RUN make install -j2
RUN rm -rf /src

FROM baseimg AS cppbuilder
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

FROM golang:1.15-buster as builder

COPY --from=cppbuilder /usr/local /usr/local

WORKDIR /src
RUN apt-get update && apt-get install -y libzmq5-dev

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

FROM debian:buster-slim as deployer
RUN apt-get update && apt-get install -y libzmq5
COPY --from=builder /src/api/cmd/query/query /bin/oneseismic-query
COPY --from=builder /src/api/cmd/fetch/fetch /bin/oneseismic-fetch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
