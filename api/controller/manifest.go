package controller

import (
	"context"
	"fmt"
	"net/http"

	l "github.com/equinor/seismic-cloud-api/api/logger"
	"github.com/equinor/seismic-cloud-api/api/service/store"
	"github.com/kataras/iris/v12"
)

type ManifestController struct {
	ms store.ManifestStore
}

func NewManifestController(ms store.ManifestStore) *ManifestController {
	return &ManifestController{ms: ms}
}

// @Description get manifest file
// @Produce  application/octet-stream
// @Param   surfaceID  path    string     true        "File ID"
// @Success 200 {object} controller.Bytes OK
// @Failure 502 {object} controller.APIError "Internal Server Error"
// @Router /manifest/{manifest_id} [get]
func (msc *ManifestController) Fetch(ctx iris.Context) {
	manifestID := ctx.Params().Get("manifestID")
	bgctx := context.Background()
	manifest, err := msc.ms.Fetch(bgctx, manifestID)
	if err != nil {
		ctx.StatusCode(404)
		l.LogE(fmt.Sprintf("Could not download manifest: %s", manifestID), err)
		return
	}

	ctx.Header("Content-Type", "application/json")
	_, err = ctx.JSON(manifest)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		l.LogE(fmt.Sprintf("Error writing to response, manifestID %s", manifestID), err)
		return
	}
}
