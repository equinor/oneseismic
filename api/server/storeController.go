package server

import (
	"context"
	"net/http"

	"github.com/kataras/iris/v12"
)

type storeController struct {
	store
}

func (sc *storeController) list(ctx iris.Context) {
	blobs, err := sc.store.list(context.Background())
	if err != nil {
		ctx.StatusCode(http.StatusNotFound)
		return
	}

	ctx.Header("Content-Type", "application/json")
	_, err = ctx.JSON(blobs)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
	}
}
