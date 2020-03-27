package main

import (
	"fmt"
	"log"
	"os"

	"github.com/equinor/oneseismic/api/cmd"
	l "github.com/equinor/oneseismic/api/logger"
	"github.com/joho/godotenv"
	jww "github.com/spf13/jwalterweatherman"
)

func init() {
	godotenv.Load() // nolint, silently ignore missing or invalid .env
	jww.SetStdoutThreshold(jww.LevelFatal)
	log.SetPrefix("[INFO] ")
	l.AddLoggerSource("main.log", log.SetOutput)
	l.AddLoggerSource("setup.log", jww.SetLogOutput)
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
		"STITCH_GRPC_ADDR",
	}

	for _, env := range envs {
		m[env] = os.Getenv(env)
	}

	return m
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in main", r)
		}
		l.Wait()

	}()
	err := cmd.Serve(getEnvs())
	if err != nil {
		l.LogE("Failed to start server", err)
		l.Wait()
		os.Exit(1)
	}

}
