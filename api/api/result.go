package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/message"
	"github.com/go-redis/redis/v8"
	"github.com/gin-gonic/gin"
)

type Result struct {
	Timeout    time.Duration
	Storage    redis.Cmdable
	Keyring    *auth.Keyring
}

/*
 * Silly helper to centralise the name/key of the header object. It's not
 * likely to change too much, but it beats hardcoding the key with formatting
 * all over the place.
 */
func headerkey(pid string) string {
	return fmt.Sprintf("%s/header.json", pid)
}

func parseProcessHeader(doc []byte) (*message.ProcessHeader, error) {
	ph, err := (&message.ProcessHeader{}).Unpack(doc)
	if err != nil {
		log.Printf("bad process header: %s", string(doc))
		return ph, fmt.Errorf("unable to parse process header: %w", err)
	}

	if ph.Ntasks <= 0 {
		log.Printf("bad process header: %s", string(doc))
		return ph, fmt.Errorf("processheader.parts = %d; want >= 1", ph.Ntasks)
	}
	return ph, nil
}

func collectResult(
	ctx context.Context,
	storage redis.Cmdable,
	pid string,
	head *message.ProcessHeader,
	tiles chan []byte,
	failure chan error,
) {
	// This close is quite important - when the tiles channel is closed, it is
	// a signal to the caller that all partial results are in and processed,
	// and that the transfer is completed.
	defer close(tiles)

	tiles <- head.RawHeader

	streamCursor := "0"
	count := 0
	for count < head.Ntasks {
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
			for key, tile := range message.Values {
				// If we get a key named "error", something failed in the process
				// fetching fragments. Pass the error-text to failure-channel
				if key == "error" {
					log.Printf("pid=%s received error %s. Exit!", pid, tile.(string))
					failure <- errors.New(tile.(string))
					return
				}

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

	head, err := parseProcessHeader(body)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	count, err := r.Storage.XLen(ctx, pid).Result()
	// MORE tiles available than expected: This is the signal from
	// fetch-server that something failed - return error to let
	// client know that the request failed
	if count > int64(head.Ntasks) {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	tiles := make(chan []byte)
	failure := make(chan error)
	go collectResult(ctx, r.Storage, pid, head, tiles, failure)

	w := ctx.Writer
	header := w.Header()
	header.Set("Transfer-Encoding", "chunked")
	header.Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)

	for {
		select {
		case output, ok := <-tiles:
			if !ok {
				w.(http.Flusher).Flush()
				return
			}
			w.Write(output)

		// If we already started streaming and THEN something fails,
		// we cannot change the http status-code. The most standard
		// thing is to just close the stream and leave it to the client
		// to deal with partial data. Another approach is to introduce
		// a trailer-header to carry the final status
		//
		// https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.40)
		//
		// but http client-libraries rarely support this (as of Q3/2021)
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

	head, err := parseProcessHeader(body)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	count, err := r.Storage.XLen(ctx, pid).Result()
	// Fewer tiles available than expected: we are working on it
	if count < int64(head.Ntasks) {
		ctx.AbortWithStatus(http.StatusAccepted)
		return
	// MORE tiles available than expected: This is the signal from
	// fetch-server that something failed - return error
	} else if count > int64(head.Ntasks) {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	tiles := make(chan []byte, 1000)
	failure := make(chan error)
	go collectResult(ctx, r.Storage, pid, head, tiles, failure)

	result := make([]byte, 0)

	for tile := range tiles {
		result = append(result, tile...)
	}

	select {
	case err = <-failure:
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	default:
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
	 // TODO: See comment in graphql.go/basicSlice() and possibly wipe
	 // the discussion above + code below returning "pending"
	body, err := r.Storage.Get(ctx, headerkey(pid)).Bytes()
	if err == redis.Nil {
		/* request sucessful, but key does not exist */
		ctx.JSON(http.StatusOK, gin.H {
			"status": "pending",
			"progress": fmt.Sprintf("0/0"), // because we don't know anything yet
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
	completed := fmt.Sprintf("%d/%d", count, proc.Ntasks)

	if count == int64(proc.Ntasks) {
		ctx.JSON(http.StatusOK, gin.H {
			"status": "finished",
			"progress": completed,
		})
	} else if count > int64(proc.Ntasks) {
		ctx.JSON(http.StatusOK, gin.H {
			"status": "failed",
			"progress": fmt.Sprintf("0/%d", proc.Ntasks),
		})
	} else {
		ctx.JSON(http.StatusOK, gin.H {
			"status": "working",
			"progress": completed,
		})
	}
}
