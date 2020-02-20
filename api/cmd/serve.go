package cmd

import (
	"net/http"

	"github.com/equinor/seismic-cloud/api/events"
	l "github.com/equinor/seismic-cloud/api/logger"
	"github.com/equinor/seismic-cloud/api/service"

	"github.com/equinor/seismic-cloud/api/server"
	"github.com/equinor/seismic-cloud/api/service/store"
	"github.com/pkg/profile"
)

func stitchConfig() interface{} {
	sGRPC := StitchGrpcAddr()
	if len(sGRPC) > 0 {
		return service.GrpcOpts{Addr: sGRPC, Insecure: true}
	}
	return nil
}

func surfaceStoreConfig() interface{} {
	if len(AzStorageAccount()) > 0 && len(AzStorageKey()) > 0 {
		return store.AzureBlobSettings{
			StorageURL:    AzStorageURL(),
			AccountName:   AzStorageAccount(),
			AccountKey:    AzStorageKey(),
			ContainerName: AzSurfaceContainerName(),
		}
	}

	if len(LocalSurfacePath()) > 0 {
		return store.BasePath(LocalSurfacePath())
	}

	return make(map[string][]byte)

}

func manifestStoreConfig() interface{} {

	if len(AzStorageAccount()) > 0 && len(AzStorageKey()) > 0 {
		return store.AzureBlobSettings{
			StorageURL:    AzStorageURL(),
			AccountName:   AzStorageAccount(),
			AccountKey:    AzStorageKey(),
			ContainerName: AzManifestContainerName(),
		}
	}

	if len(ManifestDbURI()) > len("mongodb://") {
		return store.ConnStr(ManifestDbURI())
	}

	if len(ManifestStoragePath()) > 0 {
		return store.BasePath(ManifestStoragePath())
	}

	return nil

}

func createHTTPServerOptions() ([]server.HTTPServerOption, error) {
	opts := make([]server.HTTPServerOption, 0)

	if UseAuth() {
		authServer, err := AuthServer()
		if err != nil {
			return nil, events.E("AuthServer config", err)
		}
		opts = append(opts,
			server.WithOAuth2(server.OAuth2Option{
				AuthServer: authServer,
				Audience:   ResourceID(),
				Issuer:     Issuer(),
				ApiSecret:  []byte(ApiSecret()),
			}))
	}

	ms, err := store.NewManifestStore(manifestStoreConfig())
	if err != nil {
		return nil, events.E("Accessing manifest store", err)
	}
	opts = append(
		opts,
		server.WithManifestStore(ms))

	ssC, err := store.NewSurfaceStore(surfaceStoreConfig())
	if err != nil {
		return nil, events.E("Accessing surface store", err)
	}
	opts = append(
		opts,
		server.WithSurfaceStore(ssC))

	st, err := service.NewStitch(stitchConfig())
	if err != nil {
		return nil, events.E("Stitch error", err)
	}
	opts = append(
		opts,
		server.WithStitcher(st))

	if len(HostAddr()) > 0 {
		opts = append(
			opts,
			server.WithHostAddr(HostAddr()))
	}

	if Profiling() {
		opts = append(
			opts,
			server.WithProfiling())
	}

	return opts, nil
}

func serve(opts []server.HTTPServerOption) error {

	hs, err := server.NewHTTPServer(opts...)

	if err != nil {
		return events.E("Error configuring http server", err)
	}
	err = hs.Serve()

	if err != nil && err != http.ErrServerClosed {
		return events.E("Error running http server", err)
	}
	return nil
}

func Serve() {
	Init("")
	var p *profile.Profile
	if Profiling() {
		l.LogI("Enabling profiling")
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
		defer p.Stop()
	}
	if len(LogDBConnStr()) > 0 {
		l.LogI("Switch log sink from os.Stdout to psqlDB")

		err := l.SetLogSink(l.ConnString(LogDBConnStr()), events.DebugLevel)
		if err != nil {
			l.LogE("Switching log sink", err)
			return
		}

	}

	opts, err := createHTTPServerOptions()
	if err != nil {
		l.LogE("Creating http server options", err)
		return
	}
	l.LogI("Starting server")
	err = serve(opts)
	if err != nil {
		l.LogE("Error starting http server", err)
	}

}
