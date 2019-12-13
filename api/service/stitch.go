package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
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
	CubeManifest *store.Manifest
}

func NewStitch(stype interface{}, profile bool) (Stitcher, error) {
	switch stitcher := stype.(type) {
	case GrpcOpts:
		addr := stitcher.Addr
		opts := make([]grpc.DialOption, 0)
		if stype.(GrpcOpts).Insecure {
			opts = append(opts, grpc.WithInsecure())
		}
		return &gRPCStitch{addr, opts}, nil
	default:
		return nil, events.E("Invalid stitch type", events.ErrorLevel)
	}
}

func manifestReader(ms store.Manifest) (*bytes.Buffer, error) {

	b, err := json.Marshal(ms)
	if err != nil {
		return nil, err
	}
	nb := []byte("M:\x00\x00\x00\x00")
	binary.LittleEndian.PutUint32(nb[2:], uint32(len(b)))
	nb = append(nb, b...)
	return bytes.NewBuffer(nb), nil
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
