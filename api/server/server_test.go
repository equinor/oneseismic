package server

import (
	"log"
	"testing"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/pebbe/zmq4"
	"google.golang.org/protobuf/proto"
)

func TestSlicer(t *testing.T) {
	app := iris.Default()

	reqNdpt := "inproc://" + uuid.New().String()
	repNdpt := "inproc://" + uuid.New().String()
	go coreMock(reqNdpt, repNdpt)

	mPlexName := uuid.New().String()
	registerSlicer(app, reqNdpt, repNdpt, mPlexName)

	e := httptest.New(t, app)
	jsonResponse := e.GET("/some_root/some_existing_guid/slice/0/0").
		Expect().
		Status(httptest.StatusOK).
		JSON()
	jsonResponse.Path("$.tiles[0].layout.chunk_size").Number().Equal(1)
	jsonResponse.Path("$.tiles[0].v").Array().Elements(0.1)
}

func coreMock(reqNdpt string, repNdpt string) {
	in, _ := zmq4.NewSocket(zmq4.PULL)
	in.Connect(reqNdpt)

	out, _ := zmq4.NewSocket(zmq4.ROUTER)
	out.SetRouterMandatory(1)
	out.Connect(repNdpt)

	for {
		m, _ := in.RecvMessage(0)
		fr := oneseismic.FetchResponse{Requestid: m[1]}
		req := oneseismic.ApiRequest{}
		err := proto.Unmarshal([]byte(m[2]), &req)
		if err != nil {
			log.Fatalln("Failed to decode request:", err)
		}
		if req.Guid == "some_existing_guid" {
			fr.Function = &oneseismic.FetchResponse_Slice{
				Slice: &oneseismic.SliceResponse{
					Tiles: []*oneseismic.SliceTile{
						&oneseismic.SliceTile{
							Layout: &oneseismic.SliceLayout{
								ChunkSize:  1,
								Iterations: 0,
							},
							V: []float32{0.1},
						},
					},
				},
			}
		}
		bytes, err := proto.Marshal(&fr)
		if err != nil {
			log.Fatalln("Failed to encode:", err)
		}
		m[2] = string(bytes)
		_, err = out.SendMessage(m)

		for err == zmq4.EHOSTUNREACH {
			_, err = out.SendMessage(m)
		}
	}
}
