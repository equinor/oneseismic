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

	rsaKeys, err := auth.GetRSAKeys(os.Getenv("AUTHSERVER") + "/.well-known/openid-configuration")
	if err != nil {
		golog.Error("could not get keyset", err)
	}

	app := iris.Default()
	app.Logger().SetLevel(logLevel)
	app.Use(auth.CheckJWT(rsaKeys, []byte(os.Getenv("API_SECRET"))))
	app.Use(auth.ValidateIssuer(os.Getenv("ISSUER")))
	app.Use(iris.Gzip)
	enableSwagger(app)

	storageURL := strings.ReplaceAll(os.Getenv("AZURE_STORAGE_URL"), "{}", "%s")
	account := os.Getenv("AZURE_STORAGE_ACCOUNT")

	storageEndpoint, err := url.Parse(
		fmt.Sprintf(storageURL, account))
	if err != nil {
		golog.Fatal(err)
	}
	err = server.Register(
		app,
		*storageEndpoint,
		account,
		os.Getenv("AZURE_STORAGE_ACCESS_KEY"),
		os.Getenv("ZMQ_REQ_ADDR"),
		os.Getenv("ZMQ_REP_ADDR"),
	)
	if err != nil {
		golog.Error("register endpoints: %w", err)
	}

	err = app.Run(iris.Addr(os.Getenv("HOST_ADDR")))
	if err != nil {
		golog.Fatal(err)
	}
}

func enableSwagger(app *iris.Application) {
	app.Get("/swagger/{any:path}", swagger.WrapHandler(swaggerFiles.Handler))
}
