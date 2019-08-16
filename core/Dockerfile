FROM ubuntu:19.10 as build-dependencies

RUN apt-get update\
    && apt-get -y install --no-install-recommends apt-utils  2>&1
RUN apt-get install -y ca-certificates
RUN update-ca-certificates

RUN apt-get -y install --no-install-recommends automake \
    make cmake gcc g++ curl \
    libc-dev linux-headers-generic git \
    build-essential autoconf libtool pkg-config \
    libgrpc++ libprotobuf-dev protobuf-compiler protobuf-compiler-grpc 2>&1

WORKDIR /tmp
RUN curl https://dl.google.com/go/go1.12.9.linux-amd64.tar.gz --output go1.12.9.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.12.9.linux-amd64.tar.gz 
ENV PATH="/usr/local/go/bin:${PATH}"
RUN mkdir -p /opt/go
ENV GOPATH="/opt/go"
ENV PATH="/opt/go/bin:${PATH}"
RUN go get -u  google.golang.org/grpc 
RUN go get -u github.com/golang/protobuf/protoc-gen-go

WORKDIR /
RUN git clone https://github.com/statoil/segyio

WORKDIR /segyio
RUN cmake . -DEXPERIMENTAL=ON -DBUILD_PYTHON=OFF -DBUILD_TESTING=OFF -DBUILD_SHARED_LIBS=OFF
RUN make && make install

# Cleanup 
WORKDIR /
RUN rm -rf /segyio
RUN rm -rf /tmp/**.*


FROM build-dependencies as builder
COPY core /src
COPY protos /protos

RUN protoc -I /protos --grpc_out=/src/core-server/proto --plugin=protoc-gen-grpc=`which grpc_cpp_plugin` /protos/core.proto
RUN protoc -I /protos --cpp_out=/src/core-server/proto /protos/core.proto

RUN mkdir -p /build
WORKDIR /build
RUN cmake ../src -DBUILD_SHARED_LIBS=off -DCMAKE_BUILD_TYPE=Release
RUN make

FROM alpine:latest as deploy
RUN apk --no-cache add libc-dev g++ ca-certificates nmap-ncat
COPY core/entrypoint.sh /usr/local/bin/

RUN chmod +x /usr/local/bin/entrypoint.sh

RUN adduser -D -g '' appuser
COPY --from=builder /build/stitch /bin/stitch
USER appuser
WORKDIR /home/appuser


EXPOSE 5000

ENTRYPOINT ["entrypoint.sh"]
CMD ["cat"]
