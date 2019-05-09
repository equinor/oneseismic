package controller

import (
	"fmt"
	"os/exec"

	"github.com/kataras/iris"
)

func Stitch(ctx iris.Context) {

	// cmd := exec.Command("../build/stitch", "shatter.manifest", "-i", "./cubes")
	cmd := exec.Command("cat")

	fmt.Printf("Stiching: %d bytes\n", ctx.Request().ContentLength)

	cmd.Stdin = ctx.Request().Body

	cmd.Stdout = ctx.ResponseWriter()

	err := cmd.Start()
	if err != nil {
		ctx.StatusCode(500)
		fmt.Println("Stich error:", err)
	} else {
		cmd.Wait()
	}

}
