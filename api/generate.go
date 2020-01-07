package main

//Swagger doc Generate
//go:generate go get github.com/swaggo/swag/cmd/swag@v1.6.3
//go:generate swag init

//GRPC Service generate definition
//go:generate go get github.com/golang/protobuf/protoc-gen-go
//go:generate protoc -I ../protos ../protos/core.proto --go_out=plugins=grpc:proto
