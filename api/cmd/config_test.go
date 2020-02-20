package cmd

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoAuth(t *testing.T) {
	reset()
	assert.Equal(t, true, useAuth())
	setDefault("NO_AUTH", true)
	assert.Equal(t, false, useAuth())
}

func TestAuthServer(t *testing.T) {
	reset()
	_, err := authServer()
	assert.Error(t, err)

	anURL := &url.URL{Scheme: "http", Host: "some.host"}
	setDefault("AUTHSERVER", anURL)
	u, err := authServer()
	assert.NoError(t, err)
	assert.Equal(t, anURL, u)
}
