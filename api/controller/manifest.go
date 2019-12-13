package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
// @Param   manifestID  path    string     true        "File ID"
// @Success 200 {object} controller.Bytes OK
// @Failure 404 {string} controller.APIError "Manifest not found"
// @Failure 502 {object} controller.APIError "Internal Server Error"
// @Router /manifest/{manifest_id} [get]
func (msc *ManifestController) Download(ctx iris.Context) {
	manifestID := ctx.Params().Get("manifestID")
	bgctx := context.Background()
	manifest, err := msc.ms.Download(bgctx, manifestID)
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

// @Description post manifest
// @Produce  application/octet-stream
// @Param   manifestID  path    string     true        "File ID"
// @Success 200 {string} controller.Bytes OK
// @Failure 502 {string} controller.APIError "Internal Server Error"
// @Router /manifest/{manifest_id} [post]
func (msc *ManifestController) Upload(ctx iris.Context) {
	manifestID := ctx.Params().Get("manifestID")
	var mani store.Manifest

	dr := json.NewDecoder(ctx.Request().Body)
	err := dr.Decode(&mani)
	if err != nil {
		ctx.StatusCode(502)
		l.LogE("Unmarshaling to Manifest", err)
		return
	}
	dctx, cancel := context.WithTimeout(ctx.Request().Context(), 1*time.Second)
	defer cancel()
	err = msc.ms.Upload(dctx, manifestID, mani)
	if err != nil {
		ctx.StatusCode(502)
		l.LogE(fmt.Sprintf("Upload manifest: %s", manifestID), err)
		return
	}

}
