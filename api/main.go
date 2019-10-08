package main

import (
	"log"
	"os"

	"github.com/equinor/seismic-cloud/api/cmd"
	"github.com/equinor/seismic-cloud/api/events"
	l "github.com/equinor/seismic-cloud/api/logger"
	jww "github.com/spf13/jwalterweatherman"
)

func initLogging() {
	l.SetLogSink(os.Stdout, events.DebugLevel)
	jww.SetStdoutThreshold(jww.LevelFatal)
	log.SetPrefix("[INFO] ")
	l.AddLoggerSource("main.log", log.SetOutput)
	l.AddLoggerSource("setup.log", jww.SetLogOutput)
}

func main() {
	initLogging()
	cmd.Execute()
}
