package server

import (
	"fmt"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/equinor/oneseismic/api/logger"
	"github.com/google/uuid"
	"github.com/kataras/iris/v12"
	"google.golang.org/protobuf/encoding/protojson"
)

type sliceController struct {
	slicer sliceModel
}

func (sc *sliceController) slice(ctx iris.Context) {
	token, ok := ctx.Values().Get("jwt").(*jwt.Token)
	if !ok {
		logger.LogE("jwt", fmt.Errorf("missing"))
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}
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
	requestid := uuid.New().String()
	slice, err := sc.slicer.fetchSlice(root, guid, dim, lineno, requestid, token.Raw)
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
	storageURL string,
	reqNdpt string,
	repNdpt string,
	mPlexName string,
) sliceController {
	jobs := make(chan job)

	go multiplexer(jobs, mPlexName, reqNdpt, repNdpt)

	return sliceController{slicer: &slicer{storageURL: storageURL, jobs: jobs}}
}
