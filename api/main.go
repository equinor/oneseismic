package main

import (
	"equinor/seismic-cloud/api/controller"
	"equinor/seismic-cloud/api/service"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"

	"github.com/dgrijalva/jwt-go"
	jwtmiddleware "github.com/iris-contrib/middleware/jwt"
	"github.com/kataras/iris"
)

func getAuthServer() (*url.URL, error) {
	envAuth := os.Getenv("AUTHSERVER")
	if len(envAuth) == 0 {
		return nil, fmt.Errorf("No authserver set")
	}
	u, err := url.Parse(envAuth)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func server() *iris.Application {
	app := iris.Default()

	authServer, err := getAuthServer()
	if err != nil {
		log.Panic(fmt.Errorf("Couldn't read authserver: %v", err))
	}
	sigKeySet, err := service.GetKeySet(authServer)
	if err != nil {
		log.Panic(fmt.Errorf("Couldn't get keyset: %v", err))
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

	app.Use(jwtHandler.Serve)

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("Hello world!")
	})

	app.Post("/stitch", controller.Stitch)

	manifestIDExpr := "^[a-zA-z0-9\\-]{1,40}$"
	manifestIDRegex, err := regexp.Compile(manifestIDExpr)
	if err != nil {
		panic(err)
	}

	app.Macros().Get("string").RegisterFunc("manifestID", manifestIDRegex.MatchString)
	app.Post("/stitch/{id:string manifestID() else 502}", func(ctx iris.Context) {
		ctx.HTML("Hello id: " + ctx.Params().Get("id"))
	})

	return app
}

func main() {
	app := server()
	app.Run(iris.Addr(":8080"))
}
