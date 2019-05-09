package config

import "net/url"

type Config struct {
	AuthServer *url.URL
	StitchPath string
}

var cfg = &Config{}

func Load() (*Config, error) {

	if err := cfg.verify(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (cfg *Config) verify() error {

	return nil
}
