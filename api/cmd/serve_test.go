package cmd

import (
	"reflect"
	"testing"

	"github.com/equinor/seismic-cloud-api/api/config"
	"github.com/equinor/seismic-cloud-api/api/server"
	"github.com/equinor/seismic-cloud-api/api/service"
	"github.com/equinor/seismic-cloud-api/api/tests"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)


func createHTTPServerOptionsTest() []server.HTTPServerOption {
	s := tests.NewTestServiceSetup()
	opts := []server.HTTPServerOption{
		server.WithManifestStore(s.ManifestStore),
		server.WithSurfaceStore(s.SurfaceStore),
		server.WithStitcher(s.Stitch),
		server.WithHostAddr("localhost:8080"),
		server.WithHTTPOnly()}

	return opts
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
