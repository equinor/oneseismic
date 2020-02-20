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

func stitchConfig(c config) interface{} {
	sGRPC := c.stitchGrpcAddr
	if len(sGRPC) > 0 {
		return service.GrpcOpts{Addr: sGRPC, Insecure: true}
	}

	return nil
}

func createHTTPServerOptions(c config) ([]server.HTTPServerOption, error) {
	opts := make([]server.HTTPServerOption, 0)

	if !c.noAuth {
		opts = append(opts,
			server.WithOAuth2(server.OAuth2Option{
				AuthServer: &c.authServer,
				Audience:   c.resourceID,
				Issuer:     c.issuer,
				ApiSecret:  []byte(c.apiSecret),
			}))
	}

	ms, err := store.NewManifestStore(c.azureBlobSettings)
	if err != nil {
		return nil, events.E("Accessing manifest store", err)
	}

	opts = append(opts, server.WithManifestStore(ms))

	st, err := service.NewStitch(stitchConfig(c))
	if err != nil {
		return nil, events.E("Stitch error", err)
	}

	opts = append(opts, server.WithStitcher(st))

	if len(c.hostAddr) > 0 {
		opts = append(
			opts,
			server.WithHostAddr(c.hostAddr))
	}

	if c.profiling {
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

func Serve(m map[string]string) error {
	c, err := parseConfig(m)
	if err != nil {
		return err
	}

	var p *profile.Profile

	if c.profiling {
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

	if len(c.logDBConnStr) > 0 {
		l.LogI("Switch log sink from os.Stdout to psqlDB")

		err := l.SetLogSink(l.ConnString(c.logDBConnStr), events.DebugLevel)
		if err != nil {
			l.LogE("Switching log sink", err)
			return err
		}
	}

	opts, err := createHTTPServerOptions(*c)
	if err != nil {
		l.LogE("Creating http server options", err)
		return err
	}

	l.LogI("Starting server")

	err = serve(opts)

	if err != nil {
		l.LogE("Error starting http server", err)
		return err
	}

	return nil
}
