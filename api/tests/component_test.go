package tests

import (
	"context"
	"net/http"

	"github.com/stretchr/testify/assert"

	"fmt"
	"log"
	"os"
	"testing"
	"time"

	cserver "github.com/equinor/seismic-cloud-api/api/corestub/server"
	server "github.com/equinor/seismic-cloud-api/api/server"
)

const apiurl = "localhost:18080"
const csurl = "localhost:10000"

func TestMain(m *testing.M) {
	ts := NewTestServiceSetup()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := cserver.StartServer(ctx, csurl)
		if err != nil {
			fmt.Println(err)
		}
	}()
	defer cancel()

	opts := []server.HTTPServerOption{
		server.WithManifestStore(ts.ManifestStore),
		server.WithSurfaceStore(ts.SurfaceStore),
		server.WithStitcher(ts.Stitch),
		server.WithHostAddr(apiurl),
		server.WithHTTPOnly()}
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

func TestStitchNoSurface(t *testing.T) {

	resp, err := http.Get("http://" + apiurl + "/stitch/exists/not-exists")
	assert.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestStitchNoManifest(t *testing.T) {

	resp, err := http.Get("http://" + apiurl + "/stitch/not-exists/exists")
	assert.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
