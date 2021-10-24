package main

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/pborman/getopt/v2"

	"github.com/equinor/oneseismic/api/api"
	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/util"
)

type opts struct {
	broker  string
	signkey string
}

func parseopts() opts {
	help := getopt.BoolLong("help", 0, "print this help text")
	opts := opts {
		broker:  os.Getenv("BROKER"),
		signkey: os.Getenv("SIGN_KEY"),
	}

	getopt.FlagLong(
		&opts.signkey,
		"sign-key",
		0,
		"Signing key used for response authorization tokens. " +
			"Must match signing key in api/query",
		"key",
	)

	getopt.FlagLong(
		&opts.broker,
		"broker",
		0,
		"Message broker (redis) URL",
		"url",
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

	keyring := auth.MakeKeyring([]byte(opts.signkey))
	result  := api.Result {
		Timeout: time.Second * 15,
		Storage: redis.NewClient(&redis.Options {
			Addr: opts.broker,
			DB: 0,
		}),
		Keyring: &keyring,
	}

	app := gin.Default()
	results := app.Group("/result")
	results.Use(auth.ResultAuth(&keyring))
	results.Use(util.Compression())
	results.GET("/:pid", result.Get)
	results.GET("/:pid/stream", result.Stream)
	results.GET("/:pid/status", result.Status)
	app.Run(":8080")
}
