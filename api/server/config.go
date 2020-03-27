package server

import (
	"fmt"
	"net/url"
	"strconv"
)

type errInvalidConfig struct {
	Name string
	Err  error
}

func (e *errInvalidConfig) Unwrap() error { return e.Err }
func (e *errInvalidConfig) Error() string { return fmt.Sprintf(e.Name) }

type Config struct {
	Profiling         bool
	HostAddr          string
	StitchGrpcAddr    string
	AzureBlobSettings AzureBlobSettings
	LogDBConnStr      string
	OAuth2Option      OAuth2Option
}

func orDefaultBool(val string, def bool) bool {
	if val, err := strconv.ParseBool(val); err == nil {
		return val
	}

	return def
}

func azb(m map[string]string) AzureBlobSettings {
	return AzureBlobSettings{
		StorageURL:  m["AZURE_STORAGE_URL"],
		AccountName: m["AZURE_STORAGE_ACCOUNT"],
		AccountKey:  m["AZURE_STORAGE_ACCESS_KEY"],
	}
}

func ParseConfig(m map[string]string) (*Config, error) {
	authServer, err := url.ParseRequestURI(m["AUTHSERVER"])
	if err != nil {
		return nil, &errInvalidConfig{Name: "Invalid AUTHSERVER", Err: err}
	}

	apiSecret, err := verifyAPISecret(m["API_SECRET"])
	if err != nil {
		return nil, err
	}

	conf := &Config{
		OAuth2Option: OAuth2Option{
			AuthServer: authServer,
			APISecret:  []byte(*apiSecret),
			Audience:   m["RESOURCE_ID"],
			Issuer:     m["ISSUER"],
		},
		AzureBlobSettings: azb(m),
		HostAddr:          m["HOST_ADDR"],
		LogDBConnStr:      m["LOGDB_CONNSTR"],
		Profiling:         orDefaultBool(m["PROFILING"], false),
		StitchGrpcAddr:    m["STITCH_GRPC_ADDR"],
	}

	return conf, nil
}

func verifyAPISecret(sec string) (*string, error) {
	if len(sec) < 8 {
		return nil, &errInvalidConfig{"Invalid API_SECRET", fmt.Errorf("len(%s) == %d < 8", sec, len(sec))}
	}

	return &sec, nil
}
