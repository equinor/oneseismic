package config

import (
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"github.com/spf13/viper"
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
		{"DomainList", "DOMAIN_LIST", DomainList, "test", "", "test"},
		{"DomainMail", "DOMAIN_MAIL", DomainMail, "test", "", "test"},
		{"StitchGrpcAddr", "STITCH_GRPC_ADDR", StitchGrpcAddr, "test", "", "test"},
		{"ResourceID", "RESOURCE_ID", ResourceID, "test", "", "test"},
		{"CertFile", "CERT_FILE", CertFile, "cert.crt", "", "cert.crt"},
		{"KeyFile", "KEY_FILE", KeyFile, "cert.key", "", "cert.key"},
		{"HTTPOnly", "HTTP_ONLY", HTTPOnly, true, false, true},
		{"UseTLS", "TLS", UseTLS, true, false, true},
		{"UseLetsEncrypt", "LETSENCRYPT", UseLetsEncrypt, true, false, true},
		{"Profiling", "PROFILING", Profiling, true, false, true},
		{"Swagger", "SWAGGER", Swagger, true, false, true},
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
			viper.SetDefault(tt.key, tt.set)
			v = reflect.ValueOf(tt.f)
			r = v.Call(nil)
			assert.Equal(t, tt.wantSet, r[0].Interface(), fmt.Sprintf("After setting variables. wantSet: %v, got: %v", tt.wantDefault, r[0].Interface()))
		})
	}
	viper.Reset()
}

func TestAuth(t *testing.T) {
	assert.Equal(t, true, UseAuth(), fmt.Sprintf("UseAuth default value, expected: %v, actual: %v", true, UseAuth()))
	assert.Empty(t, AuthServer(), fmt.Sprint("AuthServer default value should be empty"))

	viper.SetDefault("NO_AUTH", false)
	viper.SetDefault("AUTHSERVER", "http://oauth2.example.com")
	if err := Load(); err != nil {
		assert.NoError(t, err, "Error loading config")
	}
	uExpected := &url.URL{Scheme: "http", Host: "oauth2.example.com"}
	assert.Equal(t, true, UseAuth(), fmt.Sprintf("UseAuth after setting, expected: %v, actual: %v", true, UseAuth()))
	assert.Equal(t, uExpected, AuthServer(), fmt.Sprintf("AuthServer after setting value, expected: %v, actual: %v", uExpected, AuthServer()))
}
