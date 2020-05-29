package server

import (
	"github.com/equinor/oneseismic/api/auth"
	"github.com/equinor/oneseismic/api/logger"
	"github.com/equinor/oneseismic/api/profiling"
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

// Serve oneseismic
func Serve(c *Config) error {
	app := newApp(c)

	app.Logger().SetLevel(c.LogLevel)
	logger.AddGoLogSource(app.Logger().SetOutput)

	if c.Profiling {
		profiling.EnablePrometheusMiddleware(app)
	}

	registerStoreController(app, c.StorageURL)
	registerSlicer(app, c.ZmqReqAddr, c.ZmqRepAddr, uuid.New().String())
	app.Get("/swagger/{any:path}", swagger.WrapHandler(swaggerFiles.Handler))

	return app.Run(iris.Addr(c.HostAddr))
}

func registerStoreController(app *iris.Application, uri string) {
	sc := storeController{&storageURL{uri}}
	app.Get("/{root:string}", sc.list)
	app.Get("/{root:string}/{guid:string}", sc.services)
	app.Get("/{root:string}/{guid:string}/slice", sc.dimensions)
	app.Get("/{root:string}/{guid:string}/slice/{dimension:int32}", sc.lines)
}

func registerSlicer(
	app *iris.Application,
	reqNdpt string,
	repNdpt string,
	mPlexName string,
) {
	sc := createSliceController(reqNdpt, repNdpt, mPlexName)
	app.Get("/{root:string}/{guid:string}/slice/{dim:int32}/{lineno:int32}", sc.get)
}
