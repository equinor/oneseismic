package server

import "crypto/rsa"

//Config needed to serve api
type Config struct {
	StorageURL string
	Issuer     string
	ZmqReqAddr string
	ZmqRepAddr string
	RSAKeys    map[string]rsa.PublicKey
}
