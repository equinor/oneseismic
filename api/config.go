package main

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type config struct {
	profiling    bool
	hostAddr     string
	storageURL   string
	accountName  string
	accountKey   string
	logDBConnStr string
	authServer   *url.URL
	audience     string
	issuer       string
	apiSecret    []byte
	zmqReqAddr   string
	zmqRepAddr   string
}

func getConfig() (*config, error) {
	authServer, err := url.ParseRequestURI(os.Getenv("AUTHSERVER"))
	if err != nil {
		return nil, fmt.Errorf("invalid AUTHSERVER: %w", err)
	}

	profiling, err := strconv.ParseBool(os.Getenv("PROFILING"))
	if err != nil {
		return nil, fmt.Errorf("could not parse PROFILING: %w", err)
	}

	conf := &config{
		authServer:   authServer,
		apiSecret:    []byte(os.Getenv("API_SECRET")),
		audience:     os.Getenv("RESOURCE_ID"),
		issuer:       os.Getenv("ISSUER"),
		storageURL:   strings.ReplaceAll(os.Getenv("AZURE_STORAGE_URL"), "{}", "%s"),
		accountName:  os.Getenv("AZURE_STORAGE_ACCOUNT"),
		accountKey:   os.Getenv("AZURE_STORAGE_ACCESS_KEY"),
		hostAddr:     os.Getenv("HOST_ADDR"),
		logDBConnStr: os.Getenv("LOGDB_CONNSTR"),
		profiling:    profiling,
		zmqRepAddr:   os.Getenv("ZMQ_REP_ADDR"),
		zmqReqAddr:   os.Getenv("ZMQ_REQ_ADDR"),
	}

	return conf, nil
}
