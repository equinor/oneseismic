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

	app.Use(auth.CheckJWT(c.SigKeySet, c.APISecret))
	app.Use(auth.ValidateIssuer(c.Issuer))

	app.Use(iris.Gzip)

	return app
}

// App for oneseismic
func App(c *Config) (*iris.Application, error) {
	app := newApp(c)

	app.Get("/swagger/{any:path}", swagger.WrapHandler(swaggerFiles.Handler))
	registerStoreController(app, c.StorageURL)
	registerSlicer(app, c.StorageURL, c.ZmqReqAddr, c.ZmqRepAddr, uuid.New().String())

	return app, nil
}

func registerStoreController(app *iris.Application, uri string) error {
	sc := storeController{&storageURL{uri}}
	app.Get("/{root:string}/", sc.list)
	app.Get("/{root:string}/{guid:string}", sc.services)
	app.Get("/{root:string}/{guid:string}/slice", sc.dimensions)
	app.Get("/{root:string}/{guid:string}/slice/{dim:int32}", sc.lines)

	return nil
}

func registerSlicer(
	app *iris.Application,
	uri string,
	reqNdpt string,
	repNdpt string,
	mPlexName string,
) {
	sc := createSliceController(uri, reqNdpt, repNdpt, mPlexName)

	app.Get("/{root:string}/{guid:string}/slice/{dim:int32}/{lineno:int32}", sc.slice)
}
