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
	token, ok := ctx.Values().Get("jwt").(string)
	if !ok {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	cubes, err := sc.store.list(context.Background(), token)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	_, err = ctx.JSON(cubes)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
	}
}

func (sc *storeController) services(ctx iris.Context) {
	token, ok := ctx.Values().Get("jwt").(string)
	if !ok {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	guid := ctx.Params().GetString("guid")
	if len(guid) == 0 {
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	_, err := sc.store.manifest(context.Background(), guid, token)
	if err != nil {
		ctx.StatusCode(http.StatusNotFound)
		return
	}

	_, err = ctx.JSON([]string{"slice"})
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
	}
}

func (sc *storeController) dimensions(ctx iris.Context) {
	token, ok := ctx.Values().Get("jwt").(string)
	if !ok {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	guid := ctx.Params().GetString("guid")
	if len(guid) == 0 {
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	dimensions, err := sc.store.dimensions(context.Background(), guid, token)
	if err != nil {
		ctx.StatusCode(http.StatusNotFound)
		return
	}

	_, err = ctx.JSON(dimensions)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
	}
}

func (sc *storeController) lines(ctx iris.Context) {
	token, ok := ctx.Values().Get("jwt").(string)
	if !ok {
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	guid := ctx.Params().GetString("guid")
	if len(guid) == 0 {
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	dimension, err := ctx.Params().GetInt32("dimension")
	if err != nil {
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	lines, err := sc.store.lines(context.Background(), guid, dimension, token)
	if err != nil {
		ctx.StatusCode(http.StatusNotFound)
		return
	}

	_, err = ctx.JSON(lines)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
	}
}
