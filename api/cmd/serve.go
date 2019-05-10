package cmd

import (
	"fmt"
	"log"
	"os"
	"regexp"

	"github.com/equinor/seismic-cloud/api/config"

	"github.com/equinor/seismic-cloud/api/controller"
	"github.com/equinor/seismic-cloud/api/service"

	"github.com/dgrijalva/jwt-go"
	jwtmiddleware "github.com/iris-contrib/middleware/jwt"
	"github.com/kataras/iris"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "serve seismic cloud provider",
	Long:  `serve seismic cloud provider.`,
	Run:   runServe,
}

func server(auth bool) *iris.Application {
	app := iris.Default()

	if auth == true {
		authServer := config.AuthServer()

		sigKeySet, err := service.GetKeySet(authServer)
		if err != nil {
			log.Fatal(fmt.Errorf("Couldn't get keyset: %v", err))
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
	}

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

func runServe(cmd *cobra.Command, args []string) {

	if viper.ConfigFileUsed() == "" {
		jww.ERROR.Println("No config file loaded")
		os.Exit(1)
	} else {
		jww.INFO.Println("Using config file:", viper.ConfigFileUsed())
	}

	app := server(config.UseAuth())
	app.Run(iris.Addr(config.HostAddr()))
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
