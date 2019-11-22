package main

import (
	"context"
	"os"

	"fmt"
	"log"
	"math"
	"net"

	pb "github.com/equinor/seismic-cloud-api/corestub/proto"
	"github.com/equinor/seismic-cloud-api/corestub/store"
	"google.golang.org/grpc"
)

type coreServer struct {
	storeAddr string
}

var ss store.SurfaceStore

func bound(x, y, z uint64) func(p *store.Point) bool {

	return func(p *store.Point) bool {
		if p.X < 0 || p.Y < 0 || p.Z < 0 {
			return false
		}
		if p.X > x || p.Y > y || p.Z > z {
			return false
		}
		return true
	}
}

func genFunc(g string, x, y, z uint32) func(p *store.Point) float32 {

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

func checkerCube(b func(p *store.Point) bool) func(p *store.Point) float32 {

	return func(p *store.Point) float32 {
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

func gradientCube(b func(p *store.Point) bool) func(p *store.Point) float32 {
	return func(p *store.Point) float32 {
		if !b(p) {
			return 0
		}
		return float32(p.Z)
	}
}

func sinCube(b func(p *store.Point) bool) func(p *store.Point) float32 {
	return func(p *store.Point) float32 {
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

	surf, err := ss.Download(in.Surfaceid)
	if err != nil {
		fmt.Println("surface download failed")
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

func main() {

	var err error
	ss, err = store.NewSurfaceStore(surfaceStoreConfig())
	if err != nil {
		panic(fmt.Errorf("No surface store, error: %v", err))
	}

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

func surfaceStoreConfig() interface{} {

	if len(os.Getenv("AZURE_STORAGE_ACCOUNT")) > 0 && len(os.Getenv("AZURE_STORAGE_ACCESS_KEY")) > 0 {
		return store.AzureBlobSettings{
			AccountName:   os.Getenv("AZURE_STORAGE_ACCOUNT"),
			AccountKey:    os.Getenv("AZURE_STORAGE_ACCESS_KEY"),
			ContainerName: "scblob",
		}
	}

	if len(os.Getenv("LOCAL_SURFACE_PATH")) > 0 {
		return os.Getenv("LOCAL_SURFACE_PATH")
	}

	return make(map[string][]byte)

}
