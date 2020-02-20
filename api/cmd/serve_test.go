package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServeDefaultError(t *testing.T) {
	m := make(map[string]string)
	err := Serve(m)
	assert.Error(t, err)
}
