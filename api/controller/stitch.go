package controller

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/kataras/iris"
)

func Stitch(ctx iris.Context) {
	// in := ctx.Body
	cmd := exec.Command("../build/stitch", "shatter.manifest", "-i", "./cubes")
	ctx.Application().Logger().Infof("Hello")
	infile, err := os.Open("surfaces/surface1.i32")
	if err != nil {
		panic(err)
	}

	cmd.Stdin = infile
	var buffer bytes.Buffer
	cmd.Stdout = &buffer
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
	ctx.Write(buffer.Bytes())
	// ctx.StreamWriter(func(w io.Writer) bool {
	// 	fmt.Fprintf(w, "Message number %d<br>", ints[i])
	// 	time.Sleep(500 * time.Millisecond) // simulate delay.
	// 	if i == len(ints)-1 {
	// 		return false // close and flush
	// 	}
	// 	i++
	// 	return true // continue write
	// })
}
