package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/gin-gonic/gin"
	"github.com/pebbe/zmq4"
	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/util"
	"github.com/equinor/oneseismic/api/internal/message"
)

type process struct {
	pid     string
	request []byte
}

func (proc *process) sendZMQ(socket *zmq4.Socket) (total int, err error) {
	return socket.SendMessage(proc.pid, proc.request)
}

type Slice struct {
	endpoint string // e.g. https://oneseismic-storage.blob.windows.net
	queue    *zmq4.Socket
	keyring  *auth.Keyring
}

func MakeSlice(
	keyring *auth.Keyring,
	endpoint string,
	queue *zmq4.Socket,
) Slice {
	return Slice {
		endpoint: endpoint,
		queue: queue,
		keyring: keyring,
	}
}

type sliceParams struct {
	guid      string
	dimension int
	lineno    int
}

/*
 * Helper function to parse the parameters of a request, to keep int
 * parsing boilerplate out of interesting logic
 */
func parseSliceParams(ctx *gin.Context) (*sliceParams, error) {
	guid := ctx.Param("guid")
	if guid == "" {
		return nil, fmt.Errorf("guid empty")
	}

	dim := ctx.Param("dimension")
	dimension, err := strconv.Atoi(dim)
	if err != nil {
		return nil, fmt.Errorf("error parsing dimension: %w", err)
	}

	lno := ctx.Param("lineno")
	lineno, err := strconv.Atoi(lno)
	if err != nil {
		return nil, fmt.Errorf("error parsing lineno: %w", err)
	}

	return &sliceParams {
		guid: guid,
		dimension: dimension,
		lineno: lineno,
	}, nil
}

/*
 * Make the process prototype for the slice. This is really just a stupid
 * copy-parameter-into-struct function, but is a hook for sanity checks,
 * hard-coded values etc (such as the function parameter).
 */
func (s *Slice) makeTask(
	pid string,
	token string,
	manifest string,
	params *sliceParams,
) message.Task {
	return message.Task {
		Pid:   pid,
		Token: token,
		Guid:  params.guid,
		StorageEndpoint: s.endpoint,
		Manifest: manifest,
		Shape: []int32 { 64, 64, 64, },
		Function: "slice",
		Params: &message.SliceParams {
			Dim:    params.dimension,
			Lineno: params.lineno,
		},
	}
}

/*
 * Get the manifest for the cube from the blob store.
 *
 * It's important that this is a blocking read, since this is the first
 * authorization mechanism in oneseismic. If the user (through the
 * on-behalf-token) does not have permissions to read the manifest, it
 * shouldn't be able to read the cube either. If so, no more processing should
 * be done, and the request discarded.
 */
func getManifest(
	ctx context.Context,
	token string,
	containerURL *url.URL,
) (string, error) {
	credentials := azblob.NewTokenCredential(token, nil)
	pipeline    := azblob.NewPipeline(credentials, azblob.PipelineOptions{})
	container   := azblob.NewContainerURL(*containerURL, pipeline)
	blob        := container.NewBlobURL("manifest.json")

	dl, err := blob.Download(
		ctx,
		0, /* offset */
		azblob.CountToEnd,
		azblob.BlobAccessConditions {},
		false, /* content-get-md5 */
	)
	if err != nil {
		return "", err
	}

	body := dl.Body(azblob.RetryReaderOptions{})
	defer body.Close()
	s, err := ioutil.ReadAll(body)
	return string(s), err
}

func (s *Slice) Get(ctx *gin.Context) {
	pid := util.MakePID()

	token := ctx.GetString("OBOJWT")
	if token == "" {
		/*
		 * The OBOJWT should be set in the middleware pipeline, so it's a
		 * programming error if it's not set
		 */
		log.Printf("%s OBOJWT was not set on request as it should be", pid)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	params, err := parseSliceParams(ctx)
	if err != nil {
		log.Printf("%s %v", pid, err)
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	container, err := url.Parse(fmt.Sprintf("%s/%s", s.endpoint, params.guid))
	if err != nil {
		log.Printf("%s %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	manifest, err := getManifest(ctx, token, container)
	if err != nil {
		log.Printf("%s %v", pid, err)
		/*
		 * Right now the error paths are trivial, so the error handling is
		 * inlined. It doesn't take much for this to become unwieldy however,
		 * at which point the error handling should probably be extracted into
		 * a helper to avoid polluting this function too much. The
		 * getManifest() and error handling is actually independent of the
		 * operation (/slice) and the second more endpoints are introduced,
		 * this should be pulled out and shared between them.
		 *
		 * The use of the azblob package makes it somewhat awkward to test as
		 * there's no obvious [1] place to inject a custom http client.
		 *
		 * [1] to my (limited) knowledge
		 */
		switch e := err.(type) {
			case azblob.StorageError:
				/*
				 * request successful, but the service returned some error e.g.
				 * a non-existing cube, unauthorized request.
				 *
				 * For now, just write the status-text into the body, which
				 * should be slightly more accurate than just the status code.
				 * Once the interface matures, this should probably be a more
				 * structured error message.
				 */
				sc := e.Response()
				ctx.String(sc.StatusCode, sc.Status)
				ctx.Abort()
			default:
				/*
				 * We don't care if the error occured is a networking error,
				 * faulty logic or something else - from the user side this is
				 * an InternalServerError regardless. At some point in the
				 * future, we might want to deal with particular errors here.
				 */
				ctx.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}

	/*
	 * Embedding a json doc as a string works (surprisingly) well, since the
	 * Pack()/encoding escapes all nested quotes. It might be reasonable at
	 * some point to change the underlying representation to messagepack, or
	 * even send the messages gzipped, but so for now strings and embedded
	 * documents should do fine.
	 *
	 * This opens an opportunity for the manifest forwaded not being quite
	 * faithful to what's stored in blob, i.e. information can be stripped out
	 * or added.
	 */
	msg := s.makeTask(pid, token, manifest, params)
	req, err := msg.Pack()
	if err != nil {
		log.Printf("%s pack error: %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	key, err := s.keyring.Sign(pid)
	if err != nil {
		log.Printf("%s %v", pid, err)
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	proc := process {
		pid: pid,
		request: req,
	}
	log.Printf("Scheduling %s", pid)
	proc.sendZMQ(s.queue)
	ctx.JSON(http.StatusOK, gin.H {
		"result": fmt.Sprintf("result/%s", pid),
		"authorization": key,
	})
}
