package server

import (
	"fmt"
	"net/http"

	"github.com/equinor/oneseismic/api/core"
	"github.com/golang/protobuf/proto"
	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
	"github.com/zeromq/goczmq"
)

type sliceController struct {
	slicer sliceModel
}

func (sc *sliceController) get(ctx iris.Context) {
	guid := ctx.Params().GetString("guid")
	if len(guid) == 0 {
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	dim, err := ctx.Params().GetInt32("dim")
	if err != nil {
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	ordinal, err := ctx.Params().GetInt32("ordinal")
	if err != nil {
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	requestid := uuid.New()

	slice, err := sc.slicer.fetchSlice(guid, requestid.String(), dim, ordinal)
	if err != nil {
		ctx.StatusCode(http.StatusNotFound)
		return
	}

	ctx.Header("Content-Type", "application/json")
	_, err = ctx.JSON(slice)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	return
}

type sliceModel interface {
	fetchSlice(
		guid string,
		requestid string,
		dim int32,
		lineno int32) (*core.SliceResponse, error)
}

type zmqDealer struct {
	dealer *goczmq.Channeler
	root   string
}

func (sm *zmqDealer) fetchSlice(guid string, requestid string, dim int32, lineno int32) (*core.SliceResponse, error) {
	ar := core.ApiRequest{
		Guid:      guid,
		Root:      sm.root,
		Requestid: requestid,
		Shape: &core.FragmentShape{
			Dim0: 64,
			Dim1: 64,
			Dim2: 64,
		},
		Function: &core.ApiRequest_Slice{
			Slice: &core.ApiSlice{
				Dim:    dim,
				Lineno: lineno,
			},
		},
	}

	fetchResponse, err := sm.fetch(&ar)
	if err != nil {
		return nil, fmt.Errorf("could not fetch FetchResponse: %w", err)
	}

	slice := fetchResponse.GetSlice()
	if slice == nil {
		return nil, fmt.Errorf("no slice in FetchResponse")
	}

	return slice, nil
}

func newZMQDealer(addr string, root string) zmqDealer {
	dealer := goczmq.NewDealerChanneler(addr)
	return zmqDealer{dealer, root}
}

func (sm *zmqDealer) fetch(ar *core.ApiRequest) (*core.FetchResponse, error) {
	msg, err := proto.Marshal(ar)
	if err != nil {
		return nil, fmt.Errorf("could not marshal ApiRequest: %w", err)
	}

	response := sm.zmqReqRep(msg)

	fetchResponse := core.FetchResponse{}
	err = proto.Unmarshal(response[0], &fetchResponse)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal FetchResponse: %w", err)
	}

	return &fetchResponse, nil
}

func (sm *zmqDealer) zmqReqRep(msg []byte) [][]byte {
	sm.dealer.SendChan <- [][]byte{msg}
	response := <-sm.dealer.RecvChan

	return response
}
