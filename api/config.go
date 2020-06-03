package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/equinor/oneseismic/api/server"
)

type config struct {
	profiling         bool
	hostAddr          string
	azureBlobSettings server.AzureBlobSettings
	logDBConnStr      string
	AuthServer        *url.URL
	Audience          string
	Issuer            string
	APISecret         []byte
	zmqReqAddr        string
	zmqRepAddr        string
}

func azb(m map[string]string) server.AzureBlobSettings {
	return server.AzureBlobSettings{
		StorageURL:  strings.ReplaceAll(m["AZURE_STORAGE_URL"], "{}", "%s"),
		AccountName: m["AZURE_STORAGE_ACCOUNT"],
		AccountKey:  m["AZURE_STORAGE_ACCESS_KEY"],
	}
}

func getEnvs() map[string]string {
	m := make(map[string]string)

	envs := [...]string{
		"API_SECRET",
		"AUTHSERVER",
		"AZURE_STORAGE_URL",
		"AZURE_STORAGE_ACCOUNT",
		"AZURE_STORAGE_ACCESS_KEY",
		"HOST_ADDR",
		"ISSUER",
		"LOGDB_CONNSTR",
		"PROFILING",
		"RESOURCE_ID",
		"ZMQ_REQ_ADDR",
		"ZMQ_REP_ADDR",
	}

	for _, env := range envs {
		m[env] = os.Getenv(env)
	}

	return m
}

func parseConfig(m map[string]string) (*config, error) {
	authServer, err := url.ParseRequestURI(m["AUTHSERVER"])
	if err != nil {
		return nil, fmt.Errorf("invalid AUTHSERVER: %w", err)
	}

	profiling, err := strconv.ParseBool(m["PROFILING"])
	if err != nil {
		return nil, fmt.Errorf("could not parse PROFILING: %w", err)
	}

	conf := &config{
		AuthServer:        authServer,
		APISecret:         []byte(m["API_SECRET"]),
		Audience:          m["RESOURCE_ID"],
		Issuer:            m["ISSUER"],
		azureBlobSettings: azb(m),
		hostAddr:          m["HOST_ADDR"],
		logDBConnStr:      m["LOGDB_CONNSTR"],
		profiling:         profiling,
		zmqRepAddr:        m["ZMQ_REP_ADDR"],
		zmqReqAddr:        m["ZMQ_REQ_ADDR"],
	}

	return conf, nil
}
