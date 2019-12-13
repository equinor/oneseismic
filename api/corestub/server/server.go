package server

import (
	"context"
	"encoding/binary"
	"io"

	"fmt"
	"math"
	"net"

	pb "github.com/equinor/seismic-cloud-api/api/proto"
	store "github.com/equinor/seismic-cloud-api/api/service/store"
	"google.golang.org/grpc"
)

type coreServer struct {
	storeAddr string
}

type Surface struct {
	Points []*Point
}

type Point struct {
	X uint64
	Y uint64
	Z uint64
}

var ss store.SurfaceStore

var grpcServer *grpc.Server

func wrapSurfaceReader(in io.Reader) (Surface, error) {
	var surf Surface
	surf, err := decodeSurface(in)
	if err != nil {
		return surf, fmt.Errorf("downloading surface: %w", err)
	}
	return surf, nil
}

func decodeSurface(in io.Reader) (Surface, error) {
	var surface Surface
	for {
		var p struct {
			X, Y, Z uint64
		}
		err := binary.Read(in, binary.LittleEndian, &p)
		if err == io.EOF {
			break
		}
		if err != nil {
			return surface, fmt.Errorf("decoding surface: %w", err)
		}
		surface.Points = append(surface.Points, &Point{X: p.X, Y: p.Y, Z: p.Z})
	}
	return surface, nil
}

func bound(x, y, z uint64) func(p *Point) bool {

	return func(p *Point) bool {
		if p.X < 0 || p.Y < 0 || p.Z < 0 {
			return false
		}
		if p.X > x || p.Y > y || p.Z > z {
			return false
		}
		return true
	}
}

func genFunc(g string, x, y, z uint32) func(p *Point) float32 {

	b := bound(uint64(x), uint64(y), uint64(z))
	switch g {
	case "checker":
		return checkerCube(b)
	case "gradient":
		return gradientCube(b)
	case "sin":
		return sinCube(b)
	default:
		return sinCube(b)
	}
}

func checkerCube(b func(p *Point) bool) func(p *Point) float32 {

	return func(p *Point) float32 {
		if !b(p) {
			return 0.0
		}
		x, y, z := p.X, p.Y, p.Z

		mx, my, mz := x%2 == 0, y%2 == 0, z%2 == 0

		if mx != my {
			if mz {
				return 0.0
			}
			return 1.0

		}

		if mz {
			return 1.0
		}
		return 0.0

	}
}

func gradientCube(b func(p *Point) bool) func(p *Point) float32 {
	return func(p *Point) float32 {
		if !b(p) {
			return 0
		}
		return float32(p.Z)
	}
}

func sinCube(b func(p *Point) bool) func(p *Point) float32 {
	return func(p *Point) float32 {
		if !b(p) {
			return 0.0
		}
		return float32(math.Sin(float64(p.Z)))
	}
}

func g(d0, d1 int32) []float32 {
	v := make([]float32, 0)
	for i := int32(0); i < d0; i++ {
		for j := int32(0); j < d1; j++ {
			v = append(v, float32(d0)+float32(d1))
		}
	}

	return v
}

func (s *coreServer) Slice(ctx context.Context, in *pb.Request) (*pb.Reply, error) {

	fmt.Println("stitching on cube ", in.Geometry.Cubeid)

	repl := &pb.Reply{Dim0: 100, Dim1: 100, V: g(100, 100)}
	return repl, nil
}

func StartServer(ctx context.Context, hostAddr string, surfaceStore store.SurfaceStore) error {

	ss = surfaceStore

	cs := &coreServer{storeAddr: ""}

	fmt.Println("starting server on ", hostAddr)
	lis, err := net.Listen("tcp", hostAddr)
	if err != nil {
		return fmt.Errorf("tcp listen: %w", err)
	}
	var opts []grpc.ServerOption
	grpcServer = grpc.NewServer(opts...)
	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()

	}()

	pb.RegisterCoreServer(grpcServer, cs)
	err = grpcServer.Serve(lis)
	if err != nil {
		return fmt.Errorf("serve grpc server: %w", err)
	}
	return nil
}
