package controller

import (
	"bytes"
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
		logger.Printf("Stiching: manifest: %s, surface: %d bytes\n", manifestID, ctx.Request().ContentLength)

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
			logger.Println("Stich error:", err)
		}

		ctx.Values().SetImmutable("StitchInfo", si)

	}
}
