package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/equinor/oneseismic/api/auth"
	_ "github.com/equinor/oneseismic/api/docs"
	"github.com/equinor/oneseismic/api/events"
	"github.com/equinor/oneseismic/api/logger"
	"github.com/equinor/oneseismic/api/profiling"
	"github.com/equinor/oneseismic/api/server"
	"github.com/joho/godotenv"
	"github.com/pkg/profile"
)

func init() {
	godotenv.Load() // nolint, silently ignore missing or invalid .env
}

//@title oneseismic
//@description oneseismic
//@license.name AGPL3
//@contact.name Equinor
//@securityDefinitions.apikey ApiKeyAuth
//@in header
//@name Authorization
func main() {
	c, err := getConfig()
	if err != nil {
		log.Fatal("Failed to load config", err)
	}

	if c.Profiling {
		var p *profile.Profile
		pOpts := []func(*profile.Profile){
			profile.ProfilePath("pprof"),
			profile.NoShutdownHook,
		}

		pOpts = append(pOpts, profile.MemProfile)
		p = profile.Start(pOpts...).(*profile.Profile)
		profiling.ServeMetrics("8081")
		defer p.Stop()
	}

	err = logToDB(c.LogDBConnStr)
	if err != nil {
		log.Fatalf("failed to log to db: %v", err)
	}

	c.SigKeySet, err = auth.GetOIDCKeySet(c.AuthServer)
	if err != nil {
		log.Fatalf("could not get keyset: %v", err)
	}

	err = server.Serve(c)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func logToDB(logDBConnStr string) error {

	if len(logDBConnStr) > 0 {
		logger.LogI("switch log sink from os.Stdout to psqlDB")

		err := logger.SetLogSink(logger.ConnString(logDBConnStr), events.DebugLevel)
		if err != nil {
			return fmt.Errorf("switching log sink: %w", err)
		}
	}

	return nil
}

func getConfig() (*server.Config, error) {
	authServer, err := url.ParseRequestURI(os.Getenv("AUTHSERVER"))
	if err != nil {
		return nil, fmt.Errorf("invalid AUTHSERVER: %w", err)
	}

	profiling, err := strconv.ParseBool(os.Getenv("PROFILING"))
	if err != nil {
		return nil, fmt.Errorf("could not parse PROFILING: %w", err)
	}

	conf := &server.Config{
		AuthServer:   authServer,
		APISecret:    []byte(os.Getenv("API_SECRET")),
		Issuer:       os.Getenv("ISSUER"),
		StorageURL:   strings.ReplaceAll(os.Getenv("AZURE_STORAGE_URL"), "{}", "%s"),
		HostAddr:     os.Getenv("HOST_ADDR"),
		LogDBConnStr: os.Getenv("LOGDB_CONNSTR"),
		LogLevel:     os.Getenv("LOG_LEVEL"),
		Profiling:    profiling,
		ZmqRepAddr:   os.Getenv("ZMQ_REP_ADDR"),
		ZmqReqAddr:   os.Getenv("ZMQ_REQ_ADDR"),
	}

	return conf, nil
}
