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
// @Accept  application/octet-stream
// @Produce  application/octet-stream
// @Param   some_id     path    string     true        "Some ID"
// @Success 200 {object} controller.fileBytes OK
// @Failure 400 {object} controller.APIError "Manifest id not found"
// @Failure 500 {object} controller.APIError "Internal Server Error"
// @Router /stitch/{manifest_id} [post]
func StitchController(
	ms store.ManifestStore,
	stitcher service.Stitcher) func(ctx iris.Context) {
	op := "stitch.file"
	return func(ctx iris.Context) {
		manifestID := ctx.Params().Get("manifestID")
		bgCtx := context.Background()
		l.LogI(op,
			fmt.Sprintf("Stitching: manifest: %s, surface: %d bytes\n",
				manifestID,
				ctx.Request().ContentLength))
		manifest, err := ms.Fetch(bgCtx, manifestID)
		if err != nil {
			ctx.StatusCode(404)

			l.LogE(op, "Manifest fetch failed", err)
			return
		}

		si, err := stitcher.Stitch(
			bgCtx,
			manifest,
			ctx.ResponseWriter(),
			ctx.Request().Body)
		if err != nil {
			ctx.StatusCode(500)

			l.LogE(op, "Stitch error", err)
		}

		ctx.Values().SetImmutable("StitchInfo", si)
	}
}

// @Description post surface query to stitch
// @Produce  application/octet-stream
// @Param   some_id     path    string     true        "Some ID"
// @Success 200 {object} controller.fileBytes OK
// @Failure 400 {object} controller.APIError "Manifest id not found"
// @Failure 400 {object} controller.APIError "Surface id not found"
// @Failure 500 {object} controller.APIError "Internal Server Error"
// @Router /stitch/{manifest_id}/{surface_id} [get]
func StitchSurfaceController(
	ms store.ManifestStore,
	ss store.SurfaceStore,
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
		surface, err := ss.Download(bgctx, surfaceID)
		if err != nil {
			l.LogE(op, "Surface fetch failed", err)
			ctx.StatusCode(404)
			return
		}

		l.LogI(op, fmt.Sprintf("Stitching: manifest: %s, surfaceID: %s \n",
			manifestID,
			surfaceID))

		si, err := stitcher.Stitch(
			bgctx,
			manifest,
			ctx.ResponseWriter(),
			surface)
		if err != nil {
			ctx.StatusCode(500)
			l.LogE(op, "Core stitch failed", err)
		}

		ctx.Values().SetImmutable("StitchInfo", si)

	}
}
