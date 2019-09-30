package cmd

import (
	"os"

	"github.com/equinor/seismic-cloud/api/events"
	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/equinor/seismic-cloud/api/service"

	"github.com/equinor/seismic-cloud/api/config"
	"github.com/equinor/seismic-cloud/api/server"
	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "serve seismic cloud provider",
	Long:  `serve seismic cloud provider.`,
	Run:   runServe,
}

func stitchConfig() interface{} {
	sCmd := config.StitchCmd()
	sTCP := config.StitchTcpAddr()
	sGRPC := config.StitchGrpcAddr()
	if len(sGRPC) > 0 {

		return service.GrpcOpts{Addr: sGRPC, Insecure: true}
	}
	if len(sTCP) > 0 {
		return service.TcpAddr(sTCP)
	}
	if len(sCmd) > 0 {
		return sCmd
	}
	return nil
}

func runServe(cmd *cobra.Command, args []string) {
	op := "serve.runServe"
	if viper.ConfigFileUsed() == "" {
		l.LogW(op, "Config from environment variables")
	} else {
		l.LogI(op, "Config loaded and validated "+viper.ConfigFileUsed())
	}

	if len(config.LogDBConnStr()) > 0 {
		l.LogI(op, "Switch log sink from os.Stdout to psqlDB")
		db, err := l.DbOpener()
		if err != nil {
			l.LogE(op, "Unable to connect to log db", err)
			return
		}
		l.SetLogSink(db, events.DebugLevel)

	}

	opts := make([]server.HttpServerOption, 0)

	if config.UseAuth() {
		opts = append(opts,
			server.WithOAuth2(config.AuthServer(), config.ResourceID(), config.Issuer()))
	}

	if config.ManifestSrc() == "db" {
		opts = append(opts,
			server.WithManifestStore(&store.ManifestDbStore{
				ConnString: config.ManifestURI()}))
	} else if config.ManifestSrc() == "path" {
		opts = append(opts,
			server.WithManifestStore(&store.ManifestFileStore{
				BasePath: config.ManifestStoragePath()}))
	}

	if len(config.AzStorageAccount()) > 0 && len(config.AzStorageKey()) > 0 {
		azure, err := store.NewAzBlobStorage(
			config.AzStorageAccount(),
			config.AzStorageKey(),
			config.AzContainerName())
		if err != nil {
			l.LogE(op, "New azure blob storage error", err)

			os.Exit(1)
		}
		opts = append(opts, server.WithSurfaceStore(azure))
	} else if len(config.LocalSurfacePath()) > 0 {
		local, err := store.NewLocalStorage(config.LocalSurfacePath())
		if err != nil {
			l.LogE(op, "New local directory error", err)
			os.Exit(1)
		}
		opts = append(opts, server.WithSurfaceStore(local))
	}

	st, err := service.NewStitch(stitchConfig(), config.Profiling())
	if err != nil {
		l.LogE(op, "Stitch tcp error", err)
		os.Exit(1)
	}
	opts = append(
		opts,
		server.WithStitcher(st))

	if len(config.HostAddr()) > 0 {
		opts = append(
			opts,
			server.WithHostAddr(config.HostAddr()))
	}

	if config.HttpOnly() {
		opts = append(
			opts,
			server.WithHttpOnly())
	}

	if config.UseLetsEncrypt() {
		opts = append(
			opts,
			server.WithLetsEncrypt(config.DomainList(), config.DomainMail()))
	}

	if config.UseTLS() {
		opts = append(
			opts,
			server.WithTLS(config.CertFile(), config.KeyFile()))
	}

	if config.Profiling() {
		opts = append(
			opts,
			server.WithProfiling())
	}

	if config.Swagger() {
		opts = append(
			opts,
			server.WithSwagger())
	}

	hs, err := server.NewHttpServer(opts...)

	if err != nil {
		l.LogE(op, "Error starting http server", err)
		os.Exit(1)
	}
	err = hs.Serve()

	if err != nil {
		l.LogE(op, "Error starting http server", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
