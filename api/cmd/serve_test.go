package cmd

import (
	"testing"

	"github.com/equinor/seismic-cloud-api/api/config"
	"github.com/stretchr/testify/assert"
)

func Test_HTTPServerOptionsNeedsConfig(t *testing.T) {
	_, err := createHTTPServerOptions()
	assert.Error(t, err)
}

func Test_createHTTPServerOptionsDefaults(t *testing.T) {
	config.SetDefaults()
	_, err := createHTTPServerOptions()
	assert.Error(t, err)
}
