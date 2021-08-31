package api

import (
	"github.com/go-redis/redis/v8"

	"github.com/equinor/oneseismic/api/internal/auth"
)

type BasicEndpoint struct {
	keyring  *auth.Keyring
	tokens   auth.Tokens
	sched    scheduler // TODO: why not expose necessary methods directly?
	source   AbstractStorage
}

func MakeBasicEndpoint(
	keyring *auth.Keyring,
	storage redis.Cmdable,
	tokens auth.Tokens,
	source AbstractStorage,
) BasicEndpoint {
	return BasicEndpoint {
		keyring:  keyring,
		tokens:   tokens,
		/*
		 * Scheduler should probably be exported (and in internal/?) and be
		 * constructed directly by the caller.
		 */
		sched:  newScheduler(storage),
		source: source,
	}
}
