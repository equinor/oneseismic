package cmd

import (
	"reflect"
	"testing"

	"github.com/spf13/viper"

	"github.com/equinor/seismic-cloud/api/config"
	"github.com/equinor/seismic-cloud/api/server"
	"github.com/equinor/seismic-cloud/api/service"
)

func TestServer(t *testing.T) {
	//TODO: integration test
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
