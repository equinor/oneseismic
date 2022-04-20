package main

import (
	"fmt"
	"log"
	"os"

	"github.com/pborman/getopt/v2"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/log/zapadapter"
	"go.uber.org/zap"

	"github.com/equinor/oneseismic/api/catalogue"
	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/postgres"
)

type opts struct {
	authserver string
	audience   string
	connstring string
	port       int
}

func parseopts() opts {
	help := getopt.BoolLong("help", 0, "help")
	opts := opts {
		authserver: os.Getenv("AUTHSERVER"),
		audience:   os.Getenv("AUDIENCE"),
		connstring: os.Getenv("CONNECTIONSTRING"),
		port:       8080,
	}

	getopt.FlagLong(
		&opts.authserver,
		"authserver",
		0,
		"OpenID Connect discovery server",
		"string",
	)
	getopt.FlagLong(
		&opts.audience,
		"audience",
		0,
		"Application (client) ID",
		"string",
	)
	getopt.FlagLong(
		&opts.connstring,
		"connectionstring",
		0,
		"Postgres DB connection string",
		"string",
	)

	getopt.FlagLong(
		&opts.port,
		"port",
		'p',
		"Port to start server on. Defaults to 8080",
		"int",
	)

	getopt.Parse()
	if *help {
		getopt.Usage()
		os.Exit(0)
	}

	return opts
}

func main() {
	opts := parseopts()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	pool, err := postgres.MakeConnectionPool(
		opts.connstring,
		zapadapter.NewLogger(logger),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	dbschema := &postgres.Schema{
		Table: "oneseismic.catalogue",
		Cols: postgres.Columns {
			Manifest: "manifest",
			Geometry: "geometry",
		},
	}

	client := postgres.NewPgClient(pool, dbschema)

	gql := catalogue.MakeGraphQL(client)

	provider := auth.GetJwksProvider(opts.authserver)
	tokenvalidator := auth.JWTvalidation(
		opts.authserver,
		opts.audience,
		provider.KeyFunc,
	)

	app := gin.Default()
	app.Use(tokenvalidator)

	graphql := app.Group("/graphql")
	graphql.GET( "", gql.Get)
	graphql.POST("", gql.Post)

	app.Run(fmt.Sprintf(":%d", opts.port))
}
