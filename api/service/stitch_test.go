package service

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/equinor/seismic-cloud/api/events"
	"gotest.tools/assert"

	pb "github.com/equinor/seismic-cloud/api/proto"
	"github.com/equinor/seismic-cloud/api/service/store"
	"google.golang.org/grpc"
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

func TestNewStitch_nil_cmd(t *testing.T) {
	_, err := NewStitch(nil, false)

	if err != nil {
		if serr, ok := err.(*events.Event); ok {
			assert.Equal(t, "Invalid stitch type", serr.Err.Error())
		} else {
			t.Errorf("Expected error type: *events.Event")
		}
	} else {
		t.Errorf("Expected error")
	}

}
