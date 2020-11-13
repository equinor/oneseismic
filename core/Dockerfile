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
    -DCMAKE_INSTALL_PREFIX=/usr \
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

FROM baseimg AS builder
WORKDIR /src
COPY core/ core

WORKDIR /src/build
RUN cmake \
    -DCMAKE_BUILD_TYPE=Release \
    -DBUILD_SHARED_LIBS=OFF \
    -DBUILD_TESTING=OFF \
    -DBUILD_PYTHON=OFF \
    -DCMAKE_INSTALL_PREFIX=/usr \
    /src/core
RUN make -j4 install

FROM debian:buster-slim as deployer
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install --no-install-recommends -y \
    libzmq5 \
    libcurl4 \
    libcurl4-gnutls-dev \
    libhiredis0.14 \
    ca-certificates
RUN apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists
COPY --from=builder /usr/bin/oneseismic-manifest /bin/oneseismic-manifest
COPY --from=builder /usr/bin/oneseismic-fragment /bin/oneseismic-fragment
