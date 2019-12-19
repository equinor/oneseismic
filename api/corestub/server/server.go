package server

import (
	"context"

	"fmt"
	"net"

	pb "github.com/equinor/seismic-cloud-api/api/proto"
	"google.golang.org/grpc"
)

type coreServer struct {
	storeAddr string
}

var grpcServer *grpc.Server

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

func StartServer(ctx context.Context, hostAddr string) error {

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
