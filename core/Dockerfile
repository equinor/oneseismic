FROM alpine:latest as builder

RUN apk --no-cache add cmake clang clang-dev make gcc g++ libc-dev linux-headers git nmap-ncat

RUN git clone https://github.com/statoil/segyio
WORKDIR /segyio
RUN cmake . -DEXPERIMENTAL=ON -DBUILD_PYTHON=OFF -DBUILD_TESTING=OFF -DBUILD_SHARED_LIBS=OFF
RUN make
RUN make install

COPY . /src

RUN mkdir -p /build
WORKDIR /build
RUN cmake ../src -DBUILD_SHARED_LIBS=off -DCMAKE_BUILD_TYPE=Release
RUN make
RUN ls
FROM alpine:latest as deploy
RUN apk --no-cache add libc-dev g++ ca-certificates nmap-ncat

RUN adduser -D -g '' appuser
COPY --from=builder /build/stitch /bin/stitch
USER appuser
WORKDIR /home/appuser



EXPOSE 5000

ENTRYPOINT ["nc","-lk","-p", "5000","-e"]
CMD ["cat"]
