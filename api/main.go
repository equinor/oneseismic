package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/equinor/oneseismic/api/server"
	"github.com/joho/godotenv"
	"github.com/pkg/profile"
)

func init() {
	godotenv.Load() // nolint, silently ignore missing or invalid .env
}

func getEnvs() map[string]string {
	m := make(map[string]string)

	envs := [...]string{
		"API_SECRET",
		"AUTHSERVER",
		"AZURE_STORAGE_URL",
		"AZURE_STORAGE_ACCOUNT",
		"AZURE_STORAGE_ACCESS_KEY",
		"HOST_ADDR",
		"ISSUER",
		"LOGDB_CONNSTR",
		"PROFILING",
		"RESOURCE_ID",
	}

	for _, env := range envs {
		m[env] = os.Getenv(env)
	}

	return m
}

func main() {
	profiling, err := strconv.ParseBool(m["PROFILING"])
	if err != nil {
		log.Fatalf("could not parse PROFILING: %v", err)
	}

	var p *profile.Profile

	if profiling {
		pOpts := []func(*profile.Profile){
			profile.ProfilePath("pprof"),
			profile.NoShutdownHook,
		}

		pOpts = append(pOpts, profile.MemProfile)
		p = profile.Start(pOpts...).(*profile.Profile)

		defer p.Stop()
	}

	err = server.Serve(m)
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("error running http server: %v", err)
	}
}
