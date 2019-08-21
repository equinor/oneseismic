FROM golang:latest

RUN apt-get update
RUN apt-get install -y python3 python3-pip
RUN apt-get install -y protobuf-compiler
RUN pip3 install pytest pytest-azurepipelines requests

# Go build reporting tools
RUN go get github.com/jstemmer/go-junit-report
RUN go get github.com/axw/gocov/gocov
RUN go get github.com/AlekSi/gocov-xml
RUN go get github.com/matm/gocov-html

RUN go get github.com/swaggo/swag/cmd/swag
RUN go get github.com/golang/protobuf/protoc-gen-go
