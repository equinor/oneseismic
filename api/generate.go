package main

//Swagger doc Generate
//go:generate go get github.com/swaggo/swag/cmd/swag
//go:generate swag init

//protobuf generation
//go:generate mkdir -p oneseismic
//go:generate go get google.golang.org/protobuf/cmd/protoc-gen-go
//go:generate protoc -I ../protos ../protos/core.proto --go_out=oneseismic
