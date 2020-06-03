package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

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

	app.Logger().SetLevel(c.logLevel)
	logger.AddGoLogSource(app.Logger().SetOutput)
	if err := logToDB(app, c.logDBConnStr); err != nil {
		return err
	}

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

	return app.Run(iris.Addr(c.hostAddr))
}

func enableSwagger(app *iris.Application) {
	app.Get("/swagger/{any:path}", swagger.WrapHandler(swaggerFiles.Handler))
}

func logToDB(app *iris.Application, logDBConnStr string) error {

	if len(logDBConnStr) > 0 {
		logger.LogI("switch log sink from os.Stdout to psqlDB")

		err := logger.SetLogSink(logger.ConnString(logDBConnStr), events.DebugLevel)
		if err != nil {
			return fmt.Errorf("switching log sink: %w", err)
		}
	}

	return nil
}

type config struct {
	profiling    bool
	hostAddr     string
	storageURL   string
	accountName  string
	accountKey   string
	logDBConnStr string
	logLevel     string
	authServer   *url.URL
	audience     string
	issuer       string
	apiSecret    []byte
	zmqReqAddr   string
	zmqRepAddr   string
}

func getConfig() (*config, error) {
	authServer, err := url.ParseRequestURI(os.Getenv("AUTHSERVER"))
	if err != nil {
		return nil, fmt.Errorf("invalid AUTHSERVER: %w", err)
	}

	profiling, err := strconv.ParseBool(os.Getenv("PROFILING"))
	if err != nil {
		return nil, fmt.Errorf("could not parse PROFILING: %w", err)
	}

	conf := &config{
		authServer:   authServer,
		apiSecret:    []byte(os.Getenv("API_SECRET")),
		audience:     os.Getenv("RESOURCE_ID"),
		issuer:       os.Getenv("ISSUER"),
		storageURL:   strings.ReplaceAll(os.Getenv("AZURE_STORAGE_URL"), "{}", "%s"),
		accountName:  os.Getenv("AZURE_STORAGE_ACCOUNT"),
		accountKey:   os.Getenv("AZURE_STORAGE_ACCESS_KEY"),
		hostAddr:     os.Getenv("HOST_ADDR"),
		logDBConnStr: os.Getenv("LOGDB_CONNSTR"),
		logLevel:     os.Getenv("LOG_LEVEL"),
		profiling:    profiling,
		zmqRepAddr:   os.Getenv("ZMQ_REP_ADDR"),
		zmqReqAddr:   os.Getenv("ZMQ_REQ_ADDR"),
	}

	return conf, nil
}
