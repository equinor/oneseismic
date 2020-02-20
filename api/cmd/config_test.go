package cmd

import (
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		f           interface{}
		set         interface{}
		wantDefault interface{}
		wantSet     interface{}
	}{
		{"Issuer", "ISSUER", Issuer, "test", "", "test"},
		{"HostAddr", "HOST_ADDR", HostAddr, "localhost:8080", "", "localhost:8080"},
		{"StitchGrpcAddr", "STITCH_GRPC_ADDR", StitchGrpcAddr, "test", "", "test"},
		{"ResourceID", "RESOURCE_ID", ResourceID, "test", "", "test"},
		{"Profiling", "PROFILING", Profiling, true, false, true},
		{"ManifestStoragePath", "MANIFEST_PATH", ManifestStoragePath, "test", "", "test"},
		{"ManifestURI", "MANIFEST_DB_URI", ManifestDbURI, "mongodb://", "", "mongodb://"},
		{"AzStorageAccount", "AZURE_STORAGE_ACCOUNT", AzStorageAccount, "test", "", "test"},
		{"AzStorageKey", "AZURE_STORAGE_ACCESS_KEY", AzStorageKey, "test", "", "test"},
		{"AzSurfaceContainerName", "AZURE_SURFACE_CONTAINER", AzSurfaceContainerName, "scblob", "", "scblob"},
		{"AzManifestContainerName", "AZURE_MANIFEST_CONTAINER", AzManifestContainerName, "scmanifest", "", "scmanifest"},
		{"AzStorageURL", "AZURE_STORAGE_URL", AzStorageURL, "https://%s.blob.core.windows.net/%s", "", "https://%s.blob.core.windows.net/%s"},
		{"LocalSurfacePath", "LOCAL_SURFACE_PATH", LocalSurfacePath, "tmp/", "", "tmp/"},
		{"LogDBConnStr", "LOGDB_CONNSTR", LogDBConnStr, "test", "", "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.f)
			r := v.Call(nil)
			assert.Equal(t, tt.wantDefault, r[0].Interface(), fmt.Sprintf("Before setting any variables. wantDefault: %v, got: %v", tt.wantDefault, r[0].Interface()))
			SetDefault(tt.key, tt.set)
			v = reflect.ValueOf(tt.f)
			r = v.Call(nil)
			assert.Equal(t, tt.wantSet, r[0].Interface(), fmt.Sprintf("After setting variables. wantSet: %v, got: %v", tt.wantDefault, r[0].Interface()))
		})
	}
	Reset()
}

func TestNoAuth(t *testing.T) {
	Reset()
	assert.Equal(t, true, UseAuth())
	SetDefault("NO_AUTH", true)
	assert.Equal(t, false, UseAuth())
}

func TestAuthServer(t *testing.T) {
	Reset()
	_, err := AuthServer()
	assert.Error(t, err)

	anURL := &url.URL{Scheme: "http", Host: "some.host"}
	SetDefault("AUTHSERVER", anURL)
	u, err := AuthServer()
	assert.NoError(t, err)
	assert.Equal(t, anURL, u)
}
