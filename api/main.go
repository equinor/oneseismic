package main

import (
	"os"
	"strings"

	"github.com/equinor/oneseismic/api/auth"
	_ "github.com/equinor/oneseismic/api/docs"
	"github.com/equinor/oneseismic/api/server"
	"github.com/joho/godotenv"
	"github.com/kataras/golog"
	"github.com/kataras/iris/v12"
)

func init() {
	godotenv.Load() // nolint, silently ignore missing or invalid .env
	golog.SetTimeFormat("")
	golog.SetLevel(os.Getenv("LOG_LEVEL"))
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

	var err error
	c.RSAKeys, err = auth.GetRSAKeys(os.Getenv("AUTHSERVER") + "/.well-known/openid-configuration")
	if err != nil {
		golog.Fatalf("could not get RSA keys: %v", err)
	}

	app := server.App(c)
	app.Logger().SetLevel(os.Getenv("LOG_LEVEL"))

	err = app.Run(iris.Addr(os.Getenv("HOST_ADDR")))
	if err != nil {
		golog.Fatalf("failed to start server: %v", err)
	}
}
