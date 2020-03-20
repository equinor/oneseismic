FROM golang:buster as base
RUN apt update && apt install -y libsodium-dev libzmq3-dev libczmq-dev
FROM base as builder
RUN apt update && apt install -y protobuf-compiler
WORKDIR /src
COPY api/go.mod .
COPY api/go.sum .
RUN go mod download

COPY api api
COPY protos protos

WORKDIR /src/api

RUN go generate

RUN go test -race ./...

RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s"

FROM base
COPY --from=builder /src/api/api /bin/api

EXPOSE 8080 8081
ENTRYPOINT ["api"]
