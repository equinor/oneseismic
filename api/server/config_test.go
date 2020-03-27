package server

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfigError(t *testing.T) {
	m := make(map[string]string)
	_, err := ParseConfig(m)
	assert.Error(t, err)

	var e *errInvalidConfig

	assert.True(t, errors.As(err, &e))
}

func TestConfigMinimum(t *testing.T) {
	m := make(map[string]string)

	m["AUTHSERVER"] = "http://some.host"
	m["API_SECRET"] = "123456789"

	conf, err := ParseConfig(m)
	assert.Nil(t, err)
	assert.Equal(t, conf.Profiling, false)
	assert.Equal(t, conf.OAuth2Option.AuthServer.String(), m["AUTHSERVER"])
	assert.Equal(t, conf.OAuth2Option.APISecret, []byte(m["API_SECRET"]))
}

func TestConfigAPI_SECRET(t *testing.T) {
	m := make(map[string]string)

	m["AUTHSERVER"] = "http://some.host"

	_, err := ParseConfig(m)

	var e *errInvalidConfig

	assert.True(t, errors.As(err, &e))
	assert.Contains(t, err.Error(), "API_SECRET")
}

func TestConfigAUTHSERVER(t *testing.T) {
	m := make(map[string]string)

	m["API_SECRET"] = "123456789"

	_, err := ParseConfig(m)

	var e *errInvalidConfig

	assert.True(t, errors.As(err, &e))
	assert.Contains(t, err.Error(), "AUTHSERVER")
}
