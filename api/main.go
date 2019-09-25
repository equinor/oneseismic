package main

import (
	"log"
	"os"
	"time"

	"github.com/equinor/seismic-cloud/api/cmd"
	"github.com/equinor/seismic-cloud/api/events"
	l "github.com/equinor/seismic-cloud/api/logger"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func recordMetrics() {
	go func() {
			for {
					opsProcessed.Inc()
					time.Sleep(2 * time.Second)
			}
	}()
}

var (
	opsProcessed = promauto.NewCounter(prometheus.CounterOpts{
			Name: "myapp_processed_ops_total",
			Help: "The total number of processed events",
	})
)

func initLogging() {
	l.SetLogSink(os.Stdout, events.DebugLevel)
	jww.SetStdoutThreshold(jww.LevelFatal)
	log.SetPrefix("[INFO] ")
	l.AddLoggerSource("main.log", log.SetOutput)
	l.AddLoggerSource("setup.log", jww.SetLogOutput)
}

func main() {
	recordMetrics()
	initLogging()
	cmd.Execute()
}
