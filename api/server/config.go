package server

import "net/url"

//Config needed to serve oneseismic
type Config struct {
	Profiling    bool
	HostAddr     string
	StorageURL   string
	AccountName  string
	AccountKey   string
	LogDBConnStr string
	LogLevel     string
	AuthServer   *url.URL
	Issuer       string
	APISecret    []byte
	ZmqReqAddr   string
	ZmqRepAddr   string
	SigKeySet    map[string]interface{}
}
