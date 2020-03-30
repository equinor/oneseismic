package server

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/equinor/oneseismic/api/middleware"
)

type Config struct {
	Profiling         bool
	HostAddr          string
	AzureBlobSettings AzureBlobSettings
	LogDBConnStr      string
	OAuth2Option      middleware.OAuth2Option
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
		return nil, fmt.Errorf("invalid AUTHSERVER: %w", err)
	}

	apiSecret, err := verifyAPISecret(m["API_SECRET"])
	if err != nil {
		return nil, err
	}

	profiling, err := strconv.ParseBool(m["PROFILING"])
	if err != nil {
		return nil, fmt.Errorf("could not parse PROFILING: %w", err)
	}

	conf := &Config{
		OAuth2Option: middleware.OAuth2Option{
			AuthServer: authServer,
			APISecret:  []byte(*apiSecret),
			Audience:   m["RESOURCE_ID"],
			Issuer:     m["ISSUER"],
		},
		AzureBlobSettings: azb(m),
		HostAddr:          m["HOST_ADDR"],
		LogDBConnStr:      m["LOGDB_CONNSTR"],
		Profiling:         profiling,
	}

	return conf, nil
}

func verifyAPISecret(sec string) (*string, error) {
	if len(sec) < 8 {
		return nil, fmt.Errorf("invalid API_SECRET: len(%s) == %d < 8", sec, len(sec))
	}

	return &sec, nil
}
