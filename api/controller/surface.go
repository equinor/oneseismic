package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/equinor/seismic-cloud/api/errors"
	"github.com/equinor/seismic-cloud/api/service"
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
// @Success 200 files Ok
// @Failure 502 {object} controller.APIError "Internal Server Error"
// @Router /surface/ [get]
func (ssc *SurfaceController) List(ctx iris.Context) {
	op := errors.Op("surface.list")
	bgctx := context.Background()
	info, err := ssc.ss.List(bgctx)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		service.Log(errors.E(op, "Files can't be listed", errors.ErrorLevel, err))
		return
	}
	ctx.JSON(info)
}

// @Description get surface file
// @Produce  application/octet-stream
// @Param   surfaceID  path    string     true        "File ID"
// @Success 200 file Ok
// @Failure 502 {object} controller.APIError "Internal Server Error"
// @Router /surface/{surfaceID} [get]
func (ssc *SurfaceController) Download(ctx iris.Context) {
	op := errors.Op("surface.download")
	surfaceID := ctx.Params().Get("surfaceID")
	bgctx := context.Background()
	reader, err := ssc.ss.Download(bgctx, surfaceID)
	if err != nil {
		ctx.StatusCode(404)
		service.Log(errors.E(op, fmt.Sprintf("Could not download surface: %s", surfaceID), errors.WarnLevel, err))
		return
	}

	ctx.Header("Content-Type", "application/octet-stream")

	_, err = io.Copy(ctx.ResponseWriter(), reader)
	if err != nil {

		service.Log(errors.E(op, fmt.Sprintf("Error writing to response, surfaceID %s", surfaceID), errors.ErrorLevel, err))
		return
	}
}

// @Description post surface file
// @Accept  application/octet-stream
// @Produce  application/octet-stream
// @Param   surfaceID  path    string     true        "File ID"
// @Success 200 {file} bloburl	Ok
// @Failure 500 {object} controller.APIError "Internal Server Error"
// @Router /surface/{surfaceID} [post]
func (ssc *SurfaceController) Upload(ctx iris.Context) {
	op := errors.Op("surface.upload")
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
		service.Log(errors.E(op, fmt.Sprintf("Could not upload surface: %s", surfaceID), errors.ErrorLevel, err))
		return
	}
	ctx.JSON(blobURL)
}