package cmd

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/equinor/oneseismic/api/server"
	"github.com/pkg/profile"
)

func Serve(m map[string]string) error {
	profiling, err := strconv.ParseBool(m["PROFILING"])
	if err != nil {
		return fmt.Errorf("could not parse PROFILING: %w", err)
	}

	var p *profile.Profile

	if profiling {
		pOpts := []func(*profile.Profile){
			profile.ProfilePath("pprof"),
			profile.NoShutdownHook,
		}

		pOpts = append(pOpts, profile.MemProfile)
		p = profile.Start(pOpts...).(*profile.Profile)

		defer p.Stop()
	}

	err = server.Serve(m)
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("error running http server: %w", err)
	}

	return nil
}
