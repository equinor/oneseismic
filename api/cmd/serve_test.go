package cmd

import (
	goctx "context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/equinor/seismic-cloud-api/api/config"
	"github.com/equinor/seismic-cloud-api/api/server"
	"github.com/equinor/seismic-cloud-api/api/service"
	"github.com/equinor/seismic-cloud-api/api/service/store"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var url = "localhost:8080"

func (m *MockStitch) Stitch(ctx goctx.Context, ms store.Manifest, out io.Writer, surfaceID string) (string, error) {
	_, err := io.Copy(out, strings.NewReader(surfaceID))
	args := m.Called(ctx, ms, out, surfaceID)
	if err != nil {
		return "", err
	}
	return args.String(0), args.Error(1)
}

type MockStitch struct {
	mock.Mock
}

func createHTTPServerOptionsTest() []server.HTTPServerOption {
	manifest, _ := json.Marshal(store.Manifest{
		Basename: "mock",
	})

	ms, _ := store.NewManifestStore(map[string][]byte{
		"default": manifest})
	ss, _ := store.NewSurfaceStore(map[string][]byte{})

	opts := []server.HTTPServerOption{
		server.WithManifestStore(ms),
		server.WithSurfaceStore(ss),
		server.WithStitcher(&MockStitch{}),
		server.WithHostAddr(url),
		server.WithHTTPOnly()}

	return opts
}

func waitForServer(url string, timeout time.Duration) error {
	ch := make(chan error, 1)
	go func() {
		for {
			time.Sleep(time.Millisecond * 10)
			res, err := http.Get(url)
			if err == nil {
				if res.StatusCode == 200 {
					ch <- nil
				}
			}
		}
	}()
	select {
	case re := <-ch:
		return re
	case <-time.After(timeout):
		return errors.New("Server not started")
	}
}

func TestServer(t *testing.T) {
	opts := createHTTPServerOptionsTest()

	go func() {
		err := serve(opts)
		if err != nil {
			t.Errorf("Serve %w", err)
		}
	}()
	timeout := time.Second
	httpURL := "http://" + url
	err := waitForServer(httpURL, timeout)

	if err != nil {
		t.Errorf("Could not start server within timeout of %v", timeout)
		return
	}

}

func Test_HTTPServerOptionsNeedsConfig(t *testing.T) {
	got, err := createHTTPServerOptions()
	assert.Errorf(t, err, "No config should fail. Got: %v", got)
}

func Test_createHTTPServerOptionsDefaults(t *testing.T) {
	config.SetDefaults()
	got, err := createHTTPServerOptions()
	assert.Errorf(t, err, "Default config should fail. Got options: %v", got)
}

func Test_stitchConfig(t *testing.T) {
	tests := []struct {
		name      string
		setConfig func()
		want      interface{}
	}{
		{"No stitchconfig", func() {}, nil},
		{"GRPC stitchconfig",
			func() {
				viper.SetDefault("STITCH_GRPC_ADDR", "example.com:12345")
			}, service.GrpcOpts{Addr: "example.com:12345", Insecure: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setConfig()
			got := stitchConfig()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stitchConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
