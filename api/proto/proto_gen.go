package seismic_core

//go:generate go get -u github.com/golang/protobuf/protoc-gen-go
//go:generate protoc -I ../../protos ../../protos/core.proto --go_out=plugins=grpc:.
