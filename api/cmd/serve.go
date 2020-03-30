package cmd

import (
	"fmt"
	"net/http"

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

	err = server.Serve(*c)
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("error running http server: %w", err)
	}

	return nil
}
