package cmd

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/equinor/oneseismic/api/service"
)

type errInvalidConfig struct {
	Name string
	Err  error
}

func (e *errInvalidConfig) Unwrap() error { return e.Err }
func (e *errInvalidConfig) Error() string { return fmt.Sprintf(e.Name) }

type config struct {
	noAuth            bool
	profiling         bool
	authServer        url.URL
	hostAddr          string
	issuer            string
	stitchGrpcAddr    string
	azureBlobSettings service.AzureBlobSettings
	resourceID        string
	logDBConnStr      string
	apiSecret         string
}

func orDefaultBool(val string, def bool) bool {
	if val, err := strconv.ParseBool(val); err == nil {
		return val
	}

	return def
}

func azb(m map[string]string) service.AzureBlobSettings {
	return service.AzureBlobSettings{
		StorageURL:  m["AZURE_STORAGE_URL"],
		AccountName: m["AZURE_STORAGE_ACCOUNT"],
		AccountKey:  m["AZURE_STORAGE_ACCESS_KEY"],
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
		apiSecret:         *apiSecret,
		authServer:        *authServer,
		azureBlobSettings: azb(m),
		hostAddr:          m["HOST_ADDR"],
		issuer:            m["ISSUER"],
		logDBConnStr:      m["LOGDB_CONNSTR"],
		noAuth:            orDefaultBool(m["NO_AUTH"], false),
		profiling:         orDefaultBool(m["PROFILING"], false),
		resourceID:        m["RESOURCE_ID"],
		stitchGrpcAddr:    m["STITCH_GRPC_ADDR"],
	}

	return conf, nil
}

func verifyAPISecret(sec string) (*string, error) {
	if len(sec) < 8 {
		return nil, &errInvalidConfig{"Invalid API_SECRET", fmt.Errorf("len(%s) == %d < 8", sec, len(sec))}
	}

	return &sec, nil
}
