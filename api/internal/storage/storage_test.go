package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCancelledAzStorageGetErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	azstorage := NewAzStorage()

	_, err := azstorage.Get(ctx, "https://example.com")
	if err == nil {
		t.Errorf("expected Get() to fail; err was nil")
	}

	msg := "context canceled"
	assert.Containsf(t, err.Error(), msg, "want err =~ '%s'; was %v", msg, err)
}
