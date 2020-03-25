package cmd

import (
	"fmt"
	"net/http"

	"github.com/equinor/oneseismic/api/events"
	l "github.com/equinor/oneseismic/api/logger"

	"github.com/equinor/oneseismic/api/server"
	"github.com/pkg/profile"
)

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
			return fmt.Errorf("switching log sink: %w", err)
		}
	}

	sURL, err := server.NewServiceURL(c.AzureBlobSettings)
	if err != nil {
		return fmt.Errorf("creating ServiceURL: %w", err)
	}

	hs, err := server.Create(sURL, c)
	if err != nil {
		return fmt.Errorf("error creating server: %w", err)
	}

	hs.RegisterEndpoints()

	err = hs.Serve()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("error running http server: %w", err)
	}

	return nil
}
