package main

import (
	"os"

	"github.com/equinor/seismic-cloud-api/corestub/server"
)

func main() {

	hostAddr := os.Getenv("SC_GRPC_HOST_ADDR")
	if len(hostAddr) < 1 {
		hostAddr = "localhost:10000"
	}
	server.StartServer(hostAddr)

}
