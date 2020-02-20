package cmd

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/equinor/seismic-cloud/api/service/store"
)

type errInvalidConfig struct {
	Name string
	Err  error
}

func (e *errInvalidConfig) Unwrap() error { return e.Err }
func (e *errInvalidConfig) Error() string { return fmt.Sprintf(e.Name) }

type config struct {
	noAuth                  bool
	profiling               bool
	authServer              url.URL
	hostAddr                string
	issuer                  string
	stitchGrpcAddr          string
	azureBlobSettings       store.AzureBlobSettings
	azManifestContainerName string
	resourceID              string
	localSurfacePath        string
	manifestStoragePath     string
	manifestDbURI           string
	logDBConnStr            string
	apiSecret               string
}

func orDefaultBool(val string, def bool) bool {
	if val, err := strconv.ParseBool(val); err == nil {
		return val
	}

	return def
}

func azb(m map[string]string) store.AzureBlobSettings {
	return store.AzureBlobSettings{
		StorageURL:    m["AZURE_STORAGE_URL"],
		AccountName:   m["AZURE_STORAGE_ACCOUNT"],
		AccountKey:    m["AZURE_STORAGE_ACCESS_KEY"],
		ContainerName: m["AZURE_SURFACE_CONTAINER"],
	}
}

func parseConfig(m map[string]string) (*config, error) {
	authServer, err := url.ParseRequestURI(m["AUTHSERVER"])
	if err != nil {
		return nil, &errInvalidConfig{Name: "Invalid AUTHSERVER", Err: err}
	}

	apiSecret, err := verifyAPISecret(m["API_SECRET"])
	if err != nil {
		return nil, err
	}

	conf := &config{
		apiSecret:               *apiSecret,
		authServer:              *authServer,
		azManifestContainerName: m["AZURE_MANIFEST_CONTAINER"],
		azureBlobSettings:       azb(m),
		hostAddr:                m["HOST_ADDR"],
		issuer:                  m["ISSUER"],
		localSurfacePath:        m["LOCAL_SURFACE_PATH"],
		logDBConnStr:            m["LOGDB_CONNSTR"],
		manifestDbURI:           m["MANIFEST_DB_URI"],
		manifestStoragePath:     m["MANIFEST_PATH"],
		noAuth:                  orDefaultBool(m["NO_AUTH"], false),
		profiling:               orDefaultBool(m["PROFILING"], false),
		resourceID:              m["RESOURCE_ID"],
		stitchGrpcAddr:          m["STITCH_GRPC_ADDR"],
	}

	return conf, nil
}

func verifyAPISecret(sec string) (*string, error) {
	if len(sec) < 8 {
		return nil, &errInvalidConfig{"Invalid API_SECRET", fmt.Errorf("len(%s) == %d < 8", sec, len(sec))}
	}

	return &sec, nil
}
