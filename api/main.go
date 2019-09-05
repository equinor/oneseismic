package main

import (
	"log"
	"os"

	"github.com/equinor/seismic-cloud/api/cmd"
	"github.com/equinor/seismic-cloud/api/errors"
	"github.com/equinor/seismic-cloud/api/service"
	jww "github.com/spf13/jwalterweatherman"
)

func initLogging() {
	service.SetLogSink(os.Stdout, errors.DebugLevel)
	jww.SetStdoutThreshold(jww.LevelFatal)
	service.WrapLogger("main.log", log.SetOutput)
	service.WrapLogger("setup.log", jww.SetLogOutput)
}

func main() {
	initLogging()
	cmd.Execute()
}
