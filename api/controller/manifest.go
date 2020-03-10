package controller

import (
	"context"
	"net/http"

	"github.com/equinor/oneseismic/api/service"
	"github.com/kataras/iris/v12"
)

type ManifestController struct {
	ms service.ManifestStore
}

func NewManifestController(ms service.ManifestStore) *ManifestController {
	return &ManifestController{ms: ms}
}

func (msc *ManifestController) List(ctx iris.Context) {
	bgctx := context.Background()
	manifests, err := msc.ms.List(bgctx)
	if err != nil {
		ctx.StatusCode(http.StatusNotFound)
		return
	}

	ctx.Header("Content-Type", "application/json")
	_, err = ctx.JSON(manifests)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	return
}
