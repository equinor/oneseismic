package controller

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"os/exec"
	"strings"

	"github.com/equinor/seismic-cloud/api/service"

	"github.com/kataras/iris"
)

func StitchController(ms service.ManifestStore,
	stitchCommand []string,
	logger *log.Logger) func(ctx iris.Context) {

	return func(ctx iris.Context) {
		manifestID := ctx.Params().Get("manifestID")
		logger.Printf("Stiching: manifest: %s, surface: %d bytes\n", manifestID, ctx.Request().ContentLength)

		manifest, err := ms.Fetch(manifestID)
		if err != nil {
			ctx.StatusCode(500)
			logger.Println("Stich error:", err)
			return
		}

		cmd := exec.Command(stitchCommand[0], stitchCommand[1:]...)
		manLength := uint32(len(manifest))
		manLengthBuff := make([]byte, 4)
		binary.LittleEndian.PutUint32(manLengthBuff, manLength)

		cmd.Stdin = io.MultiReader(
			strings.NewReader("M:"),
			bytes.NewBuffer(manLengthBuff),
			bytes.NewBuffer(manifest),
			ctx.Request().Body)

		cmd.Stdout = ctx.ResponseWriter()
		cmd.Stderr = logger.Writer()
		logger.Printf("Stiching: manfest: %v, length in LE: %v bytes\n", manifest, manLengthBuff)

		err = cmd.Run()
		if err != nil {
			ctx.StatusCode(500)
			logger.Println("Stich error:", err)

		} else {

		}
		cmd.Wait()

	}
}
