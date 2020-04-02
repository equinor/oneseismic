package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kataras/iris/v12"
)

type manifestController struct {
	ms manifestStore
}

func (msc *manifestController) list(ctx iris.Context) {
	bgctx := context.Background()
	manifests, err := msc.ms.list(bgctx)
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

func createManifestController(a AzureBlobSettings) (*manifestController, error) {
	sURL, err := newServiceURL(a)
	if err != nil {
		return nil, fmt.Errorf("creating ServiceURL: %w", err)
	}

	mc := manifestController{ms: sURL}
	return &mc, nil
}
