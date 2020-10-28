package api

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/go-redis/redis"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/gin-gonic/gin"
)

type Result struct {
	Timeout    time.Duration
	StorageURL string
	Storage    redis.Cmdable
	Keyring    *auth.Keyring
}

/*
 * A super thin interface around the azblob download features, so that the
 * implementation can be substituted for testing
 */
type dlerror struct {
	status int
	what   string
}

func (e *dlerror) Error() string {
	return e.what
}

func downloaderror(status int, msg string) *dlerror {
	return &dlerror {
		status: status,
		what: msg,
	}
}
type storage interface {
	download(ctx context.Context, id string) ([]byte, *dlerror)
}

type azstorage struct {
	container azblob.ContainerURL
}

/*
 * This function exists to:
 * 1. Something callable to tests to ensure that responses are routed to the
 *    right channels
 * 2. Provide something that's easier to use both as a direct function call and
 *    a goroutine
 *
 * It is pretty trivial in its current form, but it's not impossible that
 * future additions and refactorings creates a need for more code than
 * container.download() and sending the result to the right channel.
 *
 * By writing it this way, the container.download can by synchronous, simple,
 * and with return values as you'd expect, and this function provides the extra
 * scaffolding needed to run it concurrently with the other download jobs.
 */
func downloadToChannel(
	ctx       context.Context,
	cancel    context.CancelFunc,
	container storage,
	url       string,
	results   chan []byte,
	failures  chan *dlerror,
) {
	body, err := container.download(ctx, url)
	if err != nil {
		/*
		 * send the cancel signal *before* posting on the failure channel, in
		 * case the failure channel is blocked or not scheduled fast enough
		 */
		cancel()
		failures <- err
	} else {
		results <- body
	}
}

func collect(
	parts   int,
	success chan []byte,
	failure chan *dlerror,
	timeout time.Duration,
) ([]byte, *dlerror) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	result := make([]byte, 5)
	/* msgpack array type */
	result[0] = 0xDD

	/* msgpack array length, a 4-byte big-endian integer */
	binary.BigEndian.PutUint32(result[1:], uint32(parts))
	for i := 0; i < parts; i++ {
		select {
		case fail := <-failure:
			return nil, fail

			/*
			 * TODO: this result can probably be streamed directly to client,
			 * rather than being buffered up with append. However, doing so is
			 * awkward because Content-Length won't be set, and handling errors
			 * probably becomes more difficult
			 */
		case part := <-success:
			result = append(result, part...)

		case <- timer.C:
			msg := fmt.Sprintf("timed out; got %d parts", i)
			return nil, downloaderror(http.StatusInternalServerError, msg)
		}
	}
	return result, nil
}

func (az *azstorage) download(
	ctx context.Context,
	id string,
) ([]byte, *dlerror) {
	url := az.container.NewBlobURL(id)
	dl, err := url.Download(
		ctx,
		0, /* offset */
		azblob.CountToEnd,
		azblob.BlobAccessConditions {},
		false, /* content-get-md5 */
	)

	/*
	 * TODO: query properly what's going on here.
	 * - Did the download fail?
	 * - Not authorized?
	 * - 404/not found?
	 * - Something misconfigured?
	 *
	 * 404 probably means that the object hasn't been written _yet_, in the
	 * case of data. If the header has already been read, the result is either
	 * underway, or has failed. Right now there's no way to tell, but since
	 * this function is used both for downloading the header and result
	 * payload, retry logic should probably be wherever this is called.
	 *
	 * Eventually, this should always fail hard on 404, and just make clients
	 * re-do the request at a later time.
	 */
	if err != nil {
		return nil, downloaderror(http.StatusInternalServerError, err.Error())
	}

	body := dl.Body(azblob.RetryReaderOptions{})
	defer body.Close()
	bytes, err := ioutil.ReadAll(body)
	/*
	 * An error here probably only means broken connection midway through
	 * transfer, since the Download() call succeeded. There's probably no
	 * recovery from this, so just return the error as-is and let the request
	 * be aborted
	 */
	if err != nil {
		return nil, downloaderror(http.StatusInternalServerError, err.Error())
	}
	return bytes, nil
}

/*
 * Check if a set of tiles are ready.
 *
 * This is a hack that
 * 1. hides some redis implementation detail from the Result.Get function and
 * 2. implements a wonky retry scheme to try to reduce latency
 *
 * Under ideal circumstances results are ready or almost-ready when fetched, in
 * which case sleep-and-wait will save a round trip, authentitication and a
 * bunch of other overhead. The ready() function will still time out after a
 * little more than a second, however, and is not infinitely blocking.
 *
 * A result is ready when all identifiers have been written to storage, so this
 * effectively boils down to asking if a set of keys exists, and count the ones
 * that do.
 */
func ready(storage redis.Cmdable, identifiers []string) (bool, error) {
	waits := []time.Duration {
		200,
		100,
		100,
		200,
		500,
		0,
	}

	items := int64(len(identifiers))

	for _, wait := range waits {
		count, err := storage.Exists(identifiers...).Result()
		if err != nil {
			return false, err
		}

		if count == items {
			return true, nil
		}
		time.Sleep(wait * time.Millisecond)
	}

	return false, nil
}

func (r *Result) Get(ctx *gin.Context) {
	pid := ctx.Param("pid")
	if len(pid) == 0 {
		log.Printf("pid empty")
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	authHeader := ctx.GetHeader("x-oneseismic-authorization")
	if authHeader == "" {
		log.Printf("%s x-oneseismic-authorization header not set", pid)
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	tok := ""
	_, scanerr := fmt.Sscanf(authHeader, "Bearer %s", &tok)
	if scanerr != nil {
		log.Printf("%s malformed Authorization; was %s", pid, authHeader)
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if verr := r.Keyring.Validate(tok, pid); verr != nil {
		log.Printf("%s %v", pid, verr)
		ctx.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	token := ctx.GetString("OBOJWT")
	if len(token) == 0 {
		log.Printf("%s OBOJWT was not set on request as it should be", pid)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	URL, _ := url.Parse(fmt.Sprintf("%s/results", r.StorageURL))
	credentials := azblob.NewTokenCredential(token, nil)
	pipeline    := azblob.NewPipeline(credentials, azblob.PipelineOptions{})
	container   := azstorage {
		container: azblob.NewContainerURL(*URL, pipeline),
	}

	body, err := container.download(
		context.Background(),
		fmt.Sprintf("%s-header.json", pid),
	)
	if err != nil {
		log.Printf("Unable to download result/meta: %v", err)
		ctx.AbortWithStatus(http.StatusNotFound)
		return
	}

	type Meta struct {
		Parts int `:json:"parts"`
	}

	meta := Meta {}
	if err := json.Unmarshal(body, &meta); err != nil {
		log.Printf("%s unable to parse meta: %v", pid, err)
		log.Printf("%s header: %s", pid, string(body))
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if meta.Parts <= 0 {
		log.Printf("%s got header with invalid parts; was %d", pid, meta.Parts)
		log.Printf("%s header: %s", pid, string(body))
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	identifiers := make([]string, 0)
	for i := 0; i < meta.Parts; i++ {
		id := fmt.Sprintf("%s-%d-%d", pid, i, meta.Parts)
		identifiers = append(identifiers, id)
	}

	ready, pollerr := ready(r.Storage, identifiers)
	if pollerr != nil {
		log.Printf("%s %v", pid, pollerr)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if !ready {
		log.Printf("%s tiles timed out; result not ready yet", pid)
		// TODO: return NotReady, or is that only for a /status method?
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	tiles, rerr := r.Storage.MGet(identifiers...).Result()
	if rerr != nil {
		log.Printf("%s failed to get result from storage; %v", pid, rerr)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	result := make([]byte, 5)
	/* msgpack array type */
	result[0] = 0xDD
	/* msgpack array length, a 4-byte big-endian integer */
	binary.BigEndian.PutUint32(result[1:], uint32(meta.Parts))

	for _, tile := range tiles {
		/*
		 * A chunk of bytes is represented as a string in redis, and mapped
		 * back to a string in go.
		 *
		 * The type cast is necessary [1] to copy the value, and doubles as a
		 * sanity check. Should an object for some reason be missing, or of an
		 * unexpected type, it probably means a programmer error at some other
		 * place in the system.
		 *
		 * [1] is it really?
		 */
		chunk, ok := tile.(string)
		if !ok {
			log.Printf("%s tile.type = %T; expected string", pid, tile)
			ctx.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		result = append(result, chunk...)
	}

	ctx.Data(http.StatusOK, "application/octet-stream", result)
}