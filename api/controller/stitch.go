package controller

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/equinor/seismic-cloud/api/errors"

	"github.com/equinor/seismic-cloud/api/service"
	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/kataras/iris"
)

// @Description post surface query to stitch
// @Accept  application/octet-stream
// @Produce  application/octet-stream
// @Param   some_id     path    string     true        "Some ID"
// @Success 200 {file} file	Ok
// @Failure 400 {object} controller.APIError "Manifest id not found"
// @Failure 500 {object} controller.APIError "Internal Server Error"
// @Router /stitch/{maifest_id} [post]
func StitchController(
	ms store.ManifestStore,
	stitcher service.Stitcher) func(ctx iris.Context) {
	op := errors.Op("stich.file")
	return func(ctx iris.Context) {
		manifestID := ctx.Params().Get("manifestID")

		service.Log(errors.E(
			op,
			errors.InfoLevel,
			fmt.Sprintf("Stitching: manifest: %s, surface: %d bytes\n",
				manifestID,
				ctx.Request().ContentLength)))
		manifest, err := ms.Fetch(manifestID)
		if err != nil {
			ctx.StatusCode(404)
			// logger.Println("Manifest fetch failed:", err)
			service.Log(errors.E(op, "Manifest fetch failed", errors.ErrorLevel, err))
			return
		}

		manLength := uint32(len(manifest))
		manLengthBuff := make([]byte, 4)
		binary.LittleEndian.PutUint32(manLengthBuff, manLength)

		si, err := stitcher.Stitch(
			ctx.ResponseWriter(),
			io.MultiReader(
				strings.NewReader("M:"),
				bytes.NewBuffer(manLengthBuff),
				bytes.NewBuffer(manifest),
				ctx.Request().Body))
		if err != nil {
			ctx.StatusCode(500)
			service.Log(errors.E(op, "Stitch error:", errors.ErrorLevel, err))
		}

		ctx.Values().SetImmutable("StitchInfo", si)

	}
}

// @Description post surface query to stitch
// @Produce  application/octet-stream
// @Param   some_id     path    string     true        "Some ID"
// @Success 200 {file} file	Ok
// @Failure 400 {object} controller.APIError "Manifest id not found"
// @Failure 400 {object} controller.APIError "Surface id not found"
// @Failure 500 {object} controller.APIError "Internal Server Error"
// @Router /stitch/{maifest_id}/{surface_id} [get]
func StitchControllerWithSurfaceID(
	ms store.ManifestStore,
	ss store.SurfaceStore,
	stitcher service.Stitcher) func(ctx iris.Context) {
	op := errors.Op("stitch.surfaceid")
	return func(ctx iris.Context) {
		manifestID := ctx.Params().Get("manifestID")

		manifest, err := ms.Fetch(manifestID)
		if err != nil {
			ctx.StatusCode(404)
			service.Log(errors.E(op, "Manifest fetch failed", errors.ErrorLevel, err))
			return
		}

		manLength := uint32(len(manifest))
		manLengthBuff := make([]byte, 4)
		binary.LittleEndian.PutUint32(manLengthBuff, manLength)

		surfaceID := ctx.Params().Get("surfaceID")
		service.Log(errors.E(
			op,
			errors.InfoLevel,
			fmt.Sprintf("Stitching: manifest: %s, surfaceID: %s \n",
				manifestID,
				surfaceID)))

		reader, err := ss.Download(context.Background(), surfaceID)
		if err != nil {
			ctx.StatusCode(404)
			service.Log(errors.E(op, "Surface download failed:", errors.WarnLevel, err))
			return
		}

		service.Log(errors.E(op, fmt.Sprintf("Stitching: manifestLength: %d bytes", manLength), errors.InfoLevel))

		si, err := stitcher.Stitch(
			ctx.ResponseWriter(),
			io.MultiReader(
				strings.NewReader("M:"),
				bytes.NewBuffer(manLengthBuff),
				bytes.NewBuffer(manifest),
				reader))
		if err != nil {
			ctx.StatusCode(500)
			service.Log(errors.E(op, "core stitch", errors.ErrorLevel, err))
		}

		ctx.Values().SetImmutable("StitchInfo", si)

	}
}
