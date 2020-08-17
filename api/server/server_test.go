package server

import (
	"log"
	"math/rand"
	"net/url"
	"os"
	"testing"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/pebbe/zmq4"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"github.com/stretchr/testify/assert"

)


func TestSlicerLarge(t *testing.T) {
	if os.Getenv("TESTING") != "LARGE" {
		t.Skip()
	}		
	storageEndpoint, _ := url.Parse("http://some.url")
	account := ""
	zmqReqAddr := "inproc://" + uuid.New().String()
	zmqRepAddr := "inproc://" + uuid.New().String()
	zmqFailureAddr := "inproc://" + uuid.New().String()

	var tiles []*oneseismic.SliceTile
	r := rand.New(rand.NewSource(99))
	for i := 0; i < 30; i++ {
		v := make([]float32, 2500)
		for i := range v {
			v[i] = r.Float32()
		}
		tile := []*oneseismic.SliceTile{
			{
				Layout: &oneseismic.SliceLayout{
					ChunkSize:  1,
					Iterations: 0,
				},
				V: v,
			},
		}
		tiles = append(tiles, tile...)
	}

	slice := &oneseismic.SliceResponse{
		Tiles: tiles,
	}

	go coreMock(zmqReqAddr, zmqRepAddr, zmqFailureAddr, slice, 100)
	app := iris.Default()
	app.Use(mockOboJWT())
	Register(app, *storageEndpoint, account, zmqReqAddr, zmqRepAddr, zmqFailureAddr)

	e := httptest.New(t, app)

	for i := 0; i < 1000000000; i++ {
		e.GET("/some_guid/slice/0/0").
			Expect().
			Status(httptest.StatusOK)

		// m := protojson.UnmarshalOptions{DiscardUnknown: true}
		// sss := oneseismic.SliceResponse{}
		// err := m.Unmarshal([]byte(resp.Body().Raw()), &sss)
		// assert.Nil(t, err)
		// for k, v := range sss.Tiles {
		// 	assert.EqualValues(t, v.V, slice.Tiles[k].V)
		// }
		log.Printf("%v", i)
	}
}

func TestSlicer(t *testing.T) {
	storageEndpoint, _ := url.Parse("http://some.url")
	account := ""
	zmqReqAddr := "inproc://" + uuid.New().String()
	zmqRepAddr := "inproc://" + uuid.New().String()
	zmqFailureAddr := "inproc://" + uuid.New().String()

	slice := &oneseismic.SliceResponse {
			Tiles: []*oneseismic.SliceTile {
			{
				Layout: &oneseismic.SliceLayout {
					ChunkSize: 1,
					Iterations: 0,
				},
				V: []float32{0.1},
			},
		},
	}
	go coreMock(zmqReqAddr, zmqRepAddr, zmqFailureAddr, slice, 1)

	app := iris.Default()
	app.Use(mockOboJWT())
	Register(app, *storageEndpoint, account, zmqReqAddr, zmqRepAddr, zmqFailureAddr)

	e := httptest.New(t, app)

	resp := e.GET("/some_guid/slice/0/0").
		Expect().
		Status(httptest.StatusOK)

	m := protojson.UnmarshalOptions{DiscardUnknown: true}
	sr := oneseismic.SliceResponse{}
	err := m.Unmarshal([]byte(resp.Body().Raw()), &sr)
	assert.Nil(t, err)
	for k, v := range sr.Tiles {
		assert.Equal(t, v.V, slice.Tiles[k].V)
		assert.Equal(t, v.Layout.ChunkSize, slice.Tiles[k].Layout.ChunkSize)
		assert.Equal(t, v.Layout.Iterations, slice.Tiles[k].Layout.Iterations)
	}
}

func coreMock(
	reqNdpt string,
	repNdpt string,
	failureAddr string,
	slice *oneseismic.SliceResponse,
	numPartials int,
) {
	in, _ := zmq4.NewSocket(zmq4.PULL)
	in.Connect(reqNdpt)

	out, _ := zmq4.NewSocket(zmq4.ROUTER)
	out.SetRouterMandatory(1)
	out.Connect(repNdpt)

	for {
		m, _ := in.RecvMessageBytes(0)
		proc := process{}
		err := proc.loadZMQ(m)
		if err != nil {
			msg := "Broken process (loadZMQ) in core emulation: %s"
			log.Fatalf(msg, err.Error())
		}
		fr := oneseismic.FetchResponse{Requestid: proc.pid}
		fr.Function = &oneseismic.FetchResponse_Slice{
			Slice: slice,
		}

		bytes, _ := proto.Marshal(&fr)
		for i := 0; i < numPartials; i++ {
			partial := routedPartialResult {
				address: proc.address,
				partial: partialResult {
					pid: proc.pid,
					n: i,
					m: numPartials,
					payload: bytes,
				},
			}

			_, err = partial.sendZMQ(out)

			for err == zmq4.EHOSTUNREACH {
				_, err = out.SendMessage(m)
			}
		}
	}
}
