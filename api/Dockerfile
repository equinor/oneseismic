FROM golang as builder

WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
ARG COVERAGE_LIMIT=40
RUN ./check_test_coverage.sh
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/sc-api
RUN CGO_ENABLED=0 go test ./...

FROM scratch as deploy
COPY --from=builder /bin/sc-api /bin/sc-api

EXPOSE 8080 8081
ENTRYPOINT ["sc-api","serve"]
