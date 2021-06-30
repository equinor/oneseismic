package api

import (
	"github.com/go-redis/redis/v8"

	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/message"
)

type BasicEndpoint struct {
	endpoint string // e.g. https://oneseismic-storage.blob.windows.net
	keyring  *auth.Keyring
	tokens   auth.Tokens
	sched    scheduler
}

func MakeBasicEndpoint(
	keyring *auth.Keyring,
	endpoint string,
	storage  redis.Cmdable,
	tokens   auth.Tokens,
) BasicEndpoint {
	return BasicEndpoint {
		endpoint: endpoint,
		keyring: keyring,
		tokens:  tokens,
		/*
		 * Scheduler should probably be exported (and in internal/?) and be
		 * constructed directly by the caller.
		 */
		sched:   newScheduler(storage),
	}
}

func (be *BasicEndpoint) MakeTask(
	pid       string,
	guid      string,
	token     string,
	manifest  interface{},
	shape     []int32,
) *message.Task {
	return &message.Task {
		Pid:             pid,
		Token:           token,
		Guid:            guid,
		StorageEndpoint: be.endpoint,
		Manifest:        manifest,
		Shape:           shape,
	}
}
