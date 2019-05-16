package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	authServer          *url.URL
	insecure            bool
	stitchCmd           []string
	manifestStoragePath string
	hostAddr            string
	domainList          string
	domainMail          string
	certFile            string
	keyFile             string
	httpOnly            bool
	useLetsEncrypt      bool
	useTLS              bool
}

var cfg *Config

func Load() error {
	cfg = new(Config)

	cfg.insecure = viper.GetBool("NO_AUTH")
	if !cfg.insecure {
		a, err := parseURL(viper.GetString("AUTHSERVER"))
		if err != nil {
			return err
		}
		cfg.authServer = a
	}

	cfg.stitchCmd = strings.Split(viper.GetString("STITCH_CMD"), " ")
	cfg.manifestStoragePath = viper.GetString("MANIFEST_PATH")
	cfg.hostAddr = viper.GetString("HOST_ADDR")
	if len(cfg.hostAddr) == 0 {
		cfg.hostAddr = ":8080"
	}
	cfg.domainList = viper.GetString("DOMAIN_LIST")
	cfg.domainMail = viper.GetString("DOMAIN_MAIL")
	cfg.certFile = viper.GetString("CERT_FILE")
	cfg.keyFile = viper.GetString("KEY_FILE")

	cfg.httpOnly = viper.GetBool("HTTP_ONLY")
	cfg.useTLS = viper.GetBool("TLS")
	cfg.useLetsEncrypt = viper.GetBool("LETSENCRYPT")
	if err := cfg.verify(); err != nil {
		return err
	}

	return nil
}

func (cfg *Config) verify() error {

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

func AuthServer() *url.URL {

	return cfg.authServer
}

func UseAuth() bool {

	return !cfg.insecure
}

func HostAddr() string {
	return cfg.hostAddr
}

func StitchCmd() []string {
	return cfg.stitchCmd
}

func ManifestStoragePath() string {
	return cfg.manifestStoragePath
}

func HttpOnly() bool {
	return cfg.httpOnly
}

func UseTLS() bool {
	return cfg.useTLS
}

func UseLetsEncrypt() bool {
	return cfg.useLetsEncrypt
}

func DomainList() string {
	return cfg.domainList
}

func DomainMail() string {
	return cfg.domainMail
}

func CertFile() string {
	return cfg.certFile
}

func KeyFile() string {
	return cfg.keyFile
}
