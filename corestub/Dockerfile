FROM golang:alpine as builder
RUN apk update && apk upgrade && apk add --no-cache ca-certificates
RUN apk --no-cache add git
RUN update-ca-certificates

WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s"  -o /bin/sc-corestub

FROM scratch as deploy
COPY --from=builder /bin/sc-corestub /bin/sc-corestub
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 10000
ENTRYPOINT ["sc-corestub","serve"]
