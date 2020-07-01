package main

import (
	"crypto/rsa"
	"fmt"
	"log"
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
	c, err := getConfig()
	if err != nil {
		log.Fatal("Failed to load config", err)
	}

	err = serve(c)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func serve(c *config) error {
	logLevel := os.Getenv("LOG_LEVEL")
	golog.SetTimeFormat("")
	golog.SetLevel(logLevel)

	app := iris.Default()

	app.Logger().SetLevel(logLevel)

	rsaKeys, err := auth.GetRSAKeys(c.authServer + "/.well-known/openid-configuration")
	if err != nil {
		return fmt.Errorf("could not get keyset: %w", err)
	}
	app.Use(auth.CheckJWT(rsaKeys, c.apiSecret))
	app.Use(auth.ValidateIssuer(c.issuer))

	app.Use(iris.Gzip)
	enableSwagger(app)

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

type config struct {
	hostAddr    string
	storageURL  string
	accountName string
	accountKey  string
	authServer  string
	issuer      string
	apiSecret   []byte
	zmqReqAddr  string
	zmqRepAddr  string
	SigKeySet   map[string]rsa.PublicKey
}

func getConfig() (*config, error) {
	conf := &config{
		authServer:  os.Getenv("AUTHSERVER"),
		apiSecret:   []byte(os.Getenv("API_SECRET")),
		issuer:      os.Getenv("ISSUER"),
		storageURL:  strings.ReplaceAll(os.Getenv("AZURE_STORAGE_URL"), "{}", "%s"),
		accountName: os.Getenv("AZURE_STORAGE_ACCOUNT"),
		accountKey:  os.Getenv("AZURE_STORAGE_ACCESS_KEY"),
		hostAddr:    os.Getenv("HOST_ADDR"),
		zmqRepAddr:  os.Getenv("ZMQ_REP_ADDR"),
		zmqReqAddr:  os.Getenv("ZMQ_REQ_ADDR"),
	}

	return conf, nil
}
