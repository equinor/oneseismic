package main

//Swagger doc Generate
//go:generate go get -u github.com/swaggo/swag/cmd/swag
//go:generate swag init

//GRPC Service generate definition
//go:generate mkdir -p proto
//go:generate go get github.com/golang/protobuf/protoc-gen-go
//go:generate protoc -I ../protos ../protos/core.proto --go_out=plugins=grpc:proto
