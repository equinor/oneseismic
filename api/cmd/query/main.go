package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/equinor/oneseismic/api/api"
	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"github.com/namsral/flag"
	"github.com/pebbe/zmq4"
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

func parseopts() (opts, error) {
	type option struct {
		param *string
		flag  string
		help  string
	}

	opts := opts {}
	params := []option {
		option {
			param: &opts.authserver,
			flag: "authserver",
			help: "OpenID Connect discovery server",
		},
		option {
			param: &opts.audience,
			flag: "audience",
			help: "Audience",
		},
		option {
			param: &opts.clientID,
			flag: "client-id",
			help: "Client ID",
		},
		option {
			param: &opts.clientSecret,
			flag: "client-secret",
			help: "Client Secret",
		},
		option {
			param: &opts.storageURL,
			flag: "storage-url",
			help: "Storage URL",
		},
		option {
			param: &opts.redisURL,
			flag: "redis-url",
			help: "Redis URL",
		},
		option {
			param: &opts.bind,
			flag: "bind",
			help: "Bind URL e.g. tcp://*:port",
		},
		option {
			param: &opts.signkey,
			flag:  "sign-key",
			help:  "Signing key used for response authorization tokens",
		},
	}

	for _, opt := range params {
		flag.StringVar(opt.param, opt.flag, "", opt.help)
	}
	flag.Parse()
	for _, opt := range params {
		if *opt.param == "" {
			return opts, fmt.Errorf("%s not set", opt.flag)
		}
	}

	return opts, nil
}

func main() {
	opts, err := parseopts()
	if err != nil {
		log.Fatalf("Unable to start server: %v", err)
	}

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

	out, err := zmq4.NewSocket(zmq4.PUSH)
	if err != nil {
		log.Fatalf("Unable to create socket: %v", err)
	}
	err = out.Bind(opts.bind)
	if err != nil {
		log.Fatalf("Unable to bind queue to %s: %v", opts.bind, err)
	}
	defer out.Close()

	keyring := auth.MakeKeyring([]byte(opts.signkey))
	slice := api.MakeSlice(&keyring, opts.storageURL, out)
	result := api.Result {
		Timeout: time.Second * 15,
		StorageURL: opts.storageURL,
		Storage: redis.NewClient(&redis.Options {
			Addr: opts.redisURL,
			DB: 0,
		}),
		Keyring: &keyring,
	}

	validate := auth.ValidateJWT(openidcfg.Jwks, openidcfg.Issuer, opts.audience)
	onbehalf := auth.OnBehalfOf(openidcfg.TokenEndpoint, opts.clientID, opts.clientSecret)
	app := gin.Default()
	app.GET(
		"/query/:guid/slice/:dimension/:lineno",
		validate,
		onbehalf,
		slice.Get,
	)
	app.GET("/result/:pid", result.Get)
	app.Run(":8080")
}
