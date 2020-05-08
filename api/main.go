package main

import (
	"fmt"
	"log"

	"github.com/equinor/oneseismic/api/auth"
	"github.com/equinor/oneseismic/api/events"
	"github.com/equinor/oneseismic/api/logger"
	"github.com/equinor/oneseismic/api/profiling"
	"github.com/equinor/oneseismic/api/server"
	"github.com/iris-contrib/swagger/v12"
	"github.com/joho/godotenv"
	"github.com/kataras/iris/v12"
	"github.com/pkg/profile"
	"github.com/spf13/jwalterweatherman"
	swaggerFiles "github.com/swaggo/files"
)

func init() {
	godotenv.Load() // nolint, silently ignore missing or invalid .env
}

func main() {
	c, err := parseConfig(getEnvs())
	if err != nil {
		log.Fatal("Failed to load config")
	}

	if c.profiling {
		var p *profile.Profile
		pOpts := []func(*profile.Profile){
			profile.ProfilePath("pprof"),
			profile.NoShutdownHook,
		}

		pOpts = append(pOpts, profile.MemProfile)
		p = profile.Start(pOpts...).(*profile.Profile)
		profiling.ServeMetrics("8081")
		defer p.Stop()
	}

	err = serve(c)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func serve(c *config) error {
	sigKeySet, err := auth.GetOIDCKeySet(c.AuthServer)
	if err != nil {
		return fmt.Errorf("could not get keyset: %w", err)
	}

	app := iris.Default()
	app.Use(auth.CheckJWT(sigKeySet, c.APISecret))
	app.Use(auth.ValidateClaims(c.Audience, c.Issuer))

	app.Use(iris.Gzip)
	enableSwagger(app)
	profiling.EnablePrometheusMiddleware(app)

	err = server.Register(app, c.azureBlobSettings, c.zmqReqAddr, c.zmqRepAddr)
	if err != nil {
		return fmt.Errorf("register endpoints: %w", err)
	}

	err = setupLogger(app, c.logDBConnStr)
	if err != nil {
		return fmt.Errorf("setup log: %w", err)
	}

	return app.Run(iris.Addr(c.hostAddr))
}

func enableSwagger(app *iris.Application) {
	app.Get("/swagger/{any:path}", swagger.WrapHandler(swaggerFiles.Handler))
}

func setupLogger(app *iris.Application, LogDBConnStr string) error {
	app.Logger().SetPrefix("iris: ")
	jwalterweatherman.SetStdoutThreshold(jwalterweatherman.LevelFatal)
	log.SetPrefix("[INFO] ")
	logger.AddGoLogSource(app.Logger().SetOutput)

	if len(LogDBConnStr) > 0 {
		logger.LogI("switch log sink from os.Stdout to psqlDB")

		err := logger.SetLogSink(logger.ConnString(LogDBConnStr), events.DebugLevel)
		if err != nil {
			return fmt.Errorf("switching log sink: %w", err)
		}
	}

	return nil
}
