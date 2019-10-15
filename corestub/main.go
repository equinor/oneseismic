package main

import (
	"context"
	"os"

	"fmt"
	"log"
	"math"
	"net"

	pb "github.com/equinor/seismic-cloud/corestub/proto"
	"google.golang.org/grpc"
)

type coreServer struct {
	storeAddr string
}

func genFunc(g string) func(p *pb.Point) float32 {
	switch g {
	case "checker":
		return checkerCube
	case "gradient":
		return gradientCube
	case "sin":
		return sinCube
	default:
		return sinCube
	}
}

func checkerCube(p *pb.Point) float32 {

	x, y, z := int(math.Floor(float64(p.X))), int(math.Floor(float64(p.Y))), int(math.Floor(float64(p.Z)))

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

func gradientCube(p *pb.Point) float32 {

	return p.Z
}

func sinCube(p *pb.Point) float32 {

	return float32(math.Sin(float64(p.Z)))
}

func (s *coreServer) StitchSurface(ctx context.Context, in *pb.SurfaceRequest) (*pb.SurfaceReply, error) {

	fmt.Println("stitching on cube ", in.Basename)
	repl := &pb.SurfaceReply{
		I: make([]uint64, 0),
		V: make([]float32, 0)}
	cube := genFunc(in.Basename)
	fmt.Println("size of surface is ", len(in.Surface.Points))
	for idx, p := range in.Surface.Points {
		repl.I = append(repl.I, uint64(idx))
		repl.V = append(repl.V, cube(p))
	}
	fmt.Println("size of repl is ", len(repl.I))
	return repl, nil
}

func (s *coreServer) ShatterLink(ctx context.Context, in *pb.ShatterLinkRequest) (*pb.ShatterReply, error) {
	return nil, nil
}

func main() {
	cs := &coreServer{storeAddr: ""}

	hostAddr := os.Getenv("SC_GRPC_HOST_ADDR")
	if len(hostAddr) < 1 {
		hostAddr = "localhost:10000"
	}

	fmt.Println("starting server on ", hostAddr)
	lis, err := net.Listen("tcp", hostAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterCoreServer(grpcServer, cs)
	grpcServer.Serve(lis)
}
