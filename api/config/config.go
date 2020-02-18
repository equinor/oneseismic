package config

import (
	"crypto/rand"
	"fmt"
	l "github.com/equinor/seismic-cloud/api/logger"
	"net/url"
	"os"

	"github.com/equinor/seismic-cloud/api/events"
	"github.com/spf13/viper"
)

func Init(cfgFile string) error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return events.E("Open working dir", err)
		}
		viper.AddConfigPath(wd)
		viper.SetConfigName(".sc-api")
	}
	SetDefaults()
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok && cfgFile == "" {
			return nil
		} else {
			return err
		}
	}
	return nil
}

func SetDefaults() {
	if viper.ConfigFileUsed() == "" {
		l.LogI("Config from environment variables")
	} else {
		l.LogI("Config loaded and validated " + viper.ConfigFileUsed())
	}

	viper.SetDefault("NO_AUTH", false)
	viper.SetDefault("API_SECRET", "")
	viper.SetDefault("AUTHSERVER", "http://oauth2.example.com")
	viper.SetDefault("ISSUER", "")
	viper.SetDefault("HOST_ADDR", "localhost:8080")
	viper.SetDefault("STITCH_GRPC_ADDR", "")
	viper.SetDefault("RESOURCE_ID", "")
	viper.SetDefault("PROFILING", false)
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

func Reset() {
	viper.Reset()
}

func SetDefault(key string, val interface{}) {
	viper.SetDefault(key, val)
}

func AuthServer() (*url.URL, error) {
	s := viper.GetString("AUTHSERVER")
	if len(s) == 0 {
		return nil, fmt.Errorf("Url value empty")
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func UseAuth() bool {
	return !viper.GetBool("NO_AUTH")
}

func HostAddr() string {
	return viper.GetString("HOST_ADDR")
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

func ResourceID() string {
	return viper.GetString("RESOURCE_ID")
}
func Issuer() string {
	return viper.GetString("ISSUER")
}

func Profiling() bool {
	return viper.GetBool("PROFILING")
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
		_, err := rand.Read(b)
		if err != nil {
			panic(err)
		}
		sec = string(b)
	}
	return sec
}
