package controller

import (
	"bytes"
	"io"
	"os/exec"
	"sync"

	"github.com/kataras/iris"
)

func Stitch(ctx iris.Context) {
	// in := ctx.Body
	// cmd := exec.Command("../build/stitch", "shatter.manifest", "-i", "./cubes")
	cmd := exec.Command("cat")

	ctx.Application().Logger().Infof("Stiching: %d", ctx.Request().ContentLength)
	// infile, err := os.Open("surfaces/surface1.i32")
	// if err != nil {
	// 	ctx.StatusCode(500)
	// 	ctx.Application().Logger().Error(err)
	// }

	// io.Copy(os.Stdout, ctx.Request().Body)
	cmd.Stdin = ctx.Request().Body
	cmdOutput := &bytes.Buffer{}
	cmd.Stdout = cmdOutput

	err := cmd.Start()
	if err != nil {
		ctx.StatusCode(500)
		ctx.Application().Logger().Error(err)
	}
	var wg sync.WaitGroup
	done := false
	wg.Add(1)
	go func() {

		ctx.StreamWriter(func(w io.Writer) bool {
			n := 100000
			p := cmdOutput.Next(n)
			w.Write(p)

			if done && len(p) == 0 {
				return false
			}
			if len(p) > 0 {
				ctx.Application().Logger().Infof("Wrote %d bytes to stream", len(p))
			}
			return true // continue write
		})
		wg.Done()
	}()
	cmd.Wait()
	done = true
	wg.Wait()
}
