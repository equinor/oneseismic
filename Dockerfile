FROM alpine:latest as stitch-build

RUN apk --no-cache add cmake clang clang-dev make gcc g++ libc-dev linux-headers git
COPY core /src

RUN git clone https://github.com/statoil/segyio
WORKDIR /segyio
RUN cmake . -DEXPERIMENTAL=ON -DBUILD_PYTHON=OFF -DBUILD_TESTING=OFF -DBUILD_SHARED_LIBS=OFF
RUN make
RUN make install



RUN mkdir -p /build
WORKDIR /build
RUN cmake ../src -DBUILD_SHARED_LIBS=off -DCMAKE_BUILD_TYPE=Release
RUN make
RUN ls

FROM golang:1.12-alpine3.9 as api-build
RUN apk --no-cache add git
COPY api /src
WORKDIR /src
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s"  -o /bin/sc-api

FROM alpine as deploy
RUN apk --no-cache add libc-dev g++

RUN adduser -D -g '' appuser
COPY --from=stitch-build /build/stitch /bin/stitch
COPY --from=api-build /bin/sc-api /bin/sc-api
USER appuser
WORKDIR /home/appuser

EXPOSE 8080
ENTRYPOINT ["sc-api","--config","/conf/api.yml","serve"]
