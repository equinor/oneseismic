package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"

	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/kataras/iris"
)

type SurfaceController struct {
	ss store.SurfaceStore
}

func NewSurfaceController(ss store.SurfaceStore) *SurfaceController {
	return &SurfaceController{ss: ss}
}

// @Description get list of available surfaces
// @Produce  application/json
// @Success 200 {object} controller.fileBytes OK
// @Failure 502 {object} controller.APIError "Internal Server Error"
// @Router /surface/ [get]
func (ssc *SurfaceController) List(ctx iris.Context) {
	op := "surface.list"
	bgctx := context.Background()
	info, err := ssc.ss.List(bgctx)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		l.LogE(op, "Files can't be listed", err)
		return
	}
	if len(info) > 0 {
		ctx.Header("Content-Type", "application/json")
		ctx.JSON(info)
	} else {
		ctx.Header("Content-Type", "text/plain")
		ctx.WriteString("No valid manifests in store")
	}
}

// @Description get surface file
// @Produce  application/octet-stream
// @Param   surfaceID  path    string     true        "File ID"
// @Success 200 {object} controller.fileBytes OK
// @Failure 502 {object} controller.APIError "Internal Server Error"
// @Router /surface/{surface_id} [get]
func (ssc *SurfaceController) Download(ctx iris.Context) {
	op := "surface.download"
	surfaceID := ctx.Params().Get("surfaceID")
	bgctx := context.Background()
	reader, err := ssc.ss.Download(bgctx, surfaceID)
	if err != nil {
		ctx.StatusCode(404)
		l.LogE(op, fmt.Sprintf("Could not download surface: %s", surfaceID), err)
		return
	}

	ctx.Header("Content-Type", "application/octet-stream")

	_, err = io.Copy(ctx.ResponseWriter(), reader)
	if err != nil {

		l.LogE(op, fmt.Sprintf("Error writing to response, surfaceID %s", surfaceID), err)
		return
	}
}

// @Description post surface file
// @Accept  application/octet-stream
// @Produce  application/octet-stream
// @Param   surfaceID  path    string     true        "File ID"
// @Success 200 {object} controller.bloburl OK
// @Failure 500 {object} controller.APIError "Internal Server Error"
// @Router /surface/{surface_id} [post]
func (ssc *SurfaceController) Upload(ctx iris.Context) {
	userID, ok := ctx.Values().Get("userID").(string)

	if !ok || userID == "" {
		userID = "seismic-cloud-api"
	}

	surfaceID := ctx.Params().Get("surfaceID")

	reader := ctx.Request().Body

	bgctx := context.Background()
	blobURL, err := ssc.ss.Upload(bgctx, surfaceID, userID, reader)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		l.LogE("surface.upload", fmt.Sprintf("Could not upload surface: %s", surfaceID), err)
		return
	}
	ctx.JSON(blobURL)
}
