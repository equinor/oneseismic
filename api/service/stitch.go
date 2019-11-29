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
	Stitch(ctx context.Context, ms store.Manifest, out io.Writer, surfaceID string) (string, error)
}

func NewStitch(stype interface{}, profile bool) (Stitcher, error) {
	op := events.Op("service.NewStich")
	switch stitcher := stype.(type) {
	case GrpcOpts:
		addr := stitcher.Addr
		opts := make([]grpc.DialOption, 0)
		if stype.(GrpcOpts).Insecure {
			opts = append(opts, grpc.WithInsecure())
		}
		return &gRPCStitch{addr, opts}, nil
	default:
		return nil, events.E(op, "Invalid stitch type", events.ErrorLevel)
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

func (gs *gRPCStitch) Stitch(ctx context.Context, ms store.Manifest, out io.Writer, surfaceID string) (string, error) {
	req := &pb.SurfaceRequest{
		Surfaceid:  surfaceID,
		Basename:   ms.Basename,
		Cubexs:     ms.Cubexs,
		Cubeys:     ms.Cubeys,
		Cubezs:     ms.Cubezs,
		Fragmentxs: ms.Fragmentxs,
		Fragmentys: ms.Fragmentys,
		Fragmentzs: ms.Fragmentzs,
	}
	conn, err := grpc.Dial(string(gs.grpcAddr), gs.opts...)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	client := pb.NewCoreClient(conn)

	r, err := client.StitchSurface(
		ctx,
		req)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}

	for _, val := range r.SurfaceReplyValues {
		err = binary.Write(buf, binary.LittleEndian, val.I)
		if err != nil {
			return "", err
		}
		err = binary.Write(buf, binary.LittleEndian, val.V)
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
