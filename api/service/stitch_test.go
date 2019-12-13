package service

import (
	"testing"

	"github.com/equinor/seismic-cloud-api/api/events"
	"gotest.tools/assert"
)

func ExpectOurError(t *testing.T, err error, expectedMsg string) {
	if err != nil {
		if ourErr, ok := err.(*events.Event); ok {
			assert.Equal(t, expectedMsg, ourErr.Message)
		} else {
			t.Errorf("Expected our error type: *events.Event, got: %s", err)
		}
	} else {
		t.Errorf("Expected some error")
	}
}

func TestNewStitchNil(t *testing.T) {
	_, err := NewStitch(nil, false)
	ExpectOurError(t, err, "Invalid stitch type")
}
