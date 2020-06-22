package server

import (
	"context"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/kataras/golog"
	"github.com/kataras/iris/v12"
)

type storeController struct {
	store
}

// @Summary list of cubes
// @Description get list of available cubes
// @ID list_cubes
// @Produce  application/json
// @Success 200 {array} string
// @Failure 500 {string} string
// @security ApiKeyAuth
// @Router / [get]
func (sc *storeController) list(ctx iris.Context) {
	token, ok := ctx.Values().Get("jwt").(*jwt.Token)
	if !ok {
		golog.Error("missing jwt")
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}
	root := ctx.Params().GetString("root")
	if len(root) == 0 {
		golog.Errorf("root missing from request")
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	containers, err := sc.store.list(context.Background(), root, token.Raw)
	if err != nil {
		golog.Warnf("could not fetch slice: %w", err)
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}
	_, err = ctx.JSON(containers)
	if err != nil {
		golog.Warnf("could not make json of container list: %w", err)
		ctx.StatusCode(http.StatusInternalServerError)
	}
}

// @Summary services for cube
// @Description list services available on cube
// @ID cube
// @Produce application/json
// @Param guid path string true "guid"
// @Success 200 {array} string
// @Failure 400 {string} string
// @Failure 404 {string} string
// @Failure 500 {string} string
// @security ApiKeyAuth
// @Router /{guid} [get]
func (sc *storeController) services(ctx iris.Context) {
	token, ok := ctx.Values().Get("jwt").(*jwt.Token)
	if !ok {
		golog.Error("missing jwt")
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}

	root := ctx.Params().GetString("root")
	if len(root) == 0 {
		golog.Error("root missing from request")
		ctx.StatusCode(http.StatusBadRequest)
		return
	}

	guid := ctx.Params().GetString("guid")
	if len(guid) == 0 {
		golog.Error("guid missing from request")
		ctx.StatusCode(http.StatusBadRequest)
		return
	}

	_, err := sc.store.manifest(context.Background(), root, guid, token.Raw)
	if err != nil {
		golog.Errorf("manifest not found: %w", err)
		ctx.StatusCode(http.StatusNotFound)
		return
	}

	_, err = ctx.JSON([]string{"slice"})
	if err != nil {
		golog.Warnf("could not make json of service list: %w", err)
		ctx.StatusCode(http.StatusInternalServerError)
	}
}

// @Description list of dimensions in cube
// @Summary show dimensions in slice
// @ID dimensions
// @Produce application/json
// @Param guid path string true "guid"
// @Success 200 {string} string
// @Failure 400 {string} string
// @Failure 404 {string} string
// @Failure 500 {string} string
// @security ApiKeyAuth
// @Router /{guid}/slice [get]
func (sc *storeController) dimensions(ctx iris.Context) {
	token, ok := ctx.Values().Get("jwt").(*jwt.Token)
	if !ok {
		golog.Error("missing jwt")
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}
	root := ctx.Params().GetString("root")
	if len(root) == 0 {
		golog.Error("root missing from request")
		ctx.StatusCode(http.StatusBadRequest)
		return
	}

	guid := ctx.Params().GetString("guid")
	if len(guid) == 0 {
		golog.Error("guid missing from request")
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	dimensions, err := sc.store.dimensions(context.Background(), root, guid, token.Raw)
	if err != nil {
		golog.Error(err)
		ctx.StatusCode(http.StatusNotFound)
		return
	}

	_, err = ctx.JSON(dimensions)
	if err != nil {
		golog.Error(err)
		ctx.StatusCode(http.StatusInternalServerError)
	}
}

// @Description list lines in dimension
// @Summary show lines of a dimension in slice
// @ID lines
// @Produce application/json
// @Param guid path string true "guid"
// @Param dimension path int32 true "dimension"
// @Success 200 {string} string
// @Failure 400 {string} string
// @Failure 404 {string} string
// @Failure 500 {string} string
// @security ApiKeyAuth
// @Router /{guid}/slice/{dimension} [get]
func (sc *storeController) lines(ctx iris.Context) {
	token, ok := ctx.Values().Get("jwt").(*jwt.Token)
	if !ok {
		golog.Error("missing jwt")
		ctx.StatusCode(http.StatusInternalServerError)
		return
	}
	root := ctx.Params().GetString("root")
	if len(root) == 0 {
		golog.Error("root missing from request")
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	guid := ctx.Params().GetString("guid")
	if len(guid) == 0 {
		golog.Error("guid missing from request")
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	dimension, err := ctx.Params().GetInt32("dim")
	if err != nil {
		golog.Error("dim missing from request")
		ctx.StatusCode(http.StatusBadRequest)
		return
	}
	lines, err := sc.store.lines(context.Background(), root, guid, dimension, token.Raw)
	if err != nil {
		golog.Error(err)
		ctx.StatusCode(http.StatusNotFound)
		return
	}

	_, err = ctx.JSON(lines)
	if err != nil {
		golog.Error(err)
		ctx.StatusCode(http.StatusInternalServerError)
	}
}
