package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewStitchNil(t *testing.T) {
	_, err := NewStitch(nil)
	assert.Error(t, err)
}

func TestNewStitch(t *testing.T) {
	_, err := NewStitch(GrpcOpts{Addr: "some.addr", Insecure: true})
	assert.NoError(t, err)
}
