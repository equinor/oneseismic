package controller

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"os/exec"
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
	pipeProvider service.NamedPipeProvider,
	stitchCommand []string,
	logger *log.Logger) func(ctx iris.Context) {

	return func(ctx iris.Context) {
		tmpCmd := make([]string, len(stitchCommand))
		copy(tmpCmd, stitchCommand)
		manifestID := ctx.Params().Get("manifestID")
		logger.Printf("Stiching: manifest: %s, surface: %d bytes\n", manifestID, ctx.Request().ContentLength)

		manifest, err := ms.Fetch(manifestID)
		if err != nil {
			ctx.StatusCode(404)
			logger.Println("Stich error:", err)
			return
		}

		profiling := true
		pr, pw, err := pipeProvider.New("tmpFile")
		if err != nil {
			ctx.StatusCode(500)
			logger.Println("Stich error:", err)
			return
		}
		defer pw.Close()
		if profiling {

			tmpCmd = append(tmpCmd, "--pipe", "/tmp/tmpFile")
		}
		profileBuffer := bytes.NewBuffer(make([]byte, 0))
		go func() {
			defer pr.Close()
			// copy the data written to the PipeReader via the cmd to stdout
			if _, err := io.Copy(profileBuffer, pr); err != nil {
				log.Fatal(err)
			}
		}()
		log.Println("Running", tmpCmd)
		cmd := exec.Command(tmpCmd[0], tmpCmd[1:]...)
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
		}

		cmd.Wait()
		if profiling {
			ctx.Values().SetImmutable("Stitch", profileBuffer.String())

		}
	}
}
