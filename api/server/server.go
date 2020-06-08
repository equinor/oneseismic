package server

import (
	"github.com/equinor/oneseismic/api/auth"
	"github.com/google/uuid"
	"github.com/iris-contrib/swagger/v12"
	"github.com/iris-contrib/swagger/v12/swaggerFiles"
	"github.com/kataras/iris/v12"
)

func newApp(c *Config) *iris.Application {
	app := iris.Default()

	app.Use(iris.Gzip)

	return app
}

// App for oneseismic
func App(c *Config) (*iris.Application, error) {
	app := newApp(c)

	app.Use(auth.CheckJWT(c.RSAKeys, c.APISecret))
	app.Use(auth.ValidateIssuer(c.Issuer))

	app.Get("/swagger/{any:path}", swagger.WrapHandler(swaggerFiles.Handler))
	if err := registerStoreController(app, c.StorageURL, c.AccountName, c.AccountKey); err != nil {
		return nil, err
	}
	registerSlicer(app, c.ZmqReqAddr, c.ZmqRepAddr, c.AccountName, uuid.New().String())

	return app, nil
}

func registerStoreController(app *iris.Application, storageURL, accountName, accountKey string) error {
	sURL, err := newServiceURL(storageURL, accountName, accountKey)
	if err != nil {
		return err
	}

	sc := storeController{sURL}
	app.Get("/", sc.list)
	app.Get("/{guid:string}", sc.services)
	app.Get("/{guid:string}/slice", sc.dimensions)
	app.Get("/{guid:string}/slice/{dimension:int32}", sc.lines)

	return nil
}

func registerSlicer(
	app *iris.Application,
	reqNdpt string,
	repNdpt string,
	root string,
	mPlexName string,
) {
	sc := createSliceController(reqNdpt, repNdpt, root, mPlexName)

	app.Get("/{guid:string}/slice/{dim:int32}/{lineno:int32}", sc.get)
}
