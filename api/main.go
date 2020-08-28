package main

import (
	"net/url"
	"os"
	"time"

	"github.com/equinor/oneseismic/api/auth"
	"github.com/equinor/oneseismic/api/server"
	"github.com/joho/godotenv"
	"github.com/kataras/golog"
	"github.com/kataras/iris/v12"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	godotenv.Load() // nolint, silently ignore missing or invalid .env
}

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		panic("unknown LOG_LEVEL: " + logLevel)
	}
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(level)
	oidConf, err := auth.GetOidConfig(os.Getenv("AUTHSERVER") + "/v2.0/.well-known/openid-configuration")
	if err != nil {
		log.Error().Err(err).Msg("could not get keyset")
	}

	storageURL, err := url.Parse(os.Getenv("AZURE_STORAGE_URL"))
	if err != nil {
		log.Fatal().Err(err)
	}

	golog.SetTimeFormat(time.RFC3339Nano)
	golog.SetLevel(logLevel)
	app := iris.Default()

	app.Use(auth.CheckJWT(oidConf.Jwks))
	app.Use(auth.Validate(oidConf.Issuer, os.Getenv("AUDIENCE")))
	app.Use(auth.OboJWT(oidConf.TokenEndpoint, os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET")))

	server.Register(
		app,
		*storageURL,
		os.Getenv("ZMQ_REQ_ADDR"),
		os.Getenv("ZMQ_REP_ADDR"),
		os.Getenv("ZMQ_FAILURE_ADDR"),
	)

	err = app.Run(iris.Addr(os.Getenv("HOST_ADDR")))
	if err != nil {
		log.Fatal().Err(err)
	}
}
