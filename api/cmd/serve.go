package cmd

import (
	"net/http"

	"github.com/equinor/oneseismic/api/events"
	l "github.com/equinor/oneseismic/api/logger"
	"github.com/equinor/oneseismic/api/service"

	"github.com/equinor/oneseismic/api/server"
	"github.com/pkg/profile"
)

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

	ms, err := service.NewContainerURL(c.azureBlobSettings)
	if err != nil {
		return nil, events.E("Accessing manifest store", err)
	}

	opts = append(opts, server.WithContainerURL(ms))

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

func Serve(m map[string]string) error {
	c, err := parseConfig(m)
	if err != nil {
		return err
	}

	var p *profile.Profile

	if c.profiling {
		pOpts := []func(*profile.Profile){
			profile.ProfilePath("pprof"),
			profile.NoShutdownHook,
		}

		pOpts = append(pOpts, profile.MemProfile)
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

	hs, err := server.NewHTTPServer(server.DefaultHTTPServer(), opts...)
	if err != nil {
		return events.E("Error configuring http server", err)
	}

	err = hs.Serve()
	if err != nil && err != http.ErrServerClosed {
		return events.E("Error running http server", err)
	}

	return nil
}
