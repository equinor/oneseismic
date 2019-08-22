FROM alpine:edge as builder

RUN apk --no-cache add cmake clang clang-dev make gcc g++ libc-dev linux-headers git nmap-ncat python py-pip python-dev

RUN git clone https://github.com/statoil/segyio
WORKDIR /segyio
RUN cmake . -DEXPERIMENTAL=ON -DBUILD_PYTHON=OFF -DBUILD_TESTING=OFF -DBUILD_SHARED_LIBS=OFF
RUN make
RUN make install

COPY . /src

RUN python -m  pip install scikit-build pybind11
RUN mkdir -p /build
WORKDIR /build
RUN cmake ../src -DBUILD_SHARED_LIBS=off -DCMAKE_BUILD_TYPE=Release
RUN make

FROM alpine:latest as deploy
RUN apk --no-cache add libc-dev g++ ca-certificates nmap-ncat
COPY entrypoint.sh /usr/local/bin/

RUN chmod +x /usr/local/bin/entrypoint.sh

RUN adduser -D -g '' appuser
COPY --from=builder /build/ /usr/local/bin/
USER appuser
WORKDIR /home/appuser


EXPOSE 5000

# ENTRYPOINT ["entrypoint.sh"]
# CMD ["cat"]
