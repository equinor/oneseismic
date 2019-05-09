package main

import (
	"testing"

	"github.com/kataras/iris/httptest"
)

func TestServer(t *testing.T) {
	app := server()
	e := httptest.New(t, app)

	e.GET("/", "hello world").Expect().Status(httptest.StatusOK).Body().Equal("Hello world!")
	e.POST("/stitch").WithJSON("yolo").Expect().Status(httptest.StatusOK).Body().Equal("\"yolo\"")
}
