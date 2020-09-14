package main

//protobuf generation
//go:generate mkdir -p oneseismic
//go:generate go install google.golang.org/protobuf/cmd/protoc-gen-go
//go:generate protoc -I ../protos ../protos/core.proto --go_out=oneseismic
