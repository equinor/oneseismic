FROM ubuntu:devel as builder

RUN apt-get update && apt-get install -y build-essential
RUN apt-get install -y cmake
RUN apt-get install -y libgnutls28-dev libcurl4-gnutls-dev
RUN apt-get install -y protobuf-compiler-grpc libprotobuf-dev libgrpc++-dev
RUN apt-get install -y pkg-config
RUN apt-get install -y wget
RUN apt-get install -y unzip

WORKDIR /src/fmt
RUN wget https://github.com/fmtlib/fmt/releases/download/6.0.0/fmt-6.0.0.zip
RUN unzip fmt-6.0.0.zip
RUN mkdir -p /src/fmt/fmt-6.0.0/build
RUN cd fmt-6.0.0/build
RUN cmake /src/fmt/fmt-6.0.0 -DCMAKE_BUILD_TYPE=Release -DBUILD_TESTING=OFF -DBUILD_SHARED_LIBS=ON
RUN make install 
RUN rm -rf /src/fmt


WORKDIR /src
COPY . .
WORKDIR /build
RUN cmake /src/core -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF
RUN make

FROM ubuntu:devel as dep
RUN apt-get update && apt-get upgrade -y
RUN apt-get install -y libproto libgrpc++
COPY --from=builder /build/one-server /usr/local/bin/one-server


RUN one-server
CMD [ "./one-server"]
