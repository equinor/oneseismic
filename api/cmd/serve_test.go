package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_HTTPServerOptionsNeedsConfig(t *testing.T) {
	_, err := createHTTPServerOptions()
	assert.Error(t, err)
}

func Test_createHTTPServerOptionsDefaults(t *testing.T) {
	SetDefaults()
	_, err := createHTTPServerOptions()
	assert.Error(t, err)
}
