package main

import (
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
	clientID     string
	storageURL   string
	redisURL     string
	signkey      string
}

func parseopts() opts {
	help := getopt.BoolLong("help", 0, "print this help text")
	opts := opts {
		clientID:     os.Getenv("CLIENT_ID"),
		storageURL:   os.Getenv("STORAGE_URL"),
		redisURL:     os.Getenv("REDIS_URL"),
		signkey:      os.Getenv("SIGN_KEY"),
	}

	getopt.FlagLong(
		&opts.clientID,
		"client-id",
		0,
		"Client ID for on-behalf tokens",
		"id",
	)
	getopt.FlagLong(
		&opts.storageURL,
		"storage-url",
		0,
		"Storage URL, e.g. https://<account>.blob.core.windows.net",
		"url",
	)
	getopt.FlagLong(
		&opts.redisURL,
		"redis-url",
		0,
		"Redis URL",
		"url",
	)
	getopt.FlagLong(
		&opts.signkey,
		"sign-key",
		0,
		"Signing key used for response authorization tokens",
		"key",
	)

	getopt.Parse()
	if *help {
		getopt.Usage()
		os.Exit(0)
	}

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
	cmdable := redis.NewClient(
		&redis.Options {
			Addr: opts.redisURL,
			DB: 0,
		},
	)
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
	app.Run(":8080")
}
