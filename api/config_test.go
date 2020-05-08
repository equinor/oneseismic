package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfigError(t *testing.T) {
	m := make(map[string]string)
	_, err := parseConfig(m)
	assert.Error(t, err)
}

func TestConfigMinimum(t *testing.T) {
	m := make(map[string]string)

	m["AUTHSERVER"] = "http://some.host"
	m["API_SECRET"] = "123456789"
	m["PROFILING"] = "false"

	conf, err := parseConfig(m)
	assert.Nil(t, err)
	assert.Equal(t, conf.profiling, false)
	assert.Equal(t, conf.AuthServer.String(), m["AUTHSERVER"])
	assert.Equal(t, conf.APISecret, []byte(m["API_SECRET"]))
}

func TestConfigAPI_SECRET(t *testing.T) {
	m := make(map[string]string)

	m["AUTHSERVER"] = "http://some.host"

	_, err := parseConfig(m)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API_SECRET")
}

func TestConfigAUTHSERVER(t *testing.T) {
	m := make(map[string]string)

	m["API_SECRET"] = "123456789"

	_, err := parseConfig(m)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AUTHSERVER")
}

func TestConfigAZURE_STORAGE_URL(t *testing.T) {
	m := make(map[string]string)

	m["API_SECRET"] = "123456789"
	m["AUTHSERVER"] = "http://some.host"
	m["PROFILING"] = "false"

	m["AZURE_STORAGE_URL"] = "http://{}.some.host"

	c, err := parseConfig(m)

	assert.NoError(t, err)
	assert.Equal(t, c.azureBlobSettings.StorageURL, "http://%s.some.host")
}
