package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/go-redis/redis/v8"
	"github.com/gin-gonic/gin"
	"github.com/vmihailenco/msgpack/v5"
)

type Result struct {
	Timeout    time.Duration
	StorageURL string
	Storage    redis.Cmdable
	Keyring    *auth.Keyring
}

/*
 * Silly helper to centralise the name/key of the header object. It's not
 * likely to change too much, but it beats hardcoding the key with formatting
 * all over the place.
 */
func headerkey(pid string) string {
	return fmt.Sprintf("%s:header.json", pid)
}

type processheader struct {
	Parts int `:json:"parts"`
}

func parseProcessHeader(doc []byte) (processheader, error) {
	ph := processheader {}
	if err := json.Unmarshal(doc, &ph); err != nil {
		log.Printf("bad process header: %s", string(doc))
		return ph, fmt.Errorf("unable to parse process header: %w", err)
	}

	if ph.Parts <= 0 {
		log.Printf("bad process header: %s", string(doc))
		return ph, fmt.Errorf("processheader.parts = %d; want >= 1", ph.Parts)
	}
	return ph, nil
}

func collectResult(
	ctx context.Context,
	storage redis.Cmdable,
	pid string,
	parts int,
	tiles chan []byte,
	failure chan error,
) {
	defer close(tiles)
	defer close(failure)

	var b bytes.Buffer
	enc := msgpack.NewEncoder(&b)
	err := enc.EncodeArrayLen(parts)
	if err != nil {
		failure <- err
		return
	}
	tiles <- b.Bytes()

	streamCursor := "0"
	count := 0
	for count < parts {
		xreadArgs := redis.XReadArgs{
			Streams: []string{pid, streamCursor},
			Block:   0,
		}
		reply, err := storage.XRead(ctx, &xreadArgs).Result()

		if err != nil {
			failure <- err
			return
		}

		for _, message := range reply[0].Messages {
			for _, tile := range message.Values {
				chunk, ok := tile.(string)
				if !ok {
					msg := fmt.Sprintf("tile.type = %T; expected []byte]", tile)
					failure <- errors.New(msg)
					return
				}

				tiles <- []byte(chunk)
				count++
			}
			streamCursor = message.ID
		}
	}
}

func (r *Result) Stream(ctx *gin.Context) {
	pid := ctx.Param("pid")
	body, err := r.Storage.Get(ctx, headerkey(pid)).Bytes()
	if err != nil {
		log.Printf("Unable to get process header: %v", err)
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	meta, err := parseProcessHeader(body)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	tiles := make(chan []byte)
	failure := make(chan error)
	go collectResult(ctx, r.Storage, pid, meta.Parts, tiles, failure)

	w := ctx.Writer
	header := w.Header()
	header.Set("Transfer-Encoding", "chunked")
	header.Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	for {
		select {
		case output, ok := <-tiles:
			if !ok {
				w.(http.Flusher).Flush()
				return
			}
			w.Write(output)

		case err := <-failure:
			log.Printf("pid=%s, %s", pid, err)
			return
		}
	}
}

func (r *Result) Get(ctx *gin.Context) {
	pid := ctx.Param("pid")
	body, err := r.Storage.Get(ctx, headerkey(pid)).Bytes()
	if err != nil {
		log.Printf("Unable to get process header: %v", err)
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	meta, err := parseProcessHeader(body)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	count, err := r.Storage.XLen(ctx, pid).Result()

	if count < int64(meta.Parts) {
		ctx.AbortWithStatus(http.StatusAccepted)
		return
	}

	tiles := make(chan []byte, 1000)
	failure := make(chan error)
	go collectResult(ctx, r.Storage, pid, meta.Parts, tiles, failure)

	result := make([]byte, 0)

	for tile := range tiles {
		result = append(result, tile...)
	}

	err, ok := <-failure

	if ok {
		log.Printf("pid=%s, %s", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
	}

	ctx.Data(http.StatusOK, "application/octet-stream", result)
}

func (r *Result) Status(ctx *gin.Context) {
	pid := ctx.Param("pid")
	/*
	 * There's an interesting timing issue here - if /result is called before
	 * the job is scheduled and the header written, it is considered pending.
	 *
	 * The fact that the token checks out means that it is essentially pending
	 * - it's enqueued, but no processing has started [1]. Also, partial
	 * results have a fairly short expiration set, and requests to /result
	 * after expiration would still carry a valid auth token.
	 *
	 * The fix here is probably to include created-at and expiration in the
	 * token as well - if the token checks out, but the header does not exist,
	 * the status is pending.
	 *
	 * [1] the header-write step not completed, to be precise
	 */
	body, err := r.Storage.Get(ctx, headerkey(pid)).Bytes()
	if err == redis.Nil {
		/* request sucessful, but key does not exist */
		ctx.JSON(http.StatusAccepted, gin.H {
			"location": fmt.Sprintf("result/%s/status", pid),
			"status": "pending",
		})
		return
	}
	if err != nil {
		log.Printf("%s %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	proc, err := parseProcessHeader(body)
	if err != nil {
		log.Printf("%s %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	count, err := r.Storage.XLen(ctx, pid).Result()
	if err != nil {
		log.Printf("%s %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	done := count == int64(proc.Parts)
	completed := fmt.Sprintf("%d/%d", count, proc.Parts)

	// TODO: add (and detect) failed status
	if done {
		ctx.JSON(http.StatusOK, gin.H {
			"location": fmt.Sprintf("result/%s", pid),
			"status": "finished",
			"progress": completed,
		})
	} else {
		ctx.JSON(http.StatusAccepted, gin.H {
			"location": fmt.Sprintf("result/%s/status", pid),
			"status": "working",
			"progress": completed,
		})
	}
}
