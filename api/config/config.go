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
	hostAddr := viper.GetString("HOST_ADDR")
	if len(hostAddr) == 0 {
		hostAddr = ":8080"
	}
	cfg.hostAddr = hostAddr

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
