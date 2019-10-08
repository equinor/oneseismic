package service

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"

	pb "github.com/equinor/seismic-cloud/api/proto"
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
		t.Errorf("Stitch.decode Readall err %v", err)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Stitch.decode got %v, want %v", got, want)
		return
	}
}
