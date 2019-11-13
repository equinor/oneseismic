package controller

import (
	"context"
	"fmt"

	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/equinor/seismic-cloud/api/service"
	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/kataras/iris/v12"
)

// @Description post surface query to stitch
// @Produce  application/octet-stream
// @Param   some_id     path    string     true        "Some ID"
// @Success 200 {object} controller.fileBytes OK
// @Failure 400 {object} controller.APIError "Manifest id not found"
// @Failure 400 {object} controller.APIError "Surface id not found"
// @Failure 500 {object} controller.APIError "Internal Server Error"
// @Router /stitch/{manifest_id}/{surface_id} [get]
func StitchController(
	ms store.ManifestStore,
	stitcher service.Stitcher) func(ctx iris.Context) {
	op := "stitch.surfaceid"
	return func(ctx iris.Context) {
		manifestID := ctx.Params().Get("manifestID")
		bgctx := context.Background()

		manifest, err := ms.Fetch(bgctx, manifestID)
		if err != nil {
			ctx.StatusCode(404)
			l.LogE(op, "Manifest fetch failed", err)
			return
		}

		surfaceID := ctx.Params().Get("surfaceID")

		l.LogI(op, fmt.Sprintf("Stitching: manifest: %s, surfaceID: %s \n",
			manifestID,
			surfaceID))

		si, err := stitcher.Stitch(
			bgctx,
			manifest,
			ctx.ResponseWriter(),
			surfaceID)
		if err != nil {
			ctx.StatusCode(500)
			l.LogE(op, "Core stitch failed", err)
		}

		ctx.Values().SetImmutable("StitchInfo", si)

	}
}
