package cmd

import (
	"reflect"
	"testing"

	"github.com/equinor/seismic-cloud-api/api/config"
	"github.com/equinor/seismic-cloud-api/api/service"
	"github.com/stretchr/testify/assert"
)

func Test_HTTPServerOptionsNeedsConfig(t *testing.T) {
	_, err := createHTTPServerOptions()
	assert.Error(t, err)
}

func Test_createHTTPServerOptionsDefaults(t *testing.T) {
	config.SetDefaults()
	_, err := createHTTPServerOptions()
	assert.Error(t, err)
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
				config.SetDefault("STITCH_GRPC_ADDR", "example.com:12345")
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
