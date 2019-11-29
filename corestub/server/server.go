package server

import (
	"context"
	"encoding/binary"
	"io"

	"fmt"
	"log"
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

func (s *coreServer) StitchSurface(ctx context.Context, in *pb.SurfaceRequest) (*pb.SurfaceReply, error) {

	fmt.Println("stitching on cube ", in.Basename)
	vals := []*pb.SurfaceReplyValue{}
	cube := genFunc(in.Basename, in.Cubexs, in.Cubeys, in.Cubezs)
	id := in.Surfaceid
	r, err := ss.Download(ctx, id)
	if err != nil {
		fmt.Println("surface download failed")
		return nil, err
	}

	surf, err := wrapSurfaceReader(r)
	if err != nil {
		return nil, err
	}

	fmt.Println("size of surface is ", len(surf.Points))
	for idx, p := range surf.Points {
		val := &pb.SurfaceReplyValue{I: uint64(idx), V: cube(p)}
		vals = append(vals, val)
	}
	fmt.Println("size of repl is ", len(vals))
	repl := &pb.SurfaceReply{SurfaceReplyValues: vals}
	return repl, nil
}

func (s *coreServer) ShatterLink(ctx context.Context, in *pb.ShatterLinkRequest) (*pb.ShatterReply, error) {
	return nil, nil
}

func StartServer(hostAddr string, surfaceStore store.SurfaceStore) {

	ss = surfaceStore

	cs := &coreServer{storeAddr: ""}

	fmt.Println("starting server on ", hostAddr)
	lis, err := net.Listen("tcp", hostAddr)
	if err != nil {
		log.Fatalf("tcp listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer = grpc.NewServer(opts...)
	pb.RegisterCoreServer(grpcServer, cs)
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalf("serve grpc server: %v", err)
	}
}

func StopServer() {
	if grpcServer != nil {
		grpcServer.GracefulStop()
	}
}
