package cmd

import (
	"log"
	"os"
	"testing"

	"github.com/equinor/seismic-cloud/api/events"
	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/equinor/seismic-cloud/api/server"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
)

func Test_ttt(t *testing.T) {
	t.Errorf("XXX")
}

func TestMain(m *testing.M) {
	l.SetLogSink(os.Stdout, events.DebugLevel)
	jww.SetStdoutThreshold(jww.LevelFatal)
	log.SetPrefix("[INFO] ")
	l.AddLoggerSource("main.log", log.SetOutput)
	l.AddLoggerSource("setup.log", jww.SetLogOutput)

	c := make(chan bool)
	go func(c chan bool) {
		select {
		case <-c:
			os.Exit(0)
		default:
			startServer()
		}

	}(c)
	code := m.Run()
	c <- true
	os.Exit(code)
}

func testConfig() {
	viper.Set("host_addr", "localhost:8080")
	viper.Set("http_only", "true")
	viper.Set("letsencrypt", "false")
	viper.Set("local_surface_path", "tmp/")
	viper.Set("logdb_connstr", "")
	viper.Set("manifest_path", "tmp/")
	viper.Set("manifest_src", "path")
	viper.Set("no_auth", "true")
	viper.Set("profiling", "true")
	viper.Set("resource_id", "")
	viper.Set("STITCH_CMD", "/bin/cat")
}

func startServer() (*server.HttpServer, error) {
	viper.Reset()
	testConfig()
	opts, _ := createHTTPServerOptions()
	hs, err := startServe(opts...)
	return hs, err
}
