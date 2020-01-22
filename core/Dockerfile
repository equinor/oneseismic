FROM ubuntu:devel as builder

ENV FMT_VERSION 6.1.2

RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    libgnutls28-dev libcurl4-gnutls-dev \
    protobuf-compiler-grpc libprotobuf-dev libgrpc++-dev \
    pkg-config \
    wget \
    unzip 

WORKDIR /src/fmt
RUN wget https://github.com/fmtlib/fmt/releases/download/${FMT_VERSION}/fmt-${FMT_VERSION}.zip
RUN unzip fmt-${FMT_VERSION}.zip
RUN mkdir -p /src/fmt/fmt-${FMT_VERSION}/build
RUN cd fmt-${FMT_VERSION}/build
RUN cmake /src/fmt/fmt-${FMT_VERSION} -DCMAKE_BUILD_TYPE=Release -DFMT_TEST=OFF -DBUILD_SHARED_LIBS=ON
RUN make install 
RUN rm -rf /src/fmt


WORKDIR /src
COPY core core
COPY protos protos
WORKDIR /build
RUN cmake /src/core -DCMAKE_BUILD_TYPE=Release -DBUILD_TESTING=OFF -DBUILD_SHARED_LIBS=ON
RUN make 

ENTRYPOINT [ "./one-server"]
