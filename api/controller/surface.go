package controller

import (
	"context"
	"io"
	"log"
	"net/http"

	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/kataras/iris"
)

type SurfaceController struct {
	ss     store.SurfaceStore
	logger *log.Logger
}

func NewSurfaceController(ss store.SurfaceStore, l *log.Logger) *SurfaceController {

	return &SurfaceController{ss: ss, logger: l}
}

// @Description get list of available surfaces
// @Produce  application/json
// @Success 200 files Ok
// @Failure 502 {object} controller.APIError "Internal Server Error"
// @Router /surface/ [get]
func (ssc *SurfaceController) List(ctx iris.Context) {
	bgctx := context.Background()
	info, err := ssc.ss.List(bgctx)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		ssc.logger.Println("Files can't be listed", err)
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
	surfaceID := ctx.Params().Get("surfaceID")
	bgctx := context.Background()
	reader, err := ssc.ss.Download(bgctx, surfaceID)
	if err != nil {
		ctx.StatusCode(404)
		ssc.logger.Printf("Could not read file: %s\n%s", surfaceID, err)
		return
	}

	ctx.Header("Content-Type", "application/octet-stream")

	_, err = io.Copy(ctx.ResponseWriter(), reader)
	if err != nil {
		ssc.logger.Println("Error: ", err)
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
	userID, ok := ctx.Values().Get("userID").(string)
	if !ok || userID == "" {
		userID = "seismic-cloud-api"
	}
	reader := ctx.Request().Body

	bgctx := context.Background()
	blobURL, err := ssc.ss.Upload(bgctx, ctx.Params().Get("surfaceID"), userID, reader)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		ssc.logger.Println("Could not upload file: ", err)
		return
	}

	ctx.JSON(blobURL)
}
