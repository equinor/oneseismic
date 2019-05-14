package server

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"

	"github.com/equinor/seismic-cloud/api/controller"

	"github.com/kataras/iris"

	"github.com/dgrijalva/jwt-go"
	"github.com/equinor/seismic-cloud/api/service"
	jwtmiddleware "github.com/iris-contrib/middleware/jwt"
)

type HttpServer struct {
	logger        *log.Logger
	manifestStore service.ManifestStore
	stitchCommand []string
	app           *iris.Application
	hostAddr      string
}
type HttpServerOption interface {
	apply(*HttpServer) error
}

func DefaultHttpServer() *HttpServer {
	return &HttpServer{
		logger:   log.New(os.Stdout, "seismic-api", log.Lshortfile),
		app:      iris.Default(),
		hostAddr: "localhost:8080"}
}

func NewHttpServer(opts ...HttpServerOption) (*HttpServer, err error) {
	hs := DefaultHttpServer()
	for _, opt := range opts {
		err = opt.apply(hs)
		if err != nil {
			return nil,fmt.Errorf("Applying config failed: %v",err)
		}
	}
	
	if hs.manifestStore == nil  {
		return nil, fmt.Errorf("Server cannot start, no manifest store set")
	}


	if hs.stitchCommand == nil || len(hs.stitchCommand) == 0  {
		return nil, fmt.Errorf("Server cannot start, stitch command is empty")
	}

	return hs, nil
}

func WithOAuth2(authServer *url.URL, resourceID string) HttpServerOption {

	return newFuncOption(func(hs *HttpServer) error {
		sigKeySet, err := service.GetKeySet(authServer)
		if err != nil {
			return fmt.Errorf("Couldn't get keyset: %v", err)
		}

		jwtHandler := jwtmiddleware.New(jwtmiddleware.Config{
			ValidationKeyGetter: func(t *jwt.Token) (interface{}, error) {
				if t.Method.Alg() != "RS256" {
					return nil, fmt.Errorf("unexpected jwt signing method=%v", t.Header["alg"])
				}
				return sigKeySet[t.Header["kid"].(string)], nil
			},

			SigningMethod: jwt.SigningMethodRS256,
		})

		hs.app.Use(jwtHandler.Serve)
		return nil
	})
}

func (hs *HttpServer) registerMacros() {
	manifestIDExpr := "^[a-zA-z0-9\\-]{1,40}$"
	manifestIDRegex, err := regexp.Compile(manifestIDExpr)
	if err != nil {
		panic(err)
	}

	hs.app.Macros().Get("string").RegisterFunc("manifestID", manifestIDRegex.MatchString)
}

func (hs *HttpServer) registerEndpoints() {
	hs.app.Get("/", func(ctx iris.Context) {
		ctx.HTML("Hello world!")
	})

	hs.app.Post("/stitch/{manifestID:string manifestID() else 502}",
		controller.StitchController(hs.manifestStore, hs.stitchCommand, hs.logger))

}

func (hs *HttpServer) Serve() error {
	hs.registerMacros()
	hs.registerEndpoints()
	return hs.app.Run(iris.Addr(hs.hostAddr))
}

func WithManifestStore(manifestStore service.ManifestStore) HttpServerOption {

	return newFuncOption(func(hs *HttpServer) (err error) {
		hs.manifestStore = manifestStore
		return
	})
}

func WithStitchCommand(stitchCommand []string) HttpServerOption {

	return newFuncOption(func(hs *HttpServer) (err error) {
		//TODO: check if it is executable
		hs.stitchCommand = stitchCommand
		return
	})
}
