package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfigError(t *testing.T) {
	m := make(map[string]string)
	_, err := ParseConfig(m)
	assert.Error(t, err)
}

func TestConfigMinimum(t *testing.T) {
	m := make(map[string]string)

	m["AUTHSERVER"] = "http://some.host"
	m["API_SECRET"] = "123456789"
	m["PROFILING"] = "false"

	conf, err := ParseConfig(m)
	assert.Nil(t, err)
	assert.Equal(t, conf.profiling, false)
	assert.Equal(t, conf.oAuth2Option.AuthServer.String(), m["AUTHSERVER"])
	assert.Equal(t, conf.oAuth2Option.APISecret, []byte(m["API_SECRET"]))
}

func TestConfigAPI_SECRET(t *testing.T) {
	m := make(map[string]string)

	m["AUTHSERVER"] = "http://some.host"

	_, err := ParseConfig(m)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API_SECRET")
}

func TestConfigAUTHSERVER(t *testing.T) {
	m := make(map[string]string)

	m["API_SECRET"] = "123456789"

	_, err := ParseConfig(m)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AUTHSERVER")
}
