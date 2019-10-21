package cmd

import (
	"net/http"
	"os"

	"github.com/equinor/seismic-cloud/api/events"
	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/equinor/seismic-cloud/api/service"

	"github.com/equinor/seismic-cloud/api/config"
	"github.com/equinor/seismic-cloud/api/server"
	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/pkg/profile"
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
	if len(sCmd) > 0 && len(sCmd[0]) > 0 {
		return sCmd
	}
	return nil
}

func surfaceStoreConfig() interface{} {
	if len(config.AzStorageAccount()) > 0 && len(config.AzStorageKey()) > 0 {
		return store.AzureBlobSettings{
			AccountName:   config.AzStorageAccount(),
			AccountKey:    config.AzStorageKey(),
			ContainerName: config.AzContainerName(),
		}

	}

	if len(config.LocalSurfacePath()) > 0 {
		return config.LocalSurfacePath()

	}

	return make(map[string][]byte)

}

func manifestStoreConfig() interface{} {
	if len(config.ManifestURI()) > len("mongodb://") {
		return store.ConnStr(config.ManifestURI())

	}

	if len(config.ManifestStoragePath()) > 0 {
		return config.ManifestStoragePath()

	}

	return nil

}

func createHTTPServerOptions() ([]server.HttpServerOption, error) {
	op := "serve.getHttpServerOptions"

	opts := make([]server.HttpServerOption, 0)

	if config.UseAuth() {
		opts = append(opts,
			server.WithOAuth2(config.AuthServer(), config.ResourceID(), config.Issuer()))
	}

	if ms, err := store.NewManifestStore(manifestStoreConfig()); err != nil {

		return nil, events.E(events.Op(op), "Accessing manifest store", err)
	} else {
		opts = append(
			opts,
			server.WithManifestStore(ms))
	}

	if ssC, err := store.NewSurfaceStore(surfaceStoreConfig()); err != nil {
		return nil, events.E(events.Op(op), "Accessing surface store", err)

	} else {
		opts = append(
			opts,
			server.WithSurfaceStore(ssC))
	}

	if st, err := service.NewStitch(stitchConfig(), config.Profiling()); err != nil {
		return nil, events.E(events.Op(op), "Stitch tcp error", err)

	} else {
		opts = append(
			opts,
			server.WithStitcher(st))
	}

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

	return opts, nil
}

func runServe(cmd *cobra.Command, args []string) {
	op := "serve.runServe"

	if viper.ConfigFileUsed() == "" {
		viper.AutomaticEnv()
		l.LogW(op, "Config from environment variables")
	} else {
		if err := config.Load(); err == nil {
			l.LogI("root.initConfig", "Config loaded and validated "+viper.ConfigFileUsed())
		}
		l.LogI(op, "Config loaded and validated "+viper.ConfigFileUsed())
	}

	var p *profile.Profile
	if config.Profiling() {
		l.LogI(op, "Enabling profiling")
		pType := "mem"
		pOpts := []func(*profile.Profile){
			profile.ProfilePath("pprof"),
			profile.NoShutdownHook,
		}
		switch pType {
		case "mem":
			pOpts = append(pOpts, profile.MemProfile)
		case "cpu":
			pOpts = append(pOpts, profile.CPUProfile)
		default:
			pOpts = append(pOpts, profile.CPUProfile)
		}

		p = profile.Start(pOpts...).(*profile.Profile)
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

	opts, err := createHTTPServerOptions()
	if err != nil {
		l.LogE(op, "Error creating http server options", err)
		os.Exit(1)
	}

	hs, err := server.NewHttpServer(opts...)

	if err != nil {
		l.LogE(op, "Error configuring http server", err)
		os.Exit(1)
	}
	err = hs.Serve()

	if err != nil && err != http.ErrServerClosed {
		l.LogE(op, "Error running http server", err)
		os.Exit(1)
	}

	if p != nil {
		p.Stop()
	}

}

func init() {
	rootCmd.AddCommand(serveCmd)
}
