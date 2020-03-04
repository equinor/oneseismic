package tests

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	server "github.com/equinor/seismic-cloud/api/server"
)

func TestMain(m *testing.M) {
	ts := NewTestServiceSetup()

	opts := []server.HTTPServerOption{
		server.WithContainerURL(ts.ManifestStore),
	}
	s, err := server.NewHTTPServer(opts...)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		err = s.Serve()
		if err != nil {
			fmt.Println(err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	exitVal := m.Run()

	os.Exit(exitVal)
}
