package cmd

import (
	"fmt"
	"log"

	"github.com/equinor/oneseismic/api/core"
	"github.com/golang/protobuf/proto"
	"github.com/zeromq/goczmq"
)

func ttt(msg []byte) ([]byte, error) {
	ar := core.ApiRequest{}
	err := proto.Unmarshal(msg, &ar)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal ApiRequest: %w", err)
	}

	fr := core.FetchResponse{
		Requestid: ar.Requestid,
		Function: &core.FetchResponse_Slice{
			Slice: &core.SliceResponse{
				Tiles: []*core.SliceTile{
					&core.SliceTile{
						Id: &core.FragmentId{Dim0: 0, Dim1: 0, Dim2: 0},
						V:  []float32{1.0},
					},
				},
				Layout: &core.SliceLayout{
					ChunkSize:   1,
					InitialSkip: 0,
					Iterations:  1,
					Substride:   0,
					Superstride: 0,
				},
			},
		},
	}

	buf, _ := proto.Marshal(&fr)

	return buf, nil
}

func serveZMQ(addr string) {
	router, err := goczmq.NewRouter(addr)
	defer router.Destroy()
	if err != nil {
		log.Fatalf("could not create router: %s", err)
	}

	for {
		request, err := router.RecvMessage()
		if len(request) == 0 {
			break
		}
		if err != nil {
			log.Fatalf("receive error: %s", err)
		}
		if len(request) < 2 {
			log.Fatalf("not enough messages: %s", err)
		}
		r, err := ttt(request[1])
		if err != nil {
			log.Fatalf("generate response: %s", err)
		}

		err = router.SendFrame(request[0], goczmq.FlagMore)
		if err != nil {
			log.Fatalf("recipient send error: %s", err)
		}

		err = router.SendFrame(r, goczmq.FlagNone)
		if err != nil {
			log.Fatalf("data send error: %s", err)
		}
	}
}
