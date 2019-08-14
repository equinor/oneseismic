package controller

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log"
	"strings"

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
	stitcher service.Stitcher,
	logger *log.Logger) func(ctx iris.Context) {

	return func(ctx iris.Context) {
		manifestID := ctx.Params().Get("manifestID")
		logger.Printf("Stitching: manifest: %s, surface: %d bytes\n", manifestID, ctx.Request().ContentLength)

		manifest, err := ms.Fetch(manifestID)
		if err != nil {
			ctx.StatusCode(404)
			logger.Println("Manifest fetch failed:", err)
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
			logger.Println("Stitch error:", err)
		}

		ctx.Values().SetImmutable("StitchInfo", si)

	}
}

// @Description post surface query to stitch
// @Accept  application/octet-stream
// @Produce  application/octet-stream
// @Param   some_id     path    string     true        "Some ID"
// @Success 200 {file} file	Ok
// @Failure 400 {object} controller.APIError "Manifest id not found"
// @Failure 400 {object} controller.APIError "Surface id not found"
// @Failure 500 {object} controller.APIError "Internal Server Error"
// @Router /stitch/{maifest_id}/{surface_id} [post]
func StitchControllerWithSurfaceID(
	ms store.ManifestStore,
	ss store.SurfaceStore,
	stitcher service.Stitcher,
	logger *log.Logger) func(ctx iris.Context) {

	return func(ctx iris.Context) {
		manifestID := ctx.Params().Get("manifestID")

		manifest, err := ms.Fetch(manifestID)
		if err != nil {
			ctx.StatusCode(404)
			logger.Println("Manifest fetch failed:", err)
			return
		}

		manLength := uint32(len(manifest))
		manLengthBuff := make([]byte, 4)
		binary.LittleEndian.PutUint32(manLengthBuff, manLength)

		surfaceID := ctx.Params().Get("surfaceID")
		userID, ok := ctx.Values().Get("userID").(string)
		if !ok || userID == "" {
			userID = "seismic-cloud-api"
		}
		logger.Printf("Stitching: manifestID: %s, surfaceID: %s \n", manifestID, surfaceID)

		reader, err := ss.Download(context.Background(), surfaceID)
		if err != nil {
			ctx.StatusCode(404)
			logger.Println("Surface download failed: ", err)
			return
		}

		buf := &bytes.Buffer{}
		nRead, err := io.Copy(buf, reader)
		if err != nil {
			logger.Println(err)
		}
		logger.Printf("Stitching: manifestLength: %d bytes, surfaceLength: %d bytes\n", manLength, nRead)

		si, err := stitcher.Stitch(
			ctx.ResponseWriter(),
			io.MultiReader(
				strings.NewReader("M:"),
				bytes.NewBuffer(manLengthBuff),
				bytes.NewBuffer(manifest),
				reader))
		if err != nil {
			ctx.StatusCode(500)
			logger.Println("Stitch error:", err)
		}

		ctx.Values().SetImmutable("StitchInfo", si)

	}
}
