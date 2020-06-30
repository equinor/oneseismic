package main

//Swagger doc Generate
//go:generate swag init

//protobuf generation
//go:generate mkdir -p oneseismic
//go:generate protoc -I ../protos ../protos/core.proto --go_out=oneseismic
