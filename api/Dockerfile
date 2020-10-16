FROM golang:1.15-buster as builder

WORKDIR /src
RUN apt-get update && apt-get install -y libzmq5-dev

COPY api/go.mod .
COPY api/go.sum .
RUN go mod download

COPY api api
WORKDIR /src/api
RUN go test -race ./...
WORKDIR /src/api/cmd/query
RUN go build -v
WORKDIR /src/api/cmd/relay
RUN go build -v

FROM debian:buster-slim as deployer
RUN apt-get update && apt-get install -y libzmq5
COPY --from=builder /src/api/cmd/query/query /bin/oneseismic-query
COPY --from=builder /src/api/cmd/relay/relay /bin/oneseismic-relay
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
