package server

import (
	"fmt"
	"testing"

	"github.com/equinor/oneseismic/api/core"
	"github.com/golang/protobuf/proto"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/stretchr/testify/assert"
	"github.com/zeromq/goczmq"
)

type mockExistingSlicer struct {
	slicesResponse core.SliceResponse
}

func (mbs *mockExistingSlicer) fetchSlice(
	guid string,
	requestid string,
	dim int32,
	lineno int32) (*core.SliceResponse, error) {
	return &mbs.slicesResponse, nil
}

func TestExistingSliceEndpoint(t *testing.T) {
	sr := core.SliceResponse{}
	hs := HTTPServer{
		app:    iris.Default(),
		slicer: &sliceController{&mockExistingSlicer{sr}},
	}
	Configure(&hs)

	e := httptest.New(t, hs.app)
	e.GET("/some_guid/slice/0/0").
		Expect().
		Status(httptest.StatusOK).
		JSON().Equal(sr)
}

type mockMissingSlice struct{}

func (mbs *mockMissingSlice) fetchSlice(
	guid string,
	requestid string,
	dim int32,
	lineno int32) (*core.SliceResponse, error) {
	return nil, fmt.Errorf("missing slice")
}

func TestMissingSliceEndpoint(t *testing.T) {
	hs := HTTPServer{
		app:    iris.Default(),
		slicer: &sliceController{&mockMissingSlice{}},
	}
	Configure(&hs)

	e := httptest.New(t, hs.app)
	e.GET("/some_guid/slice/0/0").
		Expect().
		Status(httptest.StatusNotFound)
}

func responseWithSlice() *core.FetchResponse {
	return &core.FetchResponse{
		Function: &core.FetchResponse_Slice{
			Slice: &core.SliceResponse{
				Tiles: []*core.SliceTile{
					&core.SliceTile{
						Id: &core.FragmentId{Dim0: 0, Dim1: 0, Dim2: 0},
						V:  []float32{1.0},
					},
				},
			},
		},
	}
}

func mockSlice(router *goczmq.Channeler, fetchResponse *core.FetchResponse) {
	for {
		request := <-router.RecvChan
		if len(request) == 0 {
			break
		}
		bytes, _ := proto.Marshal(fetchResponse)

		router.SendChan <- [][]byte{request[0], bytes}
	}
}

func TestSliceResponse(t *testing.T) {
	addr := "inproc://TestSliceResponse"

	router := goczmq.NewRouterChanneler(addr)
	defer router.Destroy()

	dealer := newZMQDealer(addr, "root")
	defer dealer.dealer.Destroy()

	go mockSlice(router, responseWithSlice())

	slice, err := dealer.fetchSlice("guid", "requestid", 0, 0)
	assert.NoError(t, err)
	assert.NotNil(t, slice)
}

func TestNotSliceResponse(t *testing.T) {
	addr := "inproc://TestNotSliceResponse"

	router := goczmq.NewRouterChanneler(addr)
	defer router.Destroy()

	dealer := goczmq.NewDealerChanneler(addr)
	defer dealer.Destroy()

	go mockSlice(router, &core.FetchResponse{})

	sm := zmqDealer{dealer: dealer}
	_, err := sm.fetchSlice("guid", "requestid", 0, 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "slice")
}
