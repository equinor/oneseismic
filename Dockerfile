FROM alpine as stitch-build

RUN apk --no-cache add cmake clang clang-dev make gcc g++ libc-dev linux-headers git
COPY core /src

RUN git clone https://github.com/statoil/segyio
WORKDIR /segyio
RUN cmake . -DEXPERIMENTAL=ON -DBUILD_PYTHON=OFF -DBUILD_TESTING=OFF -DBUILD_SHARED_LIBS=OFF
RUN make
RUN make install



RUN mkdir -p /build
WORKDIR /build
RUN cmake ../src -DCMAKE_BUILD_TYPE=Release
RUN make


FROM golang:1.12-alpine3.9 as api-build
RUN apk --no-cache add git
RUN adduser -D -g '' appuser
COPY api /src
WORKDIR /src
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s"  -o /bin/sc-api 

FROM scratch as deploy
COPY --from=api-build /etc/passwd /etc/passwd
COPY --from=stitch-build /build/stitch /stitch
COPY --from=api-build /bin/sc-api /sc-api
USER appuser

EXPOSE 8080
ENTRYPOINT ["/sc-api","--config","/conf/api.yml","serve"]
