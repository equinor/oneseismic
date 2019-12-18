package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"

	"github.com/equinor/seismic-cloud-api/api/events"
	pb "github.com/equinor/seismic-cloud-api/api/proto"
	"github.com/equinor/seismic-cloud-api/api/service/store"
	"google.golang.org/grpc"
)

type Stitcher interface {
	Stitch(ctx context.Context, out io.Writer, sr StitchParams) (string, error)
}

type StitchParams struct {
	Dim          int32
	Lineno       int64
	CubeManifest *store.Manifest
}

func NewStitch(stype interface{}) (Stitcher, error) {
	switch stitcher := stype.(type) {
	case GrpcOpts:
		addr := stitcher.Addr
		opts := make([]grpc.DialOption, 0)
		if stitcher.Insecure {
			opts = append(opts, grpc.WithInsecure())
		}
		return &gRPCStitch{addr, opts}, nil
	default:
		return nil, events.E("Invalid stitch type", events.ErrorLevel)
	}
}

type gRPCStitch struct {
	grpcAddr string
	opts     []grpc.DialOption
}

type GrpcOpts struct {
	Addr     string
	Insecure bool
}

func (gs *gRPCStitch) Stitch(
	ctx context.Context,
	out io.Writer,
	sp StitchParams) (string, error) {
	req := &pb.Request{
		Dim:      sp.Dim,
		Lineno:   sp.Lineno,
		Geometry: (*pb.Geometry)(sp.CubeManifest),
	}
	conn, err := grpc.Dial(string(gs.grpcAddr), gs.opts...)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	client := pb.NewCoreClient(conn)

	r, err := client.Slice(
		ctx,
		req)
	if err != nil {
		return "", events.E("GRPC Stitch", err)
	}
	buf := &bytes.Buffer{}

	err = binary.Write(buf, binary.LittleEndian, r.GetDim0())
	if err != nil {
		return "", events.E("Write Dim0", err)
	}
	err = binary.Write(buf, binary.LittleEndian, r.GetDim1())
	if err != nil {
		return "", events.E("Write Dim1", err)
	}

	for _, val := range r.GetV() {

		err = binary.Write(buf, binary.LittleEndian, val)
		if err != nil {
			return "", err
		}
	}
	_, err = io.Copy(out, buf)
	if err != nil {
		return "", err
	}

	return "", nil
}
