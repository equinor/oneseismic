package main

import (
	"context"
	"log"
	"os"

	"github.com/equinor/seismic-cloud-api/api/corestub/server"
)

func main() {

	hostAddr := os.Getenv("SC_GRPC_HOST_ADDR")
	if len(hostAddr) < 1 {
		hostAddr = "localhost:10000"
	}

	log.Fatalf("Starting corestub server: %v", server.StartServer(context.Background(), hostAddr))
}
