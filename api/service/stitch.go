package service

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"

	l "github.com/equinor/seismic-cloud/api/logger"
	pb "github.com/equinor/seismic-cloud/api/proto"
	"github.com/equinor/seismic-cloud/api/service/store"
	"google.golang.org/grpc"
)

type Stitcher interface {
	Stitch(ctx context.Context, ms store.Manifest, out io.Writer, in io.Reader) (string, error)
}

func NewStitch(stype interface{}, profile bool) (Stitcher, error) {
	switch stype.(type) {
	case []string:
		cmdArgs := stype.([]string)
		execName := cmdArgs[0]
		inf, err := os.Stat(execName)
		if err != nil {
			return nil, err
		}
		if inf.IsDir() {
			return nil, fmt.Errorf("Cannot be a directory")
		}
		return &execStitch{cmdArgs, profile}, nil
	case TcpAddr:
		addr := stype.(TcpAddr)
		return &tCPStitch{TcpAddr(string(addr))}, nil
	case GrpcOpts:
		addr := stype.(GrpcOpts).Addr
		opts := make([]grpc.DialOption, 0)
		if stype.(GrpcOpts).Insecure {
			opts = append(opts, grpc.WithInsecure())
		}
		return &gRPCStitch{addr, opts}, nil
	default:
		return nil, fmt.Errorf("Invalid stitch type")
	}
}

type execStitch struct {
	cmd     []string
	profile bool
}

func (es *execStitch) Stitch(ctx context.Context, ms store.Manifest, out io.Writer, in io.Reader) (string, error) {
	op := "execStitch.stitch"
	var e *exec.Cmd
	var pBuf *bytes.Buffer

	if es.profile {

		pr, pw, err := NewNamedPipe()
		if err != nil {
			return "", fmt.Errorf("Error opening profiling pipe: %v", err)
		}
		defer pw.Close()

		b := append(es.cmd[1:], "--profile", pr.Name())
		e = exec.Command(es.cmd[0], b...)
		pBuf = bytes.NewBuffer(make([]byte, 0))
		go func() {
			defer pr.Close()
			defer os.Remove(pr.Name())

			if _, err := io.Copy(pBuf, pr); err != nil {
				l.LogE(op, "Profiling stitch failed", err)
			}
		}()
	} else {
		e = exec.Command(es.cmd[0], es.cmd[1:]...)
	}
	errBuf := bytes.NewBuffer(make([]byte, 0))
	e.Stdout = out
	e.Stdin = in
	e.Stderr = errBuf

	err := e.Run()
	if err != nil {
		return "", fmt.Errorf("Execute stitch failed: %v - %v ", err, string(errBuf.Bytes()))
	}
	e.Wait()

	if es.profile {
		return pBuf.String(), nil
	}

	return "", nil

}

type gRPCStitch struct {
	grpcAddr string
	opts     []grpc.DialOption
}
type GrpcOpts struct {
	Addr     string
	Insecure bool
}

func decodeSurface(in io.Reader) (pb.Surface, error) {
	var surface pb.Surface
	for {
		var p struct {
			X, Y, Z float32
		}
		err := binary.Read(in, binary.LittleEndian, &p)
		if err == io.EOF {
			break
		}
		if err != nil {
			return surface, err
		}
		surface.Points = append(surface.Points, &pb.Point{X: p.X, Y: p.Y, Z: p.Z})
	}
	return surface, nil
}

func (gs *gRPCStitch) Stitch(ctx context.Context, ms store.Manifest, out io.Writer, in io.Reader) (string, error) {
	surface, err := decodeSurface(in)
	if err != nil {
		return "", err
	}
	req := &pb.SurfaceRequest{
		Surface:    &surface,
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

	for i := range r.I {
		err = binary.Write(buf, binary.LittleEndian, r.I[i])
		if err != nil {
			return "", err
		}
		err = binary.Write(buf, binary.LittleEndian, r.V[i])
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

type tCPStitch struct {
	tcpAddr TcpAddr
}

type TcpAddr string

func (ts *tCPStitch) Stitch(ctx context.Context, ms store.Manifest, out io.Writer, in io.Reader) (string, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", string(ts.tcpAddr))
	if err != nil {
		return "", err
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	rxErrChan := make(chan string)
	txErrChan := make(chan string)
	go func() {
		if _, err := io.Copy(conn, in); err != nil {
			txErrChan <- fmt.Sprintf("Sending to tcp stitch failed: %v", err)
		} else {
			txErrChan <- ""
		}
		conn.CloseWrite()

	}()
	go func() {
		if _, err := io.Copy(out, conn); err != nil {
			rxErrChan <- fmt.Sprintf("Receiving from tcp stitch failed: %v", err)
		} else {
			rxErrChan <- ""
		}
		conn.CloseRead()

	}()
	errs := make([]string, 0)
	for i := 0; i < 2; i++ {
		select {
		case e := <-rxErrChan:
			if len(e) > 0 {
				errs = append(errs, e)
			}
		case e := <-txErrChan:
			if len(e) > 0 {
				errs = append(errs, e)

			}
		}
	}
	if len(errs) > 0 {
		return "", fmt.Errorf("%s", strings.Join(errs, "\n"))
	}

	return "", nil

}
