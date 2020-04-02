package server

import (
	"testing"

	"github.com/equinor/oneseismic/api/core"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
	"github.com/stretchr/testify/assert"
)

func TestSliceModelError(t *testing.T) {
	sc := sliceController{&sliceMock{}}
	_, err := sc.slicer.fetchSlice("requestid", "guid", 0, 0)
	assert.NotNil(t, err)
}

func TestSliceModel(t *testing.T) {
	m := map[string]core.SliceResponse{"guid": core.SliceResponse{}}

	sc := sliceController{&sliceMock{m}}
	slice, err := sc.slicer.fetchSlice("requestid", "guid", 0, 0)

	assert.Nil(t, err)
	assert.NotNil(t, slice)
}

func TestExistingSliceEndpoint(t *testing.T) {
	app := iris.Default()

	m := map[string]core.SliceResponse{"some_guid": core.SliceResponse{}}
	slicer := &sliceController{&sliceMock{m}}

	app.Get("/{guid:string}/{dim:int32}/{ordinal:int32}", slicer.get)

	e := httptest.New(t, app)
	e.GET("/some_guid/0/0").
		Expect().
		Status(httptest.StatusOK).
		JSON()
}

func TestMissingSliceEndpoint(t *testing.T) {
	app := iris.Default()

	slicer := &sliceController{&sliceMock{}}
	app.Get("/{guid:string}/{dim:int32}/{ordinal:int32}", slicer.get)

	e := httptest.New(t, app)
	e.GET("/some_other_guid/0/0").
		Expect().
		Status(httptest.StatusNotFound)
}
