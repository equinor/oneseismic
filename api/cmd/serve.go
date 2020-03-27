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
			l.LogE("Switching log sink", err)
			return err
		}
	}

	hs, err := server.Create(*c)
	if err != nil {
		return fmt.Errorf("could not create server: %w", err)
	}

	server.Configure(hs)

	err = hs.Serve()
	if err != nil && err != http.ErrServerClosed {
		return events.E("Error running http server", err)
	}

	return nil
}
