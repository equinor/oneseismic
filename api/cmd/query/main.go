package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/equinor/oneseismic/api/api"
	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/pborman/getopt/v2"
)

type opts struct {
	clientID          string
	storageURL        string
	redisURL          string
	redisPassword     string
	secureConnections bool
	signkey           string
	port              string
}

func parseopts() opts {
	help := getopt.BoolLong("help", 0, "print this help text")
	opts := opts{
		clientID:      os.Getenv("CLIENT_ID"),
		storageURL:    os.Getenv("STORAGE_URL"),
		redisURL:      os.Getenv("REDIS_URL"),
		redisPassword: os.Getenv("REDIS_PASSWORD"),
		signkey:       os.Getenv("SIGN_KEY"),
	}

	getopt.FlagLong(
		&opts.clientID,
		"client-id",
		0,
		"Client ID for on-behalf tokens",
		"string",
	)
	getopt.FlagLong(
		&opts.storageURL,
		"storage-url",
		0,
		"Storage URL, e.g. https://<account>.blob.core.windows.net",
		"string",
	)
	getopt.FlagLong(
		&opts.redisURL,
		"redis-url",
		0,
		"Redis URL (host:port)",
		"string",
	)
	getopt.FlagLong(
		&opts.redisPassword,
		"redis-password",
		0,
		"Redis password. Empty by default",
		"int",
	)
	secureConnections := getopt.BoolLong(
		"secureConnections",
		0,
		"Connect to Redis securely",
	)
	getopt.FlagLong(
		&opts.signkey,
		"sign-key",
		0,
		"Signing key used for response authorization tokens",
		"string",
	)
	opts.port = "8080"
	getopt.FlagLong(
		&opts.port,
		"port",
		0,
		"Port to start server on. Defaults to 8080",
	)

	getopt.Parse()
	if *help {
		getopt.Usage()
		os.Exit(0)
	}

	opts.secureConnections = *secureConnections
	return opts
}

/*
 * Configuration for this instance of oneseismic for user-controlled clients
 *
 * Oneseismic does not really have a good concept of logged in users, sessions
 * etc. Rather, oneseismic gets tokens (in the Authorization header) or query
 * parameters (shared access signatures) that are to query blob storage. Users
 * can obtain such tokens or signatures as they see fit.
 *
 * The clientconfig struct and the /config endpoint are meant for sharing
 * oneseismic instance and company specific configurations with clients. While
 * only auth stuff is included now, it's a natural place to add more client
 * configuration parameters later e.g. performance hints, max/min latency.
 */
type clientconfig struct {
	appid      string
	scopes     []string
	defaultStorageResource string
}

func (c *clientconfig) Get(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H {
		/*
		 * oneseismic's app-id
		 */
		"client_id": c.appid,
		/*
		 * The scopes (permissions) that oneseismic requests in order to
		 * function
		 */
		"scopes": c.scopes,

		/*
		 * The default storage account resource URL, e.g.
		 * https://<acc>.blob.core.windows.net. While most of the oneseismic
		 * infrastructure doesn't mandate it, it will be overwhelmingly likely
		 * that one oneseismic instance maps to a single storage account.
		 * Having the "backing resource" programmatically available to users
		 * makes for pretty programs, since it is sufficient to specify the
		 * oneseismic instance and query the rest from there.
		 */
		"default-storage-resource": c.defaultStorageResource,
	})
}

func main() {
	opts := parseopts()

	keyring := auth.MakeKeyring([]byte(opts.signkey))
	redisOptions := &redis.Options{
		Addr:     opts.redisURL,
		Password: opts.redisPassword,
		DB:       0,
	}

	if opts.secureConnections {
		redisOptions.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}
	cmdable := redis.NewClient(redisOptions)

	scheduler := api.NewScheduler(cmdable)
	gql := api.MakeGraphQL(&keyring, opts.storageURL, scheduler)

	cfg := clientconfig {
		appid: opts.clientID,
		scopes: []string{
			fmt.Sprintf("api://%s/One.Read", opts.clientID),
		},
		defaultStorageResource: opts.storageURL,
	}

	app := gin.Default()
	
	graphql := app.Group("/graphql")
	graphql.Use(util.GeneratePID)
	graphql.GET( "", gql.Get)
	graphql.POST("", gql.Post)

	app.GET("/config", cfg.Get)
	app.Run(fmt.Sprintf(":%s", opts.port))
}
