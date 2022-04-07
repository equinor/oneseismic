package main

import (
	"context"
	"io/ioutil"
	"log"
	"net/url"
	"net/http"

	"github.com/equinor/oneseismic/api/internal"
	"github.com/equinor/oneseismic/api/internal/util"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/dgraph-io/ristretto"
)

/*
 * This module implements a worker pool for blob fetches, with caching. The
 * intended use is for the main function to spawn a worker group that pulls
 * URLs and serves downloaded, possibly cached, blobs. The worker pool are
 * stupid pipes and have no context or reference to the task that provided the
 * URLs.
 */

/*
 * Silly & minimal interface to a cache of fragments - this hides a lot of
 * features, but should make some testing and benchmarking easier by providing
 * a way to tune or disable the cache.
 *
 * It achieves two things:
 * 1. ease-of-testing through custom cache implementations
 * 2. automates the casting, forcing the cache to only store the cacheentry
 *    type, which is way less annoying than dealing with interface{}
 */
type fragmentcache interface {
	set(string, cacheEntry)
	get(string) (cacheEntry, bool)
}

type cacheEntry struct {
	chunk []byte
	etag  *string
}

type ristrettocache struct {
	ristretto.Cache
}
func (c *ristrettocache) set(key string, val cacheEntry) {
	c.Set(key, val, 0)
}
func (c *ristrettocache) get(key string) (val cacheEntry, hit bool) {
	v, hit := c.Get(key)
	if hit {
		val = v.(cacheEntry)
	}
	return
}

/*
 * The nocache isn't really used per now, but serves as a useful reference and
 * available information for tests runs or test cases that wants to disable
 * caching altogether.
 */
type nocache struct {}
func (c *nocache) set(key string, val cacheEntry) {}
func (c *nocache) get(key string) (cacheEntry, bool) {
	return cacheEntry{}, false
}

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
	cache    fragmentcache
}

func newFetch(jobs int) *fetch {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7, // 100M
		MaxCost:     10 * (1 << 30), // 1 << 30 == 1G
		BufferItems: 64,
	})
	if err != nil {
		panic(err)
	}
	return &fetch {
		requests: make(chan request, jobs),
		cache:    &ristrettocache { Cache: *cache },
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

func downloadBlob(
	ctx    context.Context,
	blob   azblob.BlobClient,
	dlopts *azblob.DownloadBlobOptions,
) (cacheEntry, error) {
	dl, err := blob.Download(ctx, dlopts)
	if err != nil {
		return cacheEntry{}, util.UnpackAzStorageError(err)
	}
	body := dl.Body(&azblob.RetryReaderOptions{})
	defer body.Close()
	chunk, err := ioutil.ReadAll(body)
	return cacheEntry { chunk: chunk, etag: dl.ETag }, err
}

func fetchblob(
	ctx   context.Context,
	blob  *url.URL,
	cache fragmentcache,
) ([]byte, error) {
	if blob == nil  {
		log.Printf("Empty bloburl")
		return nil, internal.NewInternalError()
	}

	key := blob.Path
	cached, hit := cache.get(key)

	options := &azblob.DownloadBlobOptions{
		BlobAccessConditions: &azblob.BlobAccessConditions{
			ModifiedAccessConditions : &azblob.ModifiedAccessConditions{
				IfNoneMatch: cached.etag,
			},
		},
	}

	client, err := azblob.NewBlobClientWithNoCredential(
		blob.String(),
		&azblob.ClientOptions{},
	)

	if err != nil {
		return nil, err
	}

	cold, err := downloadBlob(ctx, client, options)
	if err == nil {
		/* nil means the azblob.Download succeeded *and* was not etag match */
		if hit {
			/* This probably means expired ETag, which again means a fragment
			* has been updated since cached. This should not happen in a
			* healthy system and must be investigated immediately.
			 */
			log.Printf(
				"ETag (= %s) expired for %v; investigate immediately",
				*cached.etag,
				blob,
			)
			return nil, internal.NewInternalError()
		} else {
			// This is good; not in cache, so clean fetch was expected.
			go cache.set(key, cold)
			return cold.chunk, nil
		}
	}

	switch e := err.(type) {
	case azblob.StorageError:
		status := e.Response().StatusCode
		if status == http.StatusNotModified {
			return cached.chunk, nil
		}
		// TODO: what other codes can actually show up here? Forbidden? No such
		// resource? For now, don't leak anything back, but log and add
		// case-by-case
		log.Printf("Unhandled azblob.StorageError: %v", err)
		return nil, internal.NewInternalError()

	default:
		log.Printf("Unhandled error type %T (= %v)", e, e)
		return nil, internal.NewInternalError()
	}
}

func (f *fetch) run() {
	for request := range f.requests {
		b, err := fetchblob(request.ctx, request.blob, f.cache)
		if err != nil {
			request.errors <- err
		} else {
			request.fragments <- fragment {
				index: request.index,
				chunk: b,
			}
		}
	}
}

func (f *fetch) startWorkers() {
	for i := 0; i < cap(f.requests); i++ {
		go f.run()
	}
}
