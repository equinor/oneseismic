package main

import (
	"fmt"
	"log"
	"os"

	"github.com/equinor/seismic-cloud/api/cmd"
	l "github.com/equinor/seismic-cloud/api/logger"
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
		"AZURE_MANIFEST_CONTAINER",
		"HOST_ADDR",
		"ISSUER",
		"LOGDB_CONNSTR",
		"MANIFEST_DB_URI",
		"MANIFEST_PATH",
		"NO_AUTH",
		"PROFILING",
		"RESOURCE_ID",
		"STITCH_GRPC_ADDR",
	}

	for _, env := range envs {
		m[env] = os.Getenv(env)
	}

	return m
}

//@title Seismic Cloud Api
//@description The Seismic Cloud Api
//@license.name proprietary
//@contact.name Equinor
//@securityDefinitions.apikey ApiKeyAuth
//@in header
//@name Authorization

//@tag.name manifest
//@tag.description Operations for manifests
//@tag.name stitch
//@tag.description Stitch together cube data
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
		os.Exit(1)
	}

}
