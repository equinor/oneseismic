package controller

import (
	"bytes"
	"log"
	"os/exec"

	"github.com/equinor/seismic-cloud/api/service"

	"github.com/kataras/iris"
)

func StitchController(ms service.ManifestStore,
	cmd *exec.Cmd,
	logger *log.Logger) func(ctx iris.Context) {

	return func(ctx iris.Context) {
		manifestID := ctx.Params().Get("manifestID")
		logger.Printf("Stiching: manifest: %s, surface: %d bytes\n", manifestID, ctx.Request().ContentLength)

		cmd.Stdin = ctx.Request().Body

		var buffer bytes.Buffer
		cmd.Stdout = &buffer

		err := cmd.Run()
		if err != nil {
			ctx.StatusCode(500)
			logger.Println("Stich error:", err)
		} else {
			cmd.Wait()
		}
		ctx.Write(buffer.Bytes())
	}
}
