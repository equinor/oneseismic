FROM debian:buster-slim AS baseimg
ENV DEBIAN_FRONTEND=noninteractive
ENV FMT_VERSION 6.1.2
ENV ZMQ_VERSION 4.6.0

RUN apt-get update && apt-get install --no-install-recommends -y \
    build-essential \
    cmake \
    libgnutls28-dev libcurl4-gnutls-dev \
    protobuf-compiler-grpc libprotobuf-dev \
    pkg-config \
    wget \
    unzip \
    libzmq3-dev \
    libspdlog-dev \
    ca-certificates
RUN apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists

FROM baseimg AS dependencies
WORKDIR /src
RUN wget -q https://github.com/fmtlib/fmt/releases/download/${FMT_VERSION}/fmt-${FMT_VERSION}.zip
RUN wget -q https://github.com/zeromq/cppzmq/archive/v${ZMQ_VERSION}.zip
RUN unzip fmt-${FMT_VERSION}.zip && unzip v${ZMQ_VERSION}.zip

WORKDIR /src/fmt-${FMT_VERSION}/build
RUN cmake \
    -DCMAKE_BUILD_TYPE=Release \
    -DBUILD_SHARED_LIBS=ON \
    -DFMT_TEST=OFF \
    -DCMAKE_INSTALL_PREFIX=/usr \
    /src/fmt-${FMT_VERSION}
RUN make -j4 install

WORKDIR /src/cppzmq-${ZMQ_VERSION}/build
RUN cmake \
    -DCMAKE_BUILD_TYPE=Release \
    -DCPPZMQ_BUILD_TESTS=OFF \
    -DBUILD_SHARED_LIBS=ON \
    -DCMAKE_INSTALL_PREFIX=/usr \
    /src/cppzmq-${ZMQ_VERSION}
RUN make install -j2
RUN rm -rf /src

FROM dependencies AS source
WORKDIR /src
COPY core/ core
COPY protos/ protos

FROM source AS build
WORKDIR /src/build
RUN cmake \
    -DCMAKE_BUILD_TYPE=Release \
    -DBUILD_SHARED_LIBS=ON \
    -DBUILD_TESTING=OFF \
    -DBUILD_PYTHON=OFF \
    -DCMAKE_INSTALL_PREFIX=/usr \
    /src/core
RUN make -j4 install
RUN rm -rf /src
