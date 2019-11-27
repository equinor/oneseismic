package service

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/equinor/seismic-cloud-api/api/events"
	"github.com/equinor/seismic-cloud-api/api/service/store"
	"gotest.tools/assert"
)

func TestEncodeManifest(t *testing.T) {
	want := store.Manifest{
		Basename:   "testmanifest",
		Cubexs:     1,
		Cubeys:     1,
		Cubezs:     1,
		Fragmentxs: 1,
		Fragmentys: 1,
		Fragmentzs: 1,
	}
	buf, err := manifestReader(want)
	if err != nil {
		t.Errorf("Stitch.encode manifestReader %v", err)
		return
	}
	dump := make([]byte, 2)
	err = binary.Read(buf, binary.LittleEndian, &dump)
	if err != nil {
		t.Errorf("Stitch.encode binary.Read err %v", err)
		return
	}
	var manilen uint32
	err = binary.Read(buf, binary.LittleEndian, &manilen)
	if err != nil {
		t.Errorf("Stitch.encode binary.Read err %v", err)
		return
	}
	var m store.Manifest
	err = json.Unmarshal(buf.Bytes(), &m)
	if err != nil {
		t.Errorf("Stitch.encode json.Unmarshal err %v", err)
	}

	if !reflect.DeepEqual(m, want) {
		t.Errorf("Stitch.encode got %v, want %v", m, want)
		return
	}
}

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
