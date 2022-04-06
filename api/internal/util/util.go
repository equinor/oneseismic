package util

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/equinor/oneseismic/api/internal"
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

type gzipWriter struct {
	gin.ResponseWriter
	writer *gzip.Writer
}

func (gz *gzipWriter) Write(b []byte) (int, error) {
	return gz.writer.Write(b)
}

// Compress the response if requested with a ?compression=kind query
//
// The implementation is roughly based on https://github.com/gin-contrib/gzip/
// with a couple of changed assumptions. The gin-contrib/gzip does not quite
// fit our usecase, and is very much geared towards compressing small
// text-responses and serving files, in a proper webserver fashion.
func Compression() gin.HandlerFunc {
	// responses are not very compressible (usually compressible to half the
	// size), and *speed* is the key anyway. Within the data centre it seems
	// like the break-even for compression time vs. saved transport cost is at
	// approx 20M responses.
	//
	// From a small rough experiment on a 14M response built from a 1.4M
	// response concatenated 10 times indicates that there are no significant
	// size improvements, but huge runtime costs in upping the compression
	// level:
	//
	// $ time gzip -1 -c response.bin | wc -c
	// 5812130
	// real    0m0.286s
	// $ time gzip -6 -c response.bin | wc -c
	// 5330006
	// real    0m1.230s

	// make new writers from a pool. There is some overhead in creating new
	// gzip writers, and they're easily re-usable. Using a sync.pool seems to
	// be the standard implementation for this.
	//
	// https://github.com/gin-contrib/gzip/blob/7bbc855cce8a575268c8f3e8d0f7a6a67f3dee65/handler.go#L22
	gzpool := sync.Pool {
		New: func() interface {} {
			gz, err := gzip.NewWriterLevel(ioutil.Discard, 1)
			if err != nil {
				panic(err)
			}
			return gz
		},
	}

	// It is very important that ctx.Next() is called - it effectively suspends
	// this handler and performs the request, then resumes where it left off.
	// It ensures that the Close(), Reset() and Put() are performed *after*
	// everything is properly written, and resources can be cleaned up.
	return func (ctx *gin.Context) {
		if (ctx.Query("compression") == "gz") {
			gz := gzpool.Get().(*gzip.Writer)
			defer gzpool.Put(gz)
			defer gz.Reset(ioutil.Discard)
			defer gz.Close()

			gz.Reset(ctx.Writer)
			ctx.Writer = &gzipWriter{ctx.Writer, gz}
			ctx.Header("Content-Encoding", "gzip")
			ctx.Next()

			if (ctx.GetHeader("Transfer-Encoding") != "chunked") {
				ctx.Header("Content-Length", fmt.Sprint(ctx.Writer.Size()))
			}
		}
	}
}

type GraphQLQuery struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

/*
 * Parse the the url?... parameters that graphql cares about (query,
 * operationName and variables), and forward the remaining parameters with
 * the graphql query. This enables users to pass query params to the blob
 * store effectively, which means SAS or other URL encoded auth can be used
 * with oneseismic.
 *
 * If a param is passed multiple times, e.g. graphql?query=...,query=... it
 * would be made into a list by net/url, but oneseismic considers this an
 * error to make it harder to make ambiguous requests. This makes for some
 * really ugly request parsing code.
 *
 * Only the query=... parameter is mandatory for GET requests.
 */
func GraphQLQueryFromGet(query url.Values) (*GraphQLQuery, error) {
	graphqueryargs := query["query"]
	if len(graphqueryargs) != 1 {
		return nil, internal.QueryError("Bad Request")
	}
	graphquery := graphqueryargs[0]

	opname := ""
	opnameargs := query["operationName"]
	if len(opnameargs) > 1 {
		return nil, internal.QueryError("Bad Request")
	}
	if len(opnameargs) == 1 {
		opname = opnameargs[0]
	}

	variables := make(map[string]interface{})
	variablesargs := query["variables"]
	if len(variablesargs) > 1 {
		return nil, internal.QueryError("Bad Request")
	}
	if len(variablesargs) == 1 {
		err := json.Unmarshal([]byte(variablesargs[0]), &variables)
		if err != nil {
			return nil, internal.QueryError("Bad Request")
		}
	}

	params := GraphQLQuery {
		Query:         graphquery,
		OperationName: opname,
		Variables:     variables,
	}

	return &params, nil
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
func FetchManifest(
	ctx          context.Context,
	containerURL *url.URL,
) ([]byte, error) {
	credentials := azblob.NewAnonymousCredential()
	pipeline    := azblob.NewPipeline(credentials, azblob.PipelineOptions{})
	container   := azblob.NewContainerURL(*containerURL, pipeline)
	blob        := container.NewBlobURL("manifest.json")
	dl, err := blob.Download(
		ctx,
		0, /* offset */
		azblob.CountToEnd,
		azblob.BlobAccessConditions {},
		false, /* content-get-md5 */
		azblob.ClientProvidedKeyOptions {},
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
 * Most calls to FetchManifest() should probably immediately call this function
 * on error and exit, but it's moved into its own function so that error paths
 * can be emulated and tested independently. An added benefit is that should a
 * function, for some reason, need FetchManifest() but want custom error+abort
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

/*
 * Custom logger for the /query family of endpoints, that logs the id of the
 * process to be generated by the request (pid).
 */
func QueryLogger(ctx *gin.Context) {
	gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"pid=%s, %s %s %s %d %s %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Keys["pid"],
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.ErrorMessage,
		)
	})(ctx)
}
