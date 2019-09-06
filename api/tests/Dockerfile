FROM golang:latest

RUN apt-get update && apt-get install -y python-pip
RUN apt-get install -y protobuf-compiler
RUN pip install pytest pytest-azurepipelines requests

RUN mkdir dep
ADD api/go.mod dep/

# Go build reporting tools
RUN go get github.com/jstemmer/go-junit-report
RUN go get github.com/axw/gocov/gocov
RUN go get github.com/AlekSi/gocov-xml
RUN go get github.com/matm/gocov-html

RUN go get github.com/swaggo/swag/cmd/swag
RUN go get github.com/golang/protobuf/protoc-gen-go


# Install go.mod dependencies
WORKDIR /go/dep
RUN go mod download
WORKDIR /go
