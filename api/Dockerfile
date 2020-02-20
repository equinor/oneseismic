FROM golang:buster as builder

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

RUN CGO_ENABLED=0\
    GOOS=linux\
    GOARCH=amd64 \
    go build -ldflags="-w -s"  -o /bin/sc-api

FROM scratch as deployer
COPY --from=builder /bin/sc-api /bin/sc-api
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080 8081
ENTRYPOINT ["sc-api"]
