package main

import (
	"log"

	"github.com/equinor/seismic-cloud/api/cmd"
	l "github.com/equinor/seismic-cloud/api/logger"
	jww "github.com/spf13/jwalterweatherman"
)

func initLogging() {
	jww.SetStdoutThreshold(jww.LevelFatal)
	log.SetPrefix("[INFO] ")
	l.AddLoggerSource("main.log", log.SetOutput)
	l.AddLoggerSource("setup.log", jww.SetLogOutput)
}

func main() {
	initLogging()
	cmd.Execute()
}
