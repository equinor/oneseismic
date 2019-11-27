package cmd

import (
	"net/http"
	"os"

	"github.com/equinor/seismic-cloud-api/api/events"
	l "github.com/equinor/seismic-cloud-api/api/logger"
	"github.com/equinor/seismic-cloud-api/api/service"

	"github.com/equinor/seismic-cloud-api/api/config"
	"github.com/equinor/seismic-cloud-api/api/server"
	"github.com/equinor/seismic-cloud-api/api/service/store"
	"github.com/pkg/profile"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "serve seismic cloud provider",
	Long:  `serve seismic cloud provider.`,
	Run:   runServe,
}

func stitchConfig() interface{} {
	sGRPC := config.StitchGrpcAddr()
	if len(sGRPC) > 0 {
		return service.GrpcOpts{Addr: sGRPC, Insecure: true}
	}
	return nil
}

func surfaceStoreConfig() interface{} {
	if len(config.AzStorageAccount()) > 0 && len(config.AzStorageKey()) > 0 {
		return store.AzureBlobSettings{
			AccountName:   config.AzStorageAccount(),
			AccountKey:    config.AzStorageKey(),
			ContainerName: config.AzSurfaceContainerName(),
		}
	}

	if len(config.LocalSurfacePath()) > 0 {
		return store.BasePath(config.LocalSurfacePath())
	}

	return make(map[string][]byte)

}

func manifestStoreConfig() interface{} {

	if len(config.AzStorageAccount()) > 0 && len(config.AzStorageKey()) > 0 {
		return store.AzureBlobSettings{
			AccountName:   config.AzStorageAccount(),
			AccountKey:    config.AzStorageKey(),
			ContainerName: config.AzManifestContainerName(),
		}
	}

	if len(config.ManifestDbURI()) > len("mongodb://") {
		return store.ConnStr(config.ManifestDbURI())
	}

	if len(config.ManifestStoragePath()) > 0 {
		return store.BasePath(config.ManifestStoragePath())
	}

	return nil

}

func createHTTPServerOptions() ([]server.HTTPServerOption, error) {
	op := "serve.getHTTPServerOptions"

	opts := make([]server.HTTPServerOption, 0)
	opts = append(opts, server.WithAPIVersion(config.Version()))

	if config.UseAuth() {
		opts = append(opts,
			server.WithOAuth2(server.OAuth2Option{
				AuthServer: config.AuthServer(),
				Audience:   config.ResourceID(),
				Issuer:     config.Issuer(),
				ApiSecret:  []byte(config.ApiSecret()),
			}))
	}

	ms, err := store.NewManifestStore(manifestStoreConfig())
	if err != nil {
		return nil, events.E(events.Op(op), "Accessing manifest store", err)
	}
	opts = append(
		opts,
		server.WithManifestStore(ms))

	ssC, err := store.NewSurfaceStore(surfaceStoreConfig())
	if err != nil {
		return nil, events.E(events.Op(op), "Accessing surface store", err)

	}
	opts = append(
		opts,
		server.WithSurfaceStore(ssC))

	st, err := service.NewStitch(stitchConfig(), config.Profiling())
	if err != nil {
		return nil, events.E(events.Op(op), "Stitch error", err)

	}
	opts = append(
		opts,
		server.WithStitcher(st))

	if len(config.HostAddr()) > 0 {
		opts = append(
			opts,
			server.WithHostAddr(config.HostAddr()))
	}

	if config.HTTPOnly() {
		opts = append(
			opts,
			server.WithHTTPOnly())
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

func serve(opts []server.HTTPServerOption) error {
	op := "serve.serve"
	hs, err := server.NewHTTPServer(opts...)

	if err != nil {
		return events.E(events.Op(op), "Error configuring http server", err)
	}
	err = hs.Serve()

	if err != nil && err != http.ErrServerClosed {
		return events.E(events.Op(op), "Error running http server", err)
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
		db, err := l.DbOpener(config.LogDBConnStr())
		if err != nil {
			l.LogE(op, "Opening db", err)
			return
		}
		err = l.SetLogSink(db, events.DebugLevel)
		if err != nil {
			l.LogE(op, "Switching log sink", err)
			return
		}

	}

	opts, err := createHTTPServerOptions()
	if err != nil {
		l.LogE(op, "Creating http server options", err)
		os.Exit(1)
	}

	err = serve(opts)
	if err != nil {
		l.LogE(op, "Error starting http server", err)
	}

	if p != nil {
		p.Stop()
	}

}

func init() {
	rootCmd.AddCommand(serveCmd)
}
