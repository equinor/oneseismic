package main

import (
	"log"
	"os"

	"github.com/equinor/oneseismic/api/cmd"
	"github.com/joho/godotenv"
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
	err := cmd.Serve(getEnvs())
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

}
