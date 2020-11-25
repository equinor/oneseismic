package util

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/equinor/oneseismic/api/internal/message"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func MakePID() string {
	return uuid.New().String()
}

func GeneratePID(ctx *gin.Context) {
	ctx.Set("pid", MakePID())
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
func GetManifest(
	ctx context.Context,
	token string,
	containerURL *url.URL,
) ([]byte, error) {
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
		return nil, err
	}

	body := dl.Body(azblob.RetryReaderOptions{})
	defer body.Close()
	return ioutil.ReadAll(body)
}

/*
 * Centralize the understanding of error conditions of azblob.download.
 *
 * There are two classes of errors:
 * 1. A hard failure, i.e. the request itself failed, such as network
 *    conditions.
 * 2. A soft failure, i.e. the request itself suceeded, but was without
 *    sufficient permissions, wrong syntax or similar. azblob probably handles
 *    this by parsing the status code and maybe the response body.
 *
 * Most calls to GetManifest() should probably immediately call this function
 * on error and exit, but it's moved into its own function so that error paths
 * can be emulated and tested independently. An added benefit is that should a
 * function, for some reason, need GetManifest() but want custom error+abort
 * handling, it is sufficient to implement bespoke error handling and simply
 * not call this.
 */
func AbortOnManifestError(ctx *gin.Context, err error) {
	switch e := err.(type) {
	case azblob.StorageError:
		/*
		 * Request successful, but the service returned some error e.g. a
		 * non-existing cube, unauthorized request.
		 *
		 * For now, just write the status-text into the body, which should be
		 * slightly more accurate than just the status code. Once the interface
		 * matures, this should probably be a more structured error message.
		 */
		sc := e.Response()
		ctx.String(sc.StatusCode, sc.Status)
		ctx.Abort()
	default:
		/*
		 * We don't care if the error occured is a networking error, faulty
		 * logic or something else - from the user side this is an
		 * InternalServerError regardless. At some point in the future, we might
		 * want to deal with particular errors here.
		 */
		ctx.AbortWithStatus(http.StatusInternalServerError)
	}
}

func ParseManifest(doc []byte) (*message.Manifest, error) {
	m := message.Manifest{}
	return m.Unpack(doc)
}
