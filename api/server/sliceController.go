package server

import (
	"net/http"

	"github.com/equinor/oneseismic/api/oneseismic"
	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
	"google.golang.org/protobuf/encoding/protojson"
)

type sliceModel interface {
	fetchSlice(
		auth string,
		guid string,
		dim int32,
		lineno int32,
		requestid string) (*oneseismic.SliceResponse, error)
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
	lineno, err := ctx.Params().GetInt32("lineno")
	if err != nil {
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	requestid := uuid.New().String()
	auth := ctx.GetHeader("Authorization")
	slice, err := sc.slicer.fetchSlice(auth, guid, dim, lineno, requestid)
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
	storageEndpoint string,
	root string,
) sliceController {
	sessions := newSessions()
	go sessions.Run(reqNdpt, repNdpt)
	sc := sliceController{slicer: &slicer{mm: &mMultiplexer{storageEndpoint, root}, sessions: sessions}}
	return sc
}
