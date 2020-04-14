FROM golang:buster as builder

WORKDIR /src
RUN apt-get update && apt-get install -y libzmq5-dev
COPY api/go.mod .
COPY api/go.sum .
RUN go mod download

COPY api api

WORKDIR /src/api
RUN go generate

RUN go test -race ./...

RUN GOOS=linux\
    GOARCH=amd64 \
    go build -ldflags="-w -s"

FROM debian:buster-slim as deployer
RUN apt-get update && apt-get install -y libzmq5
COPY --from=builder /src/api/api /bin/api

EXPOSE 8080 8081
ENTRYPOINT ["api"]
