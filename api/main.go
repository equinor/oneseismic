package main

import (
	"fmt"
	"log"

	"github.com/equinor/oneseismic/api/auth"
	_ "github.com/equinor/oneseismic/api/docs"
	"github.com/equinor/oneseismic/api/events"
	"github.com/equinor/oneseismic/api/logger"
	"github.com/equinor/oneseismic/api/profiling"
	"github.com/equinor/oneseismic/api/server"
	"github.com/iris-contrib/swagger/v12"
	"github.com/iris-contrib/swagger/v12/swaggerFiles"
	"github.com/joho/godotenv"
	"github.com/kataras/iris/v12"
	"github.com/pkg/profile"
	"github.com/spf13/jwalterweatherman"
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
	c, err := getConfig()
	if err != nil {
		log.Fatal("Failed to load config", err)
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
	app := iris.Default()

	sigKeySet, err := auth.GetOIDCKeySet(c.authServer)
	if err != nil {
		return fmt.Errorf("could not get keyset: %w", err)
	}
	app.Use(auth.CheckJWT(sigKeySet, c.apiSecret))
	app.Use(auth.ValidateClaims(c.audience, c.issuer))

	app.Use(iris.Gzip)
	enableSwagger(app)
	profiling.EnablePrometheusMiddleware(app)

	if err := server.Register(
		app,
		c.storageURL,
		c.accountName,
		c.accountKey,
		c.zmqReqAddr,
		c.zmqRepAddr,
	); err != nil {
		return fmt.Errorf("register endpoints: %w", err)
	}

	if err := setupLogger(app, c.logDBConnStr); err != nil {
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
