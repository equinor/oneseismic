package server

import (
	"log"
	"net/url"
	"testing"

	"github.com/equinor/oneseismic/api/oneseismic"
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
	app.Use(mockOboJWT())
	Register(app, *storageEndpoint, zmqReqAddr, zmqRepAddr, zmqFailureAddr)

	e := httptest.New(t, app)
	e.GET("/some_guid/slice/0/0").
		Expect().
		Status(httptest.StatusOK).
		ContentType("application/x-protobuf")
}

func coreMock(reqNdpt string, repNdpt string, failureAddr string) {
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
		partial := routedPartialResult {
			address: proc.address,
			partial: partialResult {
				pid: proc.pid,
				n: 0,
				m: 1,
				payload: bytes,
			},
		}

		_, err = partial.sendZMQ(out)

		for err == zmq4.EHOSTUNREACH {
			_, err = out.SendMessage(m)
		}
	}
}
