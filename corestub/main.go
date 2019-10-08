package main

import (
	"context"

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

func (s *coreServer) StitchSurface(ctx context.Context, in *pb.SurfaceRequest) (*pb.SurfaceReply, error) {

	fmt.Println("stitching on cube ", in.Basename)
	repl := &pb.SurfaceReply{
		I: make([]uint64, 0),
		V: make([]float32, 0)}
	fmt.Println("size of surface is ", len(in.Surface.Points))
	for idx, val := range in.Surface.Points {
		repl.I = append(repl.I, uint64(idx))
		repl.V = append(repl.V, float32(math.Sin(float64(val.Z))))
	}
	fmt.Println("size of repl is ", len(repl.I))
	return repl, nil
}

func (s *coreServer) ShatterLink(ctx context.Context, in *pb.ShatterLinkRequest) (*pb.ShatterReply, error) {
	return nil, nil
}

func main() {
	cs := &coreServer{storeAddr: ""}

	fmt.Println("starting server on localhost:10000")
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", 10000))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterCoreServer(grpcServer, cs)
	grpcServer.Serve(lis)
}
