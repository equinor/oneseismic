package main

//Swagger doc Generate
//go:generate go get github.com/swaggo/swag/cmd/swag
//go:generate swag init

//protobuf generation
//go:generate mkdir -p core
//go:generate go get github.com/golang/protobuf/protoc-gen-go
//go:generate protoc -I ../protos ../protos/core.proto --go_out=import_path=core:core
