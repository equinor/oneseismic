package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfigError(t *testing.T) {
	m := make(map[string]string)
	_, err := parseConfig(m)
	assert.Error(t, err)
	var e *errInvalidConfig
	assert.True(t, errors.As(err, &e))
}

func TestConfigMinimum(t *testing.T) {
	m := make(map[string]string)

	m["AUTHSERVER"] = "http://some.host"
	m["API_SECRET"] = "123456789"

	conf, err := parseConfig(m)
	assert.Nil(t, err)
	assert.Equal(t, conf.noAuth, false)
	assert.Equal(t, conf.profiling, false)
	assert.Equal(t, conf.authServer.String(), m["AUTHSERVER"])
	assert.Equal(t, conf.apiSecret, m["API_SECRET"])

}

func TestConfigAPI_SECRET(t *testing.T) {
	m := make(map[string]string)

	m["AUTHSERVER"] = "http://some.host"

	_, err := parseConfig(m)
	var e *errInvalidConfig
	assert.True(t, errors.As(err, &e))
	assert.Contains(t, err.Error(), "API_SECRET")

}

func TestConfigAUTHSERVER(t *testing.T) {
	m := make(map[string]string)

	m["API_SECRET"] = "123456789"

	_, err := parseConfig(m)
	var e *errInvalidConfig
	assert.True(t, errors.As(err, &e))
	assert.Contains(t, err.Error(), "AUTHSERVER")

}
