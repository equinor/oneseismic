FROM golang:alpine as builder

ARG GITHUB_TOKEN
ENV GITHUB_TOKEN $GITHUB_TOKEN
RUN apk update && apk upgrade && apk add --no-cache git
RUN git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/equinor".insteadOf "https://github.com/equinor"
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
