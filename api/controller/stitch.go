package controller

import (
	"os/exec"

	"github.com/kataras/iris"
)

func Stitch(ctx iris.Context) {

	// cmd := exec.Command("../build/stitch", "shatter.manifest", "-i", "./cubes")
	cmd := exec.Command("cat")

	ctx.Application().Logger().Infof("Stiching: %d", ctx.Request().ContentLength)

	cmd.Stdin = ctx.Request().Body

	cmd.Stdout = ctx.ResponseWriter()

	err := cmd.Start()
	if err != nil {
		ctx.StatusCode(500)
		ctx.Application().Logger().Error(err)
	} else {
		cmd.Wait()
	}

}
