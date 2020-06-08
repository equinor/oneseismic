package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/equinor/oneseismic/api/auth"
	_ "github.com/equinor/oneseismic/api/docs"
	"github.com/equinor/oneseismic/api/events"
	"github.com/equinor/oneseismic/api/logger"
	"github.com/equinor/oneseismic/api/profiling"
	"github.com/equinor/oneseismic/api/server"
	"github.com/joho/godotenv"
	"github.com/kataras/iris/v12"
	"github.com/pkg/profile"
)

func init() {
	godotenv.Load() // nolint, silently ignore missing or invalid .env
}

//@title oneseismic
//@description oneseismic
//@license.name AGPL3
//@contact.name Equinor
//@securityDefinitions.apikey ApiKeyAuth
//@in header
//@name Authorization
func main() {
	c := &server.Config{
		APISecret:  []byte(os.Getenv("API_SECRET")),
		Issuer:     os.Getenv("ISSUER"),
		StorageURL: strings.ReplaceAll(os.Getenv("AZURE_STORAGE_URL"), "{}", "%s"),
		ZmqRepAddr: os.Getenv("ZMQ_REP_ADDR"),
		ZmqReqAddr: os.Getenv("ZMQ_REQ_ADDR"),
	}

	logDB := os.Getenv("LOGDB_CONNSTR")
	if len(logDB) > 0 {
		err := logger.SetLogSink(logger.ConnString(logDB), events.DebugLevel)
		if err != nil {
			log.Fatalf("switching log sink to db: %v", err)
		}
	}
	var err error
	c.SigKeySet, err = auth.GetOIDCKeySet(os.Getenv("AUTHSERVER"))
	if err != nil {
		log.Fatalf("could not get keyset: %v", err)
	}

	app, err := server.App(c)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	app.Logger().SetLevel(os.Getenv("LOG_LEVEL"))
	logger.AddGoLogSource(app.Logger().SetOutput)

	doProfiling, _ := strconv.ParseBool(os.Getenv("PROFILING"))
	if doProfiling {
		var p *profile.Profile
		pOpts := []func(*profile.Profile){
			profile.ProfilePath("pprof"),
			profile.NoShutdownHook,
		}

		pOpts = append(pOpts, profile.MemProfile)
		p = profile.Start(pOpts...).(*profile.Profile)
		profiling.ServeMetrics("8081")
		profiling.EnablePrometheusMiddleware(app)
		defer p.Stop()
	}

	err = app.Run(iris.Addr(os.Getenv("HOST_ADDR")))
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
