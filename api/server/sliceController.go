package server

import (
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/kataras/golog"
	"github.com/kataras/iris/v12"
	"google.golang.org/protobuf/encoding/protojson"
)

type sliceController struct {
	slicer sliceModel
}

func (sc *sliceController) slice(ctx iris.Context) {
	token, ok := ctx.Values().Get("jwt").(*jwt.Token)
	if !ok {
		golog.Errorf("jwt missing from context")
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}
	root := ctx.Params().GetString("root")
	if len(root) == 0 {
		golog.Errorf("root missing from request")
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	guid := ctx.Params().GetString("guid")
	if len(guid) == 0 {
		golog.Errorf("guid missing from request")
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	dim, err := ctx.Params().GetInt32("dim")
	if err != nil {
		golog.Errorf("dim missing from request")
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	lineno, err := ctx.Params().GetInt32("lineno")
	if err != nil {
		golog.Errorf("lineno missing from request")
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	requestid := ""
	slice, err := sc.slicer.fetchSlice(root, guid, dim, lineno, requestid, token.Raw)
	if err != nil {
		golog.Warnf("could not fetch slice: %w", err)
		ctx.StatusCode(http.StatusNotFound)
		return
	}

	ctx.Header("Content-Type", "application/json")
	m := protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true}
	js, err := m.Marshal(slice)
	if err != nil {
		golog.Errorf("could not marshal fetch slice to json: %w", err)
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
	storageURL string,
	reqNdpt string,
	repNdpt string,
	mPlexName string,
) sliceController {
	jobs := make(chan job)

	go multiplexer(jobs, mPlexName, reqNdpt, repNdpt)

	return sliceController{slicer: &slicer{storageURL: storageURL, jobs: jobs}}
}
