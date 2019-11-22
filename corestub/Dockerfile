FROM golang:alpine as builder

WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s"  -o /bin/sc-corestub

FROM scratch as deploy
COPY --from=builder /bin/sc-corestub /bin/sc-corestub

EXPOSE 10000
ENTRYPOINT ["sc-corestub","serve"]
