package tests

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	pb "github.com/equinor/seismic-cloud-api/api/proto"
	server "github.com/equinor/seismic-cloud-api/api/server"
	"github.com/equinor/seismic-cloud-api/api/service"
	"github.com/equinor/seismic-cloud-api/api/service/store"
	cserver "github.com/equinor/seismic-cloud-api/corestub/server"
	"github.com/stretchr/testify/assert"
)

type Surface struct {
	Points []*Point
}

type Point struct {
	X uint64
	Y uint64
	Z uint64
}

const apiurl = "localhost:8080"
const csurl = "localhost:10000"

const path = "./"

func TestMain(m *testing.M) {
	ms, ss, st, err := setupTestData()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		err := cserver.StartServer(ctx, csurl, ss)
		if err != nil {
			fmt.Println(err)
		}
		return
	}()
	defer cancel()

	opts := []server.HTTPServerOption{
		server.WithManifestStore(ms),
		server.WithSurfaceStore(ss),
		server.WithStitcher(st),
		server.WithHostAddr(apiurl),
		server.WithHTTPOnly()}
	s, err := server.NewHTTPServer(opts...)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		err = s.Serve()
		if err != nil {
			fmt.Println(err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	exitVal := m.Run()

	_ = os.Remove(path + "surf")
	os.Exit(exitVal)
}

func setupTestData() (store.ManifestStore, store.SurfaceStore, service.Stitcher, error) {
	manifest := &store.Manifest{
		Basename:   "checker",
		Cubexs:     2,
		Cubeys:     2,
		Cubezs:     2,
		Fragmentxs: 2,
		Fragmentys: 2,
		Fragmentzs: 2,
	}
	bm, err := json.Marshal(manifest)
	if err != nil {
		return nil, nil, nil, err
	}
	ms, _ := store.NewManifestStore(map[string][]byte{
		"mani": bm})

	points := []*Point{&Point{X: 1, Y: 2, Z: 1}, &Point{X: 2, Y: 1, Z: 2}}
	surface := &Surface{
		Points: points,
	}

	err = writeSurf(*surface)
	if err != nil {
		return nil, nil, nil, err
	}

	ss, err := store.NewSurfaceStore(store.BasePath(path))
	if err != nil {
		return nil, nil, nil, err
	}
	st, err := service.NewStitch(service.GrpcOpts{Addr: csurl, Insecure: true}, false)
	if err != nil {
		return nil, nil, nil, err
	}
	return ms, ss, st, nil
}

func writeSurf(surf Surface) error {
	bufSurf := &bytes.Buffer{}

	for _, val := range surf.Points {
		err := binary.Write(bufSurf, binary.LittleEndian, val.X)
		if err != nil {
			fmt.Println(err)
		}
		err = binary.Write(bufSurf, binary.LittleEndian, val.Y)
		if err != nil {
			fmt.Println(err)
		}
		err = binary.Write(bufSurf, binary.LittleEndian, val.Z)
		if err != nil {
			fmt.Println(err)
		}
	}

	fs, err := os.Create(path + "surf")
	if err != nil {
		return err
	}
	defer func() {
		if err := fs.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	if _, err := fs.Write(bufSurf.Bytes()); err != nil {
		return err
	}

	return nil
}

func TestStitchSucceed(t *testing.T) {

	repls := []*pb.SurfaceReplyValue{&pb.SurfaceReplyValue{I: uint64(0), V: float32(1)}, &pb.SurfaceReplyValue{I: uint64(1), V: float32(0)}}

	buf := &bytes.Buffer{}

	for _, val := range repls {
		err := binary.Write(buf, binary.LittleEndian, val.I)
		assert.NoError(t, err)
		err = binary.Write(buf, binary.LittleEndian, val.V)
		assert.NoError(t, err)
	}

	want := buf.Bytes()

	resp, err := http.Get("http://" + apiurl + "/stitch/mani/surf")
	assert.NoError(t, err)

	defer resp.Body.Close()

	got, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)

	assert.Equal(t, want, got)
}

func TestStitchNoSurface(t *testing.T) {

	resp, err := http.Get("http://" + apiurl + "/stitch/mani/notexist")
	assert.NoError(t, err)

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestStitchNoManifest(t *testing.T) {

	resp, err := http.Get("http://" + apiurl + "/stitch/notexist/surf")
	assert.NoError(t, err)

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
