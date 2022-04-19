package main

import (
	"crypto/tls"
	"os"
	"time"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/pborman/getopt/v2"

	"github.com/equinor/oneseismic/api/api"
	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/util"
)

type opts struct {
	redisURL          string
	redisPassword     string
	secureConnections bool
	signkey           string
	port              string
}

func parseopts() opts {
	help := getopt.BoolLong("help", 0, "print this help text")
	opts := opts{
		redisURL:      os.Getenv("REDIS_URL"),
		redisPassword: os.Getenv("REDIS_PASSWORD"),
		signkey:       os.Getenv("SIGN_KEY"),
	}

	getopt.FlagLong(
		&opts.signkey,
		"sign-key",
		0,
		"Signing key used for response authorization tokens. " +
			"Must match signing key in api/query",
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
		'P',
		"Redis password. Empty by default",
		"string",
	)
	secureConnections := getopt.BoolLong(
		"secureConnections",
		0,
		"Connect to Redis securely",
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

	result := api.Result{
		Timeout: time.Second * 15,
		Storage: redis.NewClient(redisOptions),
		Keyring: &keyring,
	}

	app := gin.Default()
	results := app.Group("/result")
	results.Use(auth.ResultAuth(&keyring))
	results.Use(util.Compression())
	results.GET("/:pid", result.Get)
	results.GET("/:pid/stream", result.Stream)
	results.GET("/:pid/status", result.Status)
	app.Run(fmt.Sprintf(":%s", opts.port))
}
