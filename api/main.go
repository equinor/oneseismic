package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/equinor/oneseismic/api/auth"
	_ "github.com/equinor/oneseismic/api/docs"
	"github.com/equinor/oneseismic/api/server"
	"github.com/iris-contrib/swagger/v12"
	"github.com/iris-contrib/swagger/v12/swaggerFiles"
	"github.com/joho/godotenv"
	"github.com/kataras/golog"
	"github.com/kataras/iris/v12"
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
	logLevel := os.Getenv("LOG_LEVEL")

	golog.SetTimeFormat("")
	golog.SetLevel(logLevel)
	rsaKeys, err := auth.GetRSAKeys(os.Getenv("AUTHSERVER") + "/v2.0/.well-known/openid-configuration")
	if err != nil {
		golog.Error("could not get keyset", err)
	}

	storageURL := strings.ReplaceAll(os.Getenv("AZURE_STORAGE_URL"), "{}", "%s")
	account := os.Getenv("AZURE_STORAGE_ACCOUNT")

	storageEndpoint, err := url.Parse(
		fmt.Sprintf(storageURL, account))
	if err != nil {
		golog.Fatal(err)
	}

	app := iris.Default()

	app.Use(auth.CheckJWT(rsaKeys))
	app.Use(auth.Validate(os.Getenv("ISSUER"), os.Getenv("AUDIENCE")))
	app.Use(iris.Gzip)

	err = server.Register(
		app,
		*storageEndpoint,
		account,
		os.Getenv("AZURE_STORAGE_ACCESS_KEY"),
		os.Getenv("ZMQ_REQ_ADDR"),
		os.Getenv("ZMQ_REP_ADDR"),
		os.Getenv("ZMQ_FAILURE_ADDR"),
	)
	if err != nil {
		golog.Error("creating app: %w", err)
	}
	app.Get("/swagger/{any:path}", swagger.WrapHandler(swaggerFiles.Handler))

	err = app.Run(iris.Addr(os.Getenv("HOST_ADDR")))
	if err != nil {
		golog.Fatal(err)
	}
}
