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
	claimsmiddleware "github.com/equinor/seismic-cloud/api/middleware/claims"
	"github.com/equinor/seismic-cloud/api/service"
	jwtmiddleware "github.com/iris-contrib/middleware/jwt"
)

type serverMode int

func (sm serverMode) String() string {
	switch sm {
	case NONE:
		return "None"
	case INSECURE:
		return "Insecure"
	case SECURE:
		return "Lets Encrypt"
	case LETSENCRYPT:
		return "Secure"
	default:
		return "Unknown"
	}

}

const (
	NONE serverMode = iota
	INSECURE
	SECURE
	LETSENCRYPT
)

type HttpServer struct {
	logger        *log.Logger
	manifestStore service.ManifestStore
	stitchCmd     []string
	app           *iris.Application
	hostAddr      string
	chosenMode    serverMode
	domains       string
	domainmail    string
	privKeyFile   string
	certFile      string
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

func NewHttpServer(opts ...HttpServerOption) (hs *HttpServer, err error) {
	hs = DefaultHttpServer()
	for _, opt := range opts {
		err = opt.apply(hs)
		if err != nil {
			return nil, fmt.Errorf("Applying config failed: %v", err)
		}
	}

	if hs.manifestStore == nil {
		return nil, fmt.Errorf("Server cannot start, no manifest store set")
	}

	if hs.stitchCmd == nil || len(hs.stitchCmd) == 0 {
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

		claimsHandler := claimsmiddleware.New()

		hs.app.Use(jwtHandler.Serve)
		hs.app.Use(claimsHandler.Validate)
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
		controller.StitchController(hs.manifestStore, hs.stitchCmd, hs.logger))

}

func (hs *HttpServer) Serve() error {
	hs.registerMacros()
	hs.registerEndpoints()

	switch hs.chosenMode {
	case INSECURE:
		return hs.app.Run(iris.Addr(hs.hostAddr))
	case LETSENCRYPT:
		return hs.app.Run(iris.AutoTLS(hs.hostAddr, hs.domains, hs.domainmail))
	case SECURE:
		return hs.app.Run(iris.TLS(hs.hostAddr, hs.certFile, hs.privKeyFile))
	default:
		return fmt.Errorf("no http server mode chosen")
	}
}

func WithManifestStore(manifestStore service.ManifestStore) HttpServerOption {

	return newFuncOption(func(hs *HttpServer) (err error) {
		hs.manifestStore = manifestStore
		return
	})
}

func WithStitchCmd(stitchCmd []string) HttpServerOption {

	return newFuncOption(func(hs *HttpServer) (err error) {
		//TODO: check if it is executable
		hs.stitchCmd = stitchCmd
		return
	})
}

func WithHttpOnly() HttpServerOption {

	return newFuncOption(func(hs *HttpServer) (err error) {
		hs.chosenMode = INSECURE
		return
	})
}

func WithTLS(certFile, keyFile string) HttpServerOption {

	return newFuncOption(func(hs *HttpServer) (err error) {

		if len(certFile) == 0 {
			return fmt.Errorf("No cert file selected for TLS")
		}

		if len(keyFile) == 0 {
			return fmt.Errorf("No key file selected for TLS")
		}
		hs.chosenMode = SECURE
		hs.certFile = certFile
		hs.privKeyFile = keyFile
		return
	})
}
func WithLetsEncrypt(domains, domainmail string) HttpServerOption {

	return newFuncOption(func(hs *HttpServer) (err error) {
		if len(domains) == 0 {
			return fmt.Errorf("No domains selected for LetsEncrypt")
		}

		if len(domainmail) == 0 {
			return fmt.Errorf("No domain mail selected for LetsEncrypt")
		}
		hs.chosenMode = LETSENCRYPT
		hs.domains = domains
		hs.domainmail = domainmail
		return
	})
}
