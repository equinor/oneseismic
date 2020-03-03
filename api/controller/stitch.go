package controller

import (
	"context"
	"fmt"

	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/equinor/seismic-cloud/api/service"
	"github.com/kataras/iris/v12"
)

// @Description post surface query to stitch
// @ID stitch
// @Produce  application/octet-stream
// @Param   manifest_id     path    string     true        "The id of a manifest"
// @Param   surface_id     path    string     true        "The id of a surface"
// @Success 200 {string} string "byte stream"
// @Failure 400 {object} controller.APIError "Manifest id not found"
// @Failure 400 {object} controller.APIError "Surface id not found"
// @Failure 500 {object} controller.APIError "Internal Server Error"
// @security ApiKeyAuth
// @tags stitch
// @Router /stitch/{manifest_id}/{surface_id} [get]
func StitchSurfaceController(
	ms service.ManifestStore,
	stitcher service.Stitcher) func(ctx iris.Context) {
	return func(ctx iris.Context) {
		manifestID := ctx.Params().Get("manifestID")
		bgctx := context.Background()

		mani, err := ms.Download(bgctx, manifestID)
		if err != nil {
			ctx.StatusCode(404)
			l.LogE("Manifest fetch failed", err)
			return
		}

		surfaceID := ctx.Params().Get("surfaceID")

		l.LogI(fmt.Sprintf("Stitching: manifest: %s, surfaceID: %s \n",
			manifestID,
			surfaceID))

		si, err := stitcher.Stitch(
			bgctx,
			ctx.ResponseWriter(),
			service.StitchParams{
				Dim:          0,
				CubeManifest: mani,
			})
		if err != nil {
			ctx.StatusCode(500)
			l.LogE("Core stitch failed", err)
		}

		ctx.Values().SetImmutable("StitchInfo", si)

	}
}

// @Description post surface query to stitch
// @ID stitch_dim
// @Produce  application/octet-stream
// @Param   manifest_id     path    string     true        "The id of a manifest"
// @Param   dim     path    int     true        "The dimension, either of 0,1,2"
// @Param   lineno     path    int     true        "The line number"
// @Success 200 {object} controller.Bytes OK
// @Failure 400 {object} controller.APIError "Manifest id not found"
// @Failure 500 {object} controller.APIError "Internal Server Error"
// @security ApiKeyAuth
// @tags stitch
// @Router /stitch/{manifest_id}/dim/{dim}/{lineno} [get]
func StitchDimController(
	ms service.ManifestStore,
	stitcher service.Stitcher) func(ctx iris.Context) {
	return func(ctx iris.Context) {
		manifestID := ctx.Params().Get("manifestID")
		bgctx := context.Background()

		mani, err := ms.Download(bgctx, manifestID)
		if err != nil {
			ctx.StatusCode(404)
			l.LogE("Manifest fetch failed", err)
			return
		}

		dim, ok := ctx.Params().GetIntUnslashed("dim")

		if !ok {
			ctx.StatusCode(400)
			l.LogE("Dim convert", fmt.Errorf("Dim not found"))
			return
		}

		lineno, ok := ctx.Params().GetIntUnslashed("lineno")
		if !ok {
			ctx.StatusCode(400)
			l.LogE("Lineno convert", fmt.Errorf("Lineno not found"))
			return
		}
		l.LogI(fmt.Sprintf("Stitching: manifest: %s, dim: %d\n",
			manifestID,
			dim))

		si, err := stitcher.Stitch(
			bgctx,
			ctx.ResponseWriter(),
			service.StitchParams{
				Dim:          int32(dim),
				Lineno:       int64(lineno),
				CubeManifest: mani,
			})
		if err != nil {
			ctx.StatusCode(500)
			l.LogE("Core stitch failed", err)
		}

		ctx.Values().SetImmutable("StitchInfo", si)

	}
}
