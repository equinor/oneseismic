package server

import (
	"net/http"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/kataras/iris/v12"
	"google.golang.org/protobuf/encoding/protojson"
)

type sliceModel interface {
	fetchSlice(
		root string,
		guid string,
		dim int32,
		lineno int32,
		requestid string) (*oneseismic.SliceResponse, error)
}

type sliceController struct {
	slicer sliceModel
}

func (sc *sliceController) get(ctx iris.Context) {
	root := ctx.Params().GetString("root")
	if len(root) == 0 {
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
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
	lineno, err := ctx.Params().GetInt32("lineno")
	if err != nil {
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	requestid := ""
	slice, err := sc.slicer.fetchSlice(root, guid, dim, lineno, requestid)
	if err != nil {
		ctx.StatusCode(http.StatusNotFound)
		return
	}

	ctx.Header("Content-Type", "application/json")
	m := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true}
	js, err := m.Marshal(slice)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}
	_, err = ctx.Write(js)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	return
}

func createSliceController(
	reqNdpt string,
	repNdpt string,
	mPlexName string,
) sliceController {
	jobs := make(chan job)
	go multiplexer(jobs, mPlexName, reqNdpt, repNdpt)
	return sliceController{slicer: &slicer{mm: &mMultiplexer{jobs: jobs}}}
}
