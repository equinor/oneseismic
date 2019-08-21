FROM golang:latest

RUN apt-get update && apt-get install -y python-pip
RUN apt-get install -y protobuf-compiler
RUN pip install pytest pytest-azurepipelines requests
