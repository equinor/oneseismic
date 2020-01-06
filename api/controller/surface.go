package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"

	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/kataras/iris/v12"
)

type SurfaceController struct {
	ss store.SurfaceStore
}

func NewSurfaceController(ss store.SurfaceStore) *SurfaceController {
	return &SurfaceController{ss: ss}
}

// @Description get list of available surfaces
// @ID list_surfaces
// @Produce  application/json
// @Success 200 {array} store.SurfaceMeta "list of all surfaces"
// @Failure 500 {object}  controller.APIError "Internal Server Error"
// @security ApiKeyAuth
// @tags surface
// @Router /surface/ [get]
func (ssc *SurfaceController) List(ctx iris.Context) {
	bgctx := context.Background()
	info, err := ssc.ss.List(bgctx)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		l.LogE("Files can't be listed", err)
		return
	}
	if len(info) > 0 {
		ctx.Header("Content-Type", "application/json")
		_, err = ctx.JSON(info)
		if err != nil {
			ctx.StatusCode(http.StatusInternalServerError)
			l.LogE("JSON Encoding surfaces", err)
			return

		}
	}
}

// @Description get surface file
// @ID download_surface
// @Produce  application/octet-stream
// @Param   surfaceID  path    string     true        "File ID"
// @Success 200 {string} string "byte stream"
// @Failure 404 {object} controller.APIError "Not found"
// @Failure 502 {object} controller.APIError "Internal Server Error"
// @security ApiKeyAuth
// @tags surface
// @Router /surface/{surfaceID} [get]
func (ssc *SurfaceController) Download(ctx iris.Context) {
	surfaceID := ctx.Params().Get("surfaceID")
	bgctx := context.Background()
	reader, err := ssc.ss.Download(bgctx, surfaceID)
	if err != nil {
		ctx.StatusCode(404)
		l.LogE(fmt.Sprintf("Could not download surface: %s", surfaceID), err)
		return
	}

	ctx.Header("Content-Type", "application/octet-stream")

	_, err = io.Copy(ctx.ResponseWriter(), reader)
	if err != nil {
		ctx.StatusCode(404)
		l.LogE(fmt.Sprintf("Error writing to response, surfaceID %s", surfaceID), err)
		return
	}
}

// @Description post surface file
// @ID upload_surface
// @Accept  application/octet-stream
// @Produce  application/octet-stream
// @Param   surfaceID  path    string     true        "File ID"
// @Success 200 {string} string "byte stream"
// @Failure 500 {object} controller.APIError "Internal Server Error"
// @security ApiKeyAuth
// @tag surface
// @Router /surface/{surfaceID} [post]
func (ssc *SurfaceController) Upload(ctx iris.Context) {
	userID, ok := ctx.Values().Get("userID").(string)
	if !ok || userID == "" {
		userID = "seismic-cloud"
	}

	surfaceID := ctx.Params().Get("surfaceID")

	reader := ctx.Request().Body

	bgctx := context.Background()
	blobURL, err := ssc.ss.Upload(bgctx, surfaceID, userID, reader)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		l.LogE(fmt.Sprintf("Could not upload surface: %s", surfaceID), err)
		return
	}
	_, err = ctx.JSON(blobURL)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		l.LogE("JSON Encoding surfaces", err)
		return

	}
}
