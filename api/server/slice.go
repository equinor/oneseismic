package server

import (
	"fmt"
	"net/http"

	"github.com/equinor/oneseismic/api/core"
	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
)

type sliceModel interface {
	fetchSlice(
		requestid string,
		guid string,
		dim int32,
		lineno int32) (*core.SliceResponse, error)
}

type sliceMock struct {
	slices map[string]core.SliceResponse
}

func (s *sliceMock) fetchSlice(
	requestid string,
	guid string,
	dim int32,
	lineno int32) (*core.SliceResponse, error) {
	l, ok := s.slices[guid]
	if ok {
		return &l, nil
	}
	return nil, fmt.Errorf("no such slice")
}

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
	requestid := uuid.New().String()

	slice, err := sc.slicer.fetchSlice(requestid, guid, dim, ordinal)
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
