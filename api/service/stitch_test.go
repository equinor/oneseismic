package service

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/equinor/seismic-cloud/api/events"
	pb "github.com/equinor/seismic-cloud/api/proto"
	"github.com/equinor/seismic-cloud/api/service/store"
	"google.golang.org/grpc"
	"gotest.tools/assert"
)

type mockGRPCStitch struct {
	grpcAddr string
	opts     []grpc.DialOption
}

func TestDecodeSurface(t *testing.T) {

	var surface pb.Surface
	for x := 0; x < 10; x++ {
		for y := 0; y < 10; y++ {
			surface.Points = append(surface.Points, &pb.Point{X: float32(x), Y: float32(y), Z: float32(1.0)})
		}
	}
	want := surface
	buf := &bytes.Buffer{}
	var err error
	for _, p := range surface.Points {
		err = binary.Write(buf, binary.LittleEndian, p.X)
		if err != nil {
			return
		}
		err = binary.Write(buf, binary.LittleEndian, p.Y)
		if err != nil {
			return
		}
		err = binary.Write(buf, binary.LittleEndian, p.Z)
		if err != nil {
			return
		}
	}

	got, err := decodeSurface(buf)
	if err != nil {
		t.Errorf("Stitch.decode err %v", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Stitch.decode got %v, want %v", got, want)
		return
	}
}

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

func TestNewStitchNilCmd(t *testing.T) {
	_, err := NewStitch(nil, false)
	ExpectOurError(t, err, "Invalid stitch type")
}

func TestNewStitchNoSuchFileCmd(t *testing.T) {
	_, err := NewStitch([]string{"/no/such/file"}, false)
	ExpectOurError(t, err, "Cannot use executable: `/no/such/file`")
}

func TestNewStitchEmptyCmd(t *testing.T) {
	_, err := NewStitch([]string{}, false)
	ExpectOurError(t, err, "No command given")
}

func TestNewStitchDirCmd(t *testing.T) {
	_, err := NewStitch([]string{"/tmp"}, false)
	ExpectOurError(t, err, "Cannot use directory as executable: `/tmp`")
}
