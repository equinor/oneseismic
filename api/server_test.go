package main

import (
	"net/url"
	"testing"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/equinor/oneseismic/api/server"
	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/pebbe/zmq4"
	"google.golang.org/protobuf/proto"
)

func TestSlicer(t *testing.T) {
	storageEndpoint, _ := url.Parse("http://some.url")
	zmqReqAddr := "inproc://" + uuid.New().String()
	zmqRepAddr := "inproc://" + uuid.New().String()
	zmqFailureAddr := "inproc://" + uuid.New().String()

	go coreMock(zmqReqAddr, zmqRepAddr, zmqFailureAddr)
	app := iris.Default()
	app.Use(func()iris.Handler {
		return func(ctx iris.Context) {
			ctx.Values().Set("jwt", "sometoken")
			ctx.Next()
		}
	}())
	server.Register(app, *storageEndpoint, zmqReqAddr, zmqRepAddr, zmqFailureAddr)

	e := httptest.New(t, app)
	e.GET("/some_guid/slice/0/0").
		Expect().
		Status(httptest.StatusOK)
}

func coreMock(reqNdpt string, repNdpt string, failureAddr string) {
	in, _ := zmq4.NewSocket(zmq4.PULL)
	in.Connect(reqNdpt)

	out, _ := zmq4.NewSocket(zmq4.ROUTER)
	out.SetRouterMandatory(1)
	out.Connect(repNdpt)

	for {
		m, _ := in.RecvMessageBytes(0)
		address := string(m[0])
		pid := string(m[1])
		fr := oneseismic.FetchResponse{Requestid: pid}
		fr.Function = &oneseismic.FetchResponse_Slice{
			Slice: &oneseismic.SliceResponse{
				Tiles: []*oneseismic.SliceTile{
					{
						Layout: &oneseismic.SliceLayout{
							ChunkSize:  1,
							Iterations: 0,
						},
						V: []float32{0.1},
					},
				},
			},
		}

		bytes, _ := proto.Marshal(&fr)

		_, err := out.SendMessage(address, pid, "0/1", bytes)
		for err == zmq4.EHOSTUNREACH {
			_, err = out.SendMessage(m)
		}
	}
}
