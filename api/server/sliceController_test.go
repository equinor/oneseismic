package server

import (
	"fmt"
	"testing"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
)

type sliceMock struct {
	slices map[string]oneseismic.SliceResponse
}

func (s *sliceMock) fetchSlice(guid string,
	dim int32,
	lineno int32,
	requestid string) (*oneseismic.SliceResponse, error) {
	l, ok := s.slices[guid]
	if ok {
		return &l, nil
	}
	return nil, fmt.Errorf("no such slice")
}

func TestExistingSlice(t *testing.T) {
	app := iris.Default()

	m := map[string]oneseismic.SliceResponse{"some_guid": oneseismic.SliceResponse{}}
	sc := &sliceController{&sliceMock{m}}

	app.Get("/{guid:string}/slice/{dim:int32}/{ordinal:int32}", sc.get)

	e := httptest.New(t, app)
	e.GET("/some_guid/slice/0/0").
		Expect().
		Status(httptest.StatusOK).
		JSON()
}

func TestMissingSlice(t *testing.T) {
	app := iris.Default()

	sc := &sliceController{&sliceMock{}}
	app.Get("/{guid:string}/slice/{dim:int32}/{ordinal:int32}", sc.get)

	e := httptest.New(t, app)
	e.GET("/some_other_guid/slice/0/0").
		Expect().
		Status(httptest.StatusNotFound)
}
