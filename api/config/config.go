package config

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	authServer *url.URL
}

var cfg *Config

func SetDefaults() {
	viper.SetDefault("NO_AUTH", false)
	viper.SetDefault("API_SECRET", "")
	viper.SetDefault("AUTHSERVER", "http://oauth2.example.com")
	viper.SetDefault("ISSUER", "")
	viper.SetDefault("HOST_ADDR", "localhost:8080")
	viper.SetDefault("DOMAIN_LIST", "")
	viper.SetDefault("DOMAIN_MAIL", "")
	viper.SetDefault("STITCH_TCP_ADDR", "")
	viper.SetDefault("STITCH_GRPC_ADDR", "")
	viper.SetDefault("STITCH_CMD", "")
	viper.SetDefault("RESOURCE_ID", "")
	viper.SetDefault("CERT_FILE", "cert.crt")
	viper.SetDefault("KEY_FILE", "cert.key")
	viper.SetDefault("HTTP_ONLY", false)
	viper.SetDefault("TLS", false)
	viper.SetDefault("LETSENCRYPT", false)
	viper.SetDefault("PROFILING", false)
	viper.SetDefault("SWAGGER", false)
	viper.SetDefault("MANIFEST_PATH", "tmp/")
	viper.SetDefault("MANIFEST_DB_URI", "mongodb://")
	viper.SetDefault("AZURE_STORAGE_ACCOUNT", "")
	viper.SetDefault("AZURE_STORAGE_ACCESS_KEY", "")
	viper.SetDefault("AZURE_SURFACE_CONTAINER", "scblob")
	viper.SetDefault("AZURE_MANIFEST_CONTAINER", "scmanifest")
	viper.SetDefault("AZURE_STORAGE_URL", "https://%s.blob.core.windows.net/%s")
	viper.SetDefault("LOCAL_SURFACE_PATH", "")
	viper.SetDefault("LOGDB_CONNSTR", "")
}

func Load() error {
	cfg = new(Config)

	if !viper.GetBool("NO_AUTH") {
		a, err := parseURL(viper.GetString("AUTHSERVER"))
		if err != nil {
			return err
		}
		cfg.authServer = a
	}

	return nil
}

func parseURL(s string) (*url.URL, error) {

	if len(s) == 0 {
		return nil, fmt.Errorf("Url value empty")
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func Version() string {
	return fmt.Sprintf("Seismic Cloud API %s.%s.%s", majVer, minVer, patchVer)
}

func AuthServer() *url.URL {
	if cfg == nil {
		return nil
	}
	return cfg.authServer
}

func UseAuth() bool {
	return !viper.GetBool("NO_AUTH")
}

func HostAddr() string {
	return viper.GetString("HOST_ADDR")
}

func StitchCmd() []string {
	return strings.Split(viper.GetString("STITCH_CMD"), " ")
}

func StitchTCPAddr() string {
	return viper.GetString("STITCH_TCP_ADDR")
}

func StitchGrpcAddr() string {
	return viper.GetString("STITCH_GRPC_ADDR")
}

func AzStorageAccount() string {
	return viper.GetString("AZURE_STORAGE_ACCOUNT")
}

func AzStorageKey() string {
	return viper.GetString("AZURE_STORAGE_ACCESS_KEY")
}

func AzStorageURL() string {
	return viper.GetString("AZURE_STORAGE_URL")
}

func AzSurfaceContainerName() string {
	return viper.GetString("AZURE_SURFACE_CONTAINER")
}

func AzManifestContainerName() string {
	return viper.GetString("AZURE_MANIFEST_CONTAINER")
}

func LocalSurfacePath() string {
	return viper.GetString("LOCAL_SURFACE_PATH")
}

func HTTPOnly() bool {
	return viper.GetBool("HTTP_ONLY")
}

func UseTLS() bool {
	return viper.GetBool("TLS")
}

func UseLetsEncrypt() bool {
	return viper.GetBool("LETSENCRYPT")
}

func DomainList() string {
	return viper.GetString("DOMAIN_LIST")
}

func DomainMail() string {
	return viper.GetString("DOMAIN_MAIL")
}

func CertFile() string {
	return viper.GetString("CERT_FILE")
}

func KeyFile() string {
	return viper.GetString("KEY_FILE")
}
func ResourceID() string {
	return viper.GetString("RESOURCE_ID")
}
func Issuer() string {
	return viper.GetString("ISSUER")
}

func Profiling() bool {
	return viper.GetBool("PROFILING")
}

func Swagger() bool {
	return viper.GetBool("SWAGGER")
}

func ManifestStoragePath() string {
	return viper.GetString("MANIFEST_PATH")
}
func ManifestDbURI() string {
	return viper.GetString("MANIFEST_DB_URI")
}

func LogDBConnStr() string {
	return viper.GetString("LOGDB_CONNSTR")
}

func ApiSecret() string {
	sec := viper.GetString("API_SECRET")

	if len(sec) < 8 {
		b := make([]byte, 20)
		rand.Read(b)
		sec = string(b)
	}
	return sec
}
