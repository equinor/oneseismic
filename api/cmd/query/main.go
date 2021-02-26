package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/equinor/oneseismic/api/api"
	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/util"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/pborman/getopt/v2"
)

type opts struct {
	authserver   string
	audience     string
	clientID     string
	clientSecret string
	storageURL   string
	redisURL     string
	bind         string
	signkey      string
}

func parseopts() opts {
	help := getopt.BoolLong("help", 0, "print this help text")
	opts := opts {
		authserver:   os.Getenv("AUTHSERVER"),
		audience:     os.Getenv("AUDIENCE"),
		clientID:     os.Getenv("CLIENT_ID"),
		clientSecret: os.Getenv("CLIENT_SECRET"),
		storageURL:   os.Getenv("STORAGE_URL"),
		redisURL:     os.Getenv("REDIS_URL"),
		signkey:      os.Getenv("SIGN_KEY"),
	}

	getopt.FlagLong(
		&opts.authserver,
		"authserver",
		0,
		"OpenID Connect discovery server",
		"addr",
	)
	getopt.FlagLong(
		&opts.audience,
		"audience",
		0,
		"Audience for token validation",
		"audience",
	)
	getopt.FlagLong(
		&opts.clientID,
		"client-id",
		0,
		"Client ID for on-behalf tokens",
		"id",
	)
	getopt.FlagLong(
		&opts.clientSecret,
		"client-secret",
		0,
		"Client ID for on-behalf tokens",
		"secret",
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
		&opts.bind,
		"bind",
		0,
		"Bind URL e.g. tcp://*:port",
		"addr",
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
 * etc. Rather, oneseismic gets tokens (in the Authorization header) which it
 * uses to obtain on-behalf-of tokens that in turn are used to query blob
 * storage. Users can use the python libraries to "log in", i.e. obtain a token
 * for their AD-registered user, constructed in a way that gives oneseismic the
 * permission to perform (blob) requests on their behalf [1].
 *
 * In order to construct a token that allows oneseismic to make requests, the
 * app-id of oneseismic must be available somehow. This app-id, sometimes
 * called client-id, is public information and for web apps often coded into
 * the javascript and ultimately delivered from server-side. Conceptually, this
 * should be no different. Oneseismic is largely designed to support multiple
 * deployments, so hard-coding an app id is probably not a good idea. Forcing
 * users to store or memorize the app-id and auth-server for use with the
 * python3 oneiseismic.login module is also not a good solution.
 *
 * The microsoft authentication library (MSAL) [2] is pretty clear on wanting a
 * client-id for obtaining a token. When the oneseismic python library is used,
 * it is an extension of the instance it's trying to reach, so getting the
 * app-id and authorization server [3] from a specific setup seems pretty
 * reasonable.
 *
 * The clientconfig struct and the /config endpoint are meant for sharing
 * oneseismic instance and company specific configurations with clients. While
 * only auth stuff is included now, it's a natural place to add more client
 * configuration parameters later e.g. performance hints, max/min latency.
 *
 * [1] https://docs.microsoft.com/en-us/graph/auth-v2-user
 * [2] https://msal-python.readthedocs.io/en/latest/#msal.PublicClientApplication
 * [3] usually https://login.microsoftonline.com/<tenant-id>
 *
 * https://docs.microsoft.com/en-us/azure/storage/common/storage-auth-aad-app
 */
type clientconfig struct {
	appid      string
	authority  string
	scopes     []string
}

func (c *clientconfig) Get(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H {
		/*
		 * oneseismic's app-id
		 */
		"client_id": c.appid,
		/*
		 * URL for the token authority. Usually
		 * https://login.microsoftonline.com/<tenant>
		 */
		"authority": c.authority,
		/*
		 * The scopes (permissions) that oneseismic requests in order to
		 * function
		 */
		"scopes": c.scopes,
	})
}

func main() {
	opts := parseopts()
	httpclient := http.Client {
		Timeout: 10 * time.Second,
	}
	openidcfg, err := auth.GetOpenIDConfig(
		&httpclient,
		opts.authserver + "/v2.0/.well-known/openid-configuration",
	)
	if err != nil {
		log.Fatalf("Unable to get OpenID keyset: %v", err)
	}

	keyring := auth.MakeKeyring([]byte(opts.signkey))
	cmdable := redis.NewClient(
		&redis.Options {
			Addr: opts.redisURL,
			DB: 0,
		},
	)
	slice := api.MakeSlice(&keyring, opts.storageURL, cmdable)
	result := api.Result {
		Timeout: time.Second * 15,
		StorageURL: opts.storageURL,
		Storage: redis.NewClient(&redis.Options {
			Addr: opts.redisURL,
			DB: 0,
		}),
		Keyring: &keyring,
	}

	cfg := clientconfig {
		appid: opts.clientID,
		authority: opts.authserver,
		scopes: []string{
			fmt.Sprintf("api://%s/One.Read", opts.clientID),
		},
	}

	app := gin.Default()
	
	validate := auth.ValidateJWT(openidcfg.Jwks, openidcfg.Issuer, opts.audience)
	tokens   := auth.NewTokens(openidcfg.TokenEndpoint, opts.clientID, opts.clientSecret)
	onbehalf := auth.OnBehalfOf(tokens)
	queries := app.Group("/query")
	queries.Use(validate)
	queries.Use(onbehalf)
	queries.Use(util.GeneratePID)
	queries.Use(util.QueryLogger)
	queries.GET("/", slice.List)
	queries.GET("/:guid", slice.Entry)
	queries.GET("/:guid/slice/:dimension/:lineno", slice.Get)

	results := app.Group("/result")
	results.Use(auth.ResultAuth(&keyring))
	results.GET("/:pid", result.Get)
	results.GET("/:pid/stream", result.Stream)
	results.GET("/:pid/status", result.Status)

	app.GET("/config", cfg.Get)
	app.Run(":8080")
}
