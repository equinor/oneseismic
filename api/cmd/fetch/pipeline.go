package main

import (
	"context"
	"net/url"

	"github.com/equinor/oneseismic/api/internal/storage"
)

/*
 * The downloaded fragment (as it is stored in blob). The index is a key, used
 * to coordinate with C++ to map fragments (and by extension coordinates) to
 * the right extracted data (and coordinates).
 */
type fragment struct {
	index int
	chunk []byte
}

/*
 * A request for a blob download, complete with the output channel (including
 * error channel).
 */
type request struct {
	index     int
	blob      *url.URL
	fragments chan fragment
	errors    chan error
	ctx       context.Context
}

/*
 * This is the handle returned to the controlling process. The worker pool will
 * write the completed objects to the fragments channel, or post the error on
 * the error channel.
 *
 * Downloaded fragments are consumed by pulling messages from the fragments
 * channel. The worker pool has no concept of tasks or groups of downloads, so
 * it is up to the consumer to determine that all requested fragments are
 * received.
 *
 * The request should be aborted if any message is posted on the errors channel.
 */
type fetchQueue struct {
	fragments chan fragment
	errors    chan error
}

type fetch struct {
	requests chan request
	storage  storage.StorageClient
}

func newFetch(jobs int) *fetch {
	cache, err := storage.NewRistrettoCache()
	if err != nil {
		panic(err)
	}

	storage := storage.NewAzStorage(cache)
	return &fetch {
		requests: make(chan request, jobs),
		storage:  storage,
	}
}

/*
 * Make a new queue, or download session if you will. This is a part of the
 * make-enqueue process, which is split into two phases. For ease-of-use,
 * enqueue() will block until all the passed blobs are scheduled, and and the
 * consumers of the downloaded fragments also need access to the sink channels.
 * The easiest way to accomplish this is to split make and enqueue into two
 * functions.
 */
func (f *fetch) mkqueue() fetchQueue {
	return fetchQueue {
		fragments: make(chan fragment, cap(f.requests)),
		errors:    make(chan error,    cap(f.requests)),
	}
}

/*
 * The enqueue function is really just automation - it makes and schedules
 * requests for the passed urls. This function will block until all URLs are
 * scheduled.
 */
func (f *fetch) enqueue(
	ctx   context.Context,
	queue fetchQueue,
	urls  []*url.URL,
) {
	for i, url := range urls {
		f.requests <- request {
			index:     i,
			blob:      url,
			fragments: queue.fragments,
			errors:    queue.errors,
			ctx:       ctx,
		}
	}
}

func (f *fetch) run() {
	for request := range f.requests {
		blob, err := f.storage.Get(request.ctx, request.blob.String())
		if err != nil {
			request.errors <- err
		} else {
			request.fragments <- fragment {
				index: request.index,
				chunk: blob.Data,
			}
		}
	}
}

func (f *fetch) startWorkers() {
	for i := 0; i < cap(f.requests); i++ {
		go f.run()
	}
}
