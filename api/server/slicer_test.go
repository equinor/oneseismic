package server

import (
	"testing"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"google.golang.org/protobuf/encoding/protojson"
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

func (s *sliceMock) fetchSlice(
	guid string,
	dim int32,
	lineno int32,
	requestid string,
	token string,
) (*procIO, error) {
	return &s.pio, nil
}

func TestExistingSlice(t *testing.T) {
	app := iris.Default()
	app.Use(mockOboJWT())

	pio := mockSession()
	close(pio.err)
	sc := &sliceController{&sliceMock{pio}}
	sr := oneseismic.SliceResponse{}
	m := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true}
	js, _ := m.Marshal(&sr)
	p := partialResult{payload: js}

	go func() {
		pio.out <- p
		pio.out <- p
		close(pio.out)
	}()

	app.Get("/{guid:string}/slice/{dim:int32}/{lineno:int32}", sc.get)

	e := httptest.New(t, app)
	e.GET("/some_guid/slice/0/0").
		Expect().
		Status(httptest.StatusOK)
}
