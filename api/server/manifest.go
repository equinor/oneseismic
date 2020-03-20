package server

import (
	"context"
	"net/http"

	"github.com/kataras/iris/v12"
)

type manifestStore interface {
	list(ctx context.Context) ([]string, error)
}

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
