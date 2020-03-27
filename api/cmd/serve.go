package cmd

import (
	"fmt"
	"net/http"

	"github.com/equinor/oneseismic/api/events"
	l "github.com/equinor/oneseismic/api/logger"

	"github.com/equinor/oneseismic/api/server"
	"github.com/pkg/profile"
)

func createHTTPServerOptions(c server.Config) ([]server.HTTPServerOption, error) {
	opts := make([]server.HTTPServerOption, 0)

	opts = append(opts,
		server.WithOAuth2(server.OAuth2Option{
			AuthServer: &c.AuthServer,
			Audience:   c.ResourceID,
			Issuer:     c.Issuer,
			ApiSecret:  []byte(c.APISecret),
		}))

	if c.Profiling {
		opts = append(
			opts,
			server.WithProfiling())
	}

	return opts, nil
}

func Serve(m map[string]string) error {
	c, err := server.ParseConfig(m)
	if err != nil {
		return err
	}

	var p *profile.Profile

	if c.Profiling {
		pOpts := []func(*profile.Profile){
			profile.ProfilePath("pprof"),
			profile.NoShutdownHook,
		}

		pOpts = append(pOpts, profile.MemProfile)
		p = profile.Start(pOpts...).(*profile.Profile)

		defer p.Stop()
	}

	if len(c.LogDBConnStr) > 0 {
		l.LogI("Switch log sink from os.Stdout to psqlDB")

		err := l.SetLogSink(l.ConnString(c.LogDBConnStr), events.DebugLevel)
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

	sURL, err := server.NewServiceURL(c.AzureBlobSettings)
	if err != nil {
		return fmt.Errorf("creating ServiceURL: %w", err)
	}

	if sURL == nil {
		return fmt.Errorf("sURL should not be nil")
	}
	hs := server.Create(*sURL, *c)

	err = server.Configure(&hs, opts...)
	if err != nil {
		return events.E("Error configuring http server", err)
	}

	err = hs.Serve()
	if err != nil && err != http.ErrServerClosed {
		return events.E("Error running http server", err)
	}

	return nil
}
