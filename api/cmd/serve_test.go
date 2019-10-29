package cmd

import (
	goctx "context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/equinor/seismic-cloud/api/config"
	"github.com/equinor/seismic-cloud/api/server"
	"github.com/equinor/seismic-cloud/api/service"
	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"
)

var url = "localhost:8080"

func (m MockStitch) Stitch(ctx goctx.Context, ms store.Manifest, out io.Writer, in io.Reader) (string, error) {
	_, err := io.Copy(out, in)
	args := m.Called(ctx, ms, out, in)
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
		server.WithStitcher(MockStitch{}),
		server.WithHostAddr(url),
		server.WithHTTPOnly()}

	return opts
}

func waitForServer(url string, timeout time.Duration) error {
	ch := make(chan error, 1)
	go func() {
		for true {
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

func TestDefaultAPI(t *testing.T) {
	opts := createHTTPServerOptionsTest()

	go func() {
		serve(opts)
	}()
	timeout := time.Second
	httpURL := "http://" + url
	err := waitForServer(httpURL, timeout)

	if err != nil {
		t.Errorf("Could not start server within timeout of %v", timeout)
		return
	}

}

func Test_createHTTPServerOptions(t *testing.T) {
	tests := []struct {
		name        string
		want        []server.HTTPServerOption
		setDefaults bool
		wantErr     bool
	}{
		{"No config should fail", []server.HTTPServerOption{}, false, true},
		{"Default config should fail", []server.HTTPServerOption{}, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setDefaults {
				config.SetDefaults()
			}
			got, err := createHTTPServerOptions()
			if (err != nil) != tt.wantErr {
				t.Errorf("createHTTPServerOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("createHTTPServerOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_stitchConfig(t *testing.T) {
	tests := []struct {
		name      string
		setConfig func()
		want      interface{}
	}{
		{"No stitchconfig", func() { return }, nil},
		{"Cmd stitchconfig",
			func() {
				viper.SetDefault("STITCH_CMD", "FOO BAR")
				return
			}, []string{"FOO", "BAR"}},
		{"GRPC stitchconfig",
			func() {
				viper.SetDefault("STITCH_GRPC_ADDR", "localhost:10000")
				return
			}, service.GrpcOpts{Addr: "localhost:10000", Insecure: true}},
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
