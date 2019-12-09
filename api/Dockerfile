FROM golang:alpine as builder

ARG GITHUB_TOKEN
ENV GITHUB_TOKEN $GITHUB_TOKEN
RUN apk update && apk upgrade && apk add --no-cache ca-certificates git
RUN update-ca-certificates
RUN git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/equinor".insteadOf "https://github.com/equinor"

WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
ENV VERSION 'v0.1.0'
RUN CGO_ENABLED=0\
    GOOS=linux\
    GOARCH=amd64 \
    go build -ldflags="-w -s -X main.Version=${VERSION}"  -o /bin/sc-api

FROM scratch as deployer
COPY --from=builder /bin/sc-api /bin/sc-api
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080 8081
ENTRYPOINT ["sc-api","serve"]
