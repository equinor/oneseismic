FROM python:3-alpine3.10 as builder

RUN apk --no-cache add cmake clang clang-dev make gcc g++ libc-dev linux-headers git nmap-ncat
RUN pip3 install scikit-build pybind11

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

FROM python:3-alpine3.10 as deploy
RUN apk --no-cache add libc-dev g++ ca-certificates nmap-ncat
COPY entrypoint.sh /usr/local/bin/
RUN dos2unix /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

RUN adduser -D -g '' appuser
COPY --from=builder /build/stitch /bin/stitch
USER appuser
WORKDIR /home/appuser


EXPOSE 5000

ENTRYPOINT ["entrypoint.sh"]
CMD ["cat"]
