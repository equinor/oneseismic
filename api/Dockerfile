FROM golang:alpine as builder
RUN apk update && apk upgrade && apk add --no-cache ca-certificates
RUN apk --no-cache add git
RUN update-ca-certificates

WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s"  -o /bin/sc-api

FROM scratch as deploy
COPY --from=builder /bin/sc-api /bin/sc-api
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080 8081
ENTRYPOINT ["sc-api","serve"]
