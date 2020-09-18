package server

import (
	"testing"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"google.golang.org/protobuf/proto"
)

type sliceMock struct {
	pio procIO
}

func mockSession() procIO {
	return procIO{
		out: make(chan partialResult),
		err: make(chan string),
	}
}

func (s *sliceMock) getProcIO(
	guid string,
	dim int32,
	lineno int32,
	requestid string,
	token string,
) (*procIO, error) {
	return &s.pio, nil
}

func dummyFetchResponse() []byte {
	fr := oneseismic.FetchResponse{Requestid: "pid"}
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
	return bytes
}

func Test1Slice(t *testing.T) {
	app := iris.Default()
	app.Use(mockOboJWT())

	pio := mockSession()
	close(pio.err)

	payload := dummyFetchResponse()

	go func() {
		pio.out <- partialResult{payload: payload, m: 1, n: 0, pid: ""}
		close(pio.out)
	}()

	sc := &sliceController{&sliceMock{pio}}
	app.Get("/{guid:string}/slice/{dim:int32}/{lineno:int32}", sc.get)

	e := httptest.New(t, app)
	e.GET("/some_guid/slice/0/0").
		Expect().
		Status(httptest.StatusOK).
		JSON()
}


func Test2Slices(t *testing.T) {
	app := iris.Default()
	app.Use(mockOboJWT())

	pio := mockSession()
	close(pio.err)

	payload := dummyFetchResponse()

	go func() {
		pio.out <- partialResult{payload: payload, m: 2, n: 0, pid: ""}
		pio.out <- partialResult{payload: payload, m: 2, n: 1, pid: ""}
		close(pio.out)
	}()

	sc := &sliceController{&sliceMock{pio}}
	app.Get("/{guid:string}/slice/{dim:int32}/{lineno:int32}", sc.get)

	e := httptest.New(t, app)
	e.GET("/some_guid/slice/0/0").
		Expect().
		Status(httptest.StatusOK).
		JSON()
}

func Test1MissingSlice(t *testing.T) {
	app := iris.Default()
	app.Use(mockOboJWT())

	pio := mockSession()
	close(pio.err)

	payload := dummyFetchResponse()

	go func() {
		pio.out <- partialResult{payload: payload, m: 2, n: 0, pid: ""}
		close(pio.out)
	}()

	sc := &sliceController{&sliceMock{pio}}
	app.Get("/{guid:string}/slice/{dim:int32}/{lineno:int32}", sc.get)

	e := httptest.New(t, app)
	e.GET("/some_guid/slice/0/0").
		Expect().
		Status(httptest.StatusInternalServerError)
}
