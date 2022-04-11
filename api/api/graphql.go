package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/message"
	"github.com/equinor/oneseismic/api/internal/util"
	"github.com/equinor/oneseismic/api/internal"
)

/*
 * queryContext is for the graphql resolvers almost what the gin.context is for
 * the gin handlerfuncs. It carries "request-global" across function
 * boundaries, but unlike the gin.context it does not come with cancelling
 * semantics.
 *
 * It is meant to be stored and passed to resolvers through the context
 * argument.
 */
type queryContext struct {
	pid           string
	urlQuery      string
	session       *QuerySession
	endpoint      string
	keyring       *auth.Keyring
	scheduler     scheduler
}

/*
 * Stupid helper to get the query-context from the context. The aim is really
 * to hide the ugly string lookup and cast from the producing code.
 */
func getQueryContext(ctx context.Context) *queryContext {
	return ctx.Value("queryctx").(*queryContext)
}

/*
 * Stupid helper to set the query context on any context. It is mostly to make
 * sure that the names always match, and that tests are less effort setting up.
 */
func setQueryContext(ctx context.Context, qctx *queryContext) context.Context {
	return context.WithValue(ctx, "queryctx", qctx)
}

type gql struct {
	schema *graphql.Schema
	queryEngine QueryEngine
	endpoint  string // e.g. https://oneseismic-storage.blob.windows.net
	keyring   *auth.Keyring
	scheduler scheduler
}

type resolver struct {
}

type cube struct {
	id       graphql.ID
	manifest json.RawMessage
}

type promise struct {
	Url string `json:"url"`
	Key string `json:"key"`
}

func (promise) ImplementsGraphQLType(name string) bool {
	return name == "Promise"
}

func (p *promise) MarshalJSON() ([]byte, error) {
	return json.Marshal(p)
}

func (p *promise) UnmarshalGraphQL(input interface{}) error {
	// Unmarshal must be defined, but should never be used as an input type;
	// that's a schema bug, and all queries should be ignored.
	return errors.New("Promise is not an input type");
}

type opts struct {
	Attributes *[]string `json:"attributes"`
}

func (r *resolver) Cube(
	ctx context.Context,
	args struct { Id graphql.ID },
) (*cube, error) {
	qctx := getQueryContext(ctx)
	pid  := qctx.pid
	urls := fmt.Sprintf("%s/%s", qctx.endpoint, args.Id)
	log.Printf("Getting URL %s", urls)
	url, err := url.Parse(urls)
	if err != nil {
		log.Printf(
			"pid=%s, failed to parse URL; endpoint=%s, id=%s, error=%v",
			pid,
			qctx.endpoint,
			args.Id,
			err,
		)
		return nil, internal.NewInternalError()
	}

	doc, err := getManifest(ctx, qctx, url)
	if err != nil {
		return nil, err
	}

	err = qctx.session.InitWithManifest(doc)
	if err != nil {
		// errors here probably mean the document itself is broken
		// the URL gets recorded, but maybe the content (or digested content
		// e.g. hash) should be recorded as well)
		log.Printf(
			"pid=%s, init query engine session from %v failed: %v",
			pid,
			url,
			err,
		)
		return nil, internal.NewInternalError()
	}

	return &cube {
		id:       args.Id,
		manifest: doc,
	}, nil
}

func (c *cube) Id() graphql.ID {
	return c.id
}

func (c *cube) basicManifestQuery(
	ctx context.Context,
	path string,
	out interface {},
) error {
	qctx := getQueryContext(ctx)
	d, err := qctx.session.QueryManifest(path)
	if err != nil {
		log.Printf("pid=%s, %s failed: %v", qctx.pid, path, err)
		return internal.NewInternalError()
	}

	if len(d) == 0 {
		/*
		 * A key missing from the manifest is not an error (and if it is, it
		 * must be handled by the caller). This makes the queries more robust
		 * when properties are added to the manifest on new data and the old
		 * data isn't updated yet. Some metadata might also not be applicable.
		 */
		return internal.NewNotFoundError()
	}

	err = json.Unmarshal(d, out)
	if err != nil {
		/*
		 * This error should be *very* rare, and it means there is a mismatch
		 * between the manifest content => C++ parse-and-lookup => output. This
		 * should be investigated immediately.
		 */
		msg := "pid=%s, %s failed: unable to unmarshal %s to %T"
		log.Printf(msg, qctx.pid, path, string(d), out)
		return internal.NewInternalError()
	}
	return nil
}

func (c *cube) Linenumbers(ctx context.Context) ([][]int32, error) {
	var out [][]int32
	err := c.basicManifestQuery(ctx, "/line-numbers", &out)
	if err == nil {
		return out, nil
	} else {
		if _, ok := err.(*internal.NotFoundE); ok {
			// If this is reached, something is very wrong. The query engine
			// session init() should not succeed if the line numbers are
			// missing, which means this resolver is not constructible. This
			// should be debugged immediately.
			pid := getQueryContext(ctx).pid
			msg := "pid=%s, /line-numbers not found in %s manifest"
			log.Printf(msg, pid, c.id)
		}
	}
	return out, err
}

func (c *cube) SampleValueMin(ctx context.Context) (*float64, error) {
	var out float64
	err := c.basicManifestQuery(ctx, "/sample-value-min", &out)
	if err != nil {
		if _, ok := err.(*internal.NotFoundE); ok {
			return nil, nil
		}
	}
	return &out, err
}

func (c *cube) SampleValueMax(ctx context.Context) (*float64, error) {
	var out float64
	err := c.basicManifestQuery(ctx, "/sample-value-max", &out)
	if err != nil {
		if _, ok := err.(*internal.NotFoundE); ok {
			return nil, nil
		}
	}
	return &out, err
}

func (c *cube) FilenameOnUpload(ctx context.Context) (*string, error) {
	var out string
	err := c.basicManifestQuery(ctx, "/upload-filename", &out)
	if err != nil {
		if _, ok := err.(*internal.NotFoundE); ok {
			return nil, nil
		}
	}
	return &out, err
}

/*
 * This is the util.GetManifest function, but tuned for graphql and with
 * gin-specifics removed. Its purpose is to make for a quick migration to a
 * working graphql interface to oneseismic. Expect this function to be removed
 * or drastically change soon.
 */
func getManifest(
	ctx      context.Context,
	qctx     *queryContext,
	url      *url.URL,
) ([]byte, error) {
	// This is arguably bad; the passed url gets modified in-place. It's
	// probably ok since this is a helper function to pull the azure handling
	// stuff out of the caller body, and it is called once, but it should be
	// considered if this function should restore the rawQuery.
	url.RawQuery = qctx.urlQuery
	manifest, err := util.FetchManifest(ctx, url)
	if err == nil {
		return manifest, nil
	}

	log.Printf("pid=%s, %v", qctx.pid, err)
	switch e := err.(type) {
	case azblob.StorageError:
		status := e.Response().StatusCode
		switch status {
		case http.StatusNotFound:
			// TODO: add guid as a part of the error message?
			return nil, internal.QueryError("Not found")

		case http.StatusForbidden:
			return nil, internal.PermissionDeniedFromStatus(status)
		case http.StatusUnauthorized:
			return nil, internal.PermissionDeniedFromStatus(status)

		default:
			return nil, internal.NewInternalError()
		}
	}
	return nil, err
}

func (c *cube) basicQuery(
	ctx  context.Context,
	fun  string,
	args interface{},
	opts interface{},
) (*promise, error) {
	qctx := getQueryContext(ctx)
	pid  := qctx.pid
	msg  := message.Query {
		Pid:             pid,
		UrlQuery:        qctx.urlQuery,
		Guid:            string(c.id),
		Manifest:        c.manifest,
		StorageEndpoint: qctx.endpoint,
		Function:        fun,
		Args:            args,
		Opts:            opts,
	}
	query, err := qctx.session.PlanQuery(&msg)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		return nil, nil
	}

	key, err := qctx.keyring.Sign(pid)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		return nil, internal.NewInternalError()
	}

	go func (s scheduler) {
		err := s.Schedule(context.Background(), pid, query)
		if err != nil {
			/*
			 * Make scheduling errors fatal to detect them for debugging.
			 * Eventually this should log, maybe cancel the process, and
			 * continue.
			 */
			log.Fatalf("pid=%s, %v", pid, err)
		}
	}(qctx.scheduler)

	return &promise {
		Url: fmt.Sprintf("result/%s", pid),
		Key: key,
	}, nil
}

type sliceargs struct {
	Kind string `json:"kind"`
	Dim  int32  `json:"dim"`
	Val  int32  `json:"val"`
}

type curtainargs struct {
	Kind   string    `json:"kind"`
	Coords [][]int32 `json:"coords"`
}

type curtainargsUTM struct {
	Kind   string    `json:"kind"`
	Coords [][]float64 `json:"coords"`
}

func (c *cube) SliceByLineno(
	ctx  context.Context,
	args struct {
		Dim    int32
		Lineno int32
		Opts   *opts
	},
) (*promise, error) {
	return c.basicQuery(
		ctx,
		"slice",
		sliceargs {
			Kind: "lineno",
			Dim: args.Dim,
			Val: args.Lineno,
		},
		args.Opts,
	)
}

func (c *cube) SliceByIndex(
	ctx  context.Context,
	args struct {
		Dim   int32
		Index int32
		Opts   *opts
	},
) (*promise, error) {
	return c.basicQuery(
		ctx,
		"slice",
		sliceargs {
			Kind: "index",
			Dim: args.Dim,
			Val: args.Index,
		},
		args.Opts,
	)
}

func (c *cube) CurtainByIndex(
	ctx    context.Context,
	args   struct {
		Coords [][]int32
		Opts   *opts
	},
) (*promise, error) {
	return c.basicQuery(
		ctx,
		"curtain",
		curtainargs {
			Kind: "index",
			Coords: args.Coords,
		},
		args.Opts,
	)
}

func (c *cube) CurtainByLineno(
	ctx    context.Context,
	args   struct {
		Coords [][]int32
		Opts   *opts
	},
) (*promise, error) {
	return c.basicQuery(
		ctx,
		"curtain",
		curtainargs {
			Kind: "lineno",
			Coords: args.Coords,
		},
		args.Opts,
	)
}

func (c *cube) CurtainByUTM(
	ctx    context.Context,
	args   struct {
		Coords [][]float64
		Opts   *opts
	},
) (*promise, error) {
	return c.basicQuery(
		ctx,
		"curtain",
		curtainargsUTM {
			Kind: "utm",
			Coords: args.Coords,
		},
		args.Opts,
	)
}

func MakeGraphQL(
	keyring   *auth.Keyring,
	endpoint  string,
	scheduler scheduler,
) *gql {
	schema := `
scalar Promise

type Query {
    cube(id: ID!): Cube!
}

enum Attribute {
    cdp
    cdpx
    cdpy
}

input Opts {
    attributes: [Attribute!]
}

type Cube {
    id: ID!

    linenumbers: [[Int!]!]!
    sampleValueMin: Float
    sampleValueMax: Float
    filenameOnUpload: String

    sliceByLineno(dim: Int!, lineno: Int!, opts: Opts): Promise
    sliceByIndex(dim: Int!, index: Int!, opts: Opts): Promise
    curtainByLineno(coords: [[Int!]!]!, opts: Opts): Promise
    curtainByIndex( coords: [[Int!]!]!, opts: Opts): Promise
    curtainByUTM( coords: [[Float!]!]!, opts: Opts): Promise
}
	`
	resolver := &resolver {}
	s := graphql.MustParseSchema(schema, resolver)
	return &gql {
		schema: s,
		queryEngine: QueryEngine {
			tasksize: 10,
			pool: DefaultQueryEnginePool(),
		},
		endpoint:  endpoint,
		keyring:   keyring,
		scheduler: scheduler,
	}
}

func (g *gql) Get(ctx *gin.Context) {
	query := ctx.Request.URL.Query()
	q, err := util.GraphQLQueryFromGet(query)
	if err != nil {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	delete(query, "query")
	delete(query, "operationName")
	delete(query, "variables")

	ctx.Request.URL.RawQuery = query.Encode()
	ctx.JSON(200, g.execQuery(ctx, q))
}

func (g *gql) Post(ctx *gin.Context) {
	body := util.GraphQLQuery {}
	err := ctx.BindJSON(&body)
	if err != nil {
		log.Printf("pid=%s %v", ctx.GetString("pid"), err)
		return
	}

	ctx.JSON(200, g.execQuery(ctx, &body))
}

func (g *gql) execQuery(
	ctx   *gin.Context,
	query *util.GraphQLQuery,
) *graphql.Response {
	// The Query object is constructed here in order to have a single
	// entry/exit point for the QuerySession objects, to make sure they get put
	// back in the pool.
	session := g.queryEngine.Get()
	defer g.queryEngine.Put(session)
	qctx := queryContext {
		pid: ctx.GetString("pid"),
		urlQuery:  ctx.Request.URL.RawQuery,
		session:   session,
		endpoint:  g.endpoint,
		keyring:   g.keyring,
		scheduler: g.scheduler,
	}
	c := setQueryContext(ctx, &qctx)
	return g.schema.Exec(c, query.Query, query.OperationName, query.Variables)
}
