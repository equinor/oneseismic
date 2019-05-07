package controller

import (
	"bytes"
	"os/exec"

	"github.com/kataras/iris"
)

func Stitch(ctx iris.Context) {
	cmd := exec.Command("../build/stitch", "shatter.manifest", "-i", "./cubes")

	cmd.Stdin = ctx.Request().Body

	var buffer bytes.Buffer
	cmd.Stdout = &buffer

	err := cmd.Run()
	if err != nil {
		ctx.StatusCode(500)
		ctx.Application().Logger().Error(err)
	}
	ctx.Write(buffer.Bytes())
}
