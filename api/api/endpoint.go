package api

import (
	"github.com/go-redis/redis/v8"

	"github.com/equinor/oneseismic/api/internal/auth"
)

type BasicEndpoint struct {
	endpoint string // e.g. https://oneseismic-storage.blob.windows.net
	keyring  *auth.Keyring
	sched    scheduler
}

func MakeBasicEndpoint(
	keyring *auth.Keyring,
	endpoint string,
	storage  redis.Cmdable,
) BasicEndpoint {
	return BasicEndpoint {
		endpoint: endpoint,
		keyring: keyring,
		/*
		 * Scheduler should probably be exported (and in internal/?) and be
		 * constructed directly by the caller.
		 */
		sched:   newScheduler(storage),
	}
}
