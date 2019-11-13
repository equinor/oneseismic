FROM golang as builder

WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go test -race -v ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s"  -o /bin/sc-api

FROM scratch as deploy
COPY --from=builder /bin/sc-api /bin/sc-api
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080 8081
ENTRYPOINT ["sc-api","serve"]
