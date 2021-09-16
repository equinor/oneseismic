package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	graphql "github.com/graph-gophers/graphql-go"

	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/message"
)

type gql struct {
	schema *graphql.Schema
}

type resolver struct {
	tokens auth.Tokens
	datasource AbstractStorage
	keyring  *auth.Keyring
	sched    scheduler
}
type cube struct {
	id       graphql.ID
	root     *resolver
	manifest map[string]interface{}
	credentials string
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
	return NewInternalError("Promise is not an input type")
}

type opts struct {
	Attributes *[]string `json:"attributes"`
}

func (r *resolver) Cube(
	ctx context.Context,
	args struct { Id graphql.ID },
) (*cube, error) {
	keys := ctx.Value("keys").(map[string]string)
	pid  := GetPid(ctx)

	creds, err := EncodeCredentials(&r.tokens, keys)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		return nil, NewIllegalAccessError("Could not authenticate with given credentials")
	}

	resourceUrl := fmt.Sprintf("%s#manifest.json", string(args.Id))
	doc, err := r.datasource.Get(ctx, creds, resourceUrl); if err != nil {
		log.Printf("pid=%s %v", pid, err)
		return nil, err
	}

	manifest := make(map[string]interface{})
	err = json.Unmarshal(doc, &manifest)
	if err != nil {
		log.Printf("pid=%s %v", pid, err)
		return nil, NewInternalError("Failed converting to manifest")
	}

	return &cube {
		credentials: creds,
		id:          args.Id,
		root:        r,
		manifest:    manifest,
	}, nil
}

func (c *cube) Id() graphql.ID {
	return c.id
}

func asSliceInt32(root interface{}) ([]int32, error) {
	xs, ok := root.([]interface{})
	if !ok {
		return nil, errors.New("as([]int32) root was not []interface{}")
	}
	out := make([]int32, len(xs))
	for i, x := range xs {
		elem, ok := x.(float64)
		if !ok {
			return nil, errors.New("as([]int32) root[i] was not float64")
		}
		out[i] = int32(elem)
	}
	return out, nil
}

func asSliceSliceInt32(root interface{}) ([][]int32, error) {
	xs, ok := root.([]interface{})
	if !ok {
		return nil, errors.New("as([][]int32) root was not []interface{}")
	}
	out := make([][]int32, len(xs))
	for i, x := range xs {
		y, err := asSliceInt32(x)
		if err != nil {
			return nil, err
		}
		out[i] = y
	}
	return out, nil
}

func (c *cube) Linenumbers(ctx context.Context) ([][]int32, error) {
	doc, ok := c.manifest["line-numbers"]
	if !ok {
		keys := ctx.Value("keys").(map[string]string)
		pid  := keys["pid"]
		log.Printf(
			"Linenumbers(1) pid=%s %s/manifest.json broken; no dimensions",
			pid,
			string(c.id),
		)
		return nil, NewInternalError("Failed extracting document from manifest")
	}
	linenos, err := asSliceSliceInt32(doc)
	if err != nil {
		log.Printf("Linenumbers(2) %v", err)
		return nil, NewInternalError("Failed parsing linenumbers in manifest")
	}
	return linenos, nil
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

func (c *cube) SliceByLineno(
	ctx  context.Context,
	args struct {
		Dim    int32
		Lineno int32
		Opts   *opts
	},
) (*promise, error) {
	return c.basicSlice(
		ctx,
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
	return c.basicSlice(
		ctx,
		sliceargs {
			Kind: "index",
			Dim: args.Dim,
			Val: args.Index,
		},
		args.Opts,
	)
}

func (c *cube) basicSlice(
	ctx  context.Context,
	args sliceargs,
	opts *opts,
) (*promise, error) {
	keys := ctx.Value("keys").(map[string]string)
	pid := GetPid(ctx)
	/*
	 * Embedding a json doc as a string works (surprisingly) well, since the
	 * Pack()/encoding escapes all nested quotes. It might be reasonable at
	 * some point to change the underlying representation to messagepack, or
	 * even send the messages gzipped, but so for now strings and embedded
	 * documents should do fine.
	 *
	 * This opens an opportunity for the manifest forwarded not being quite
	 * faithful to what's stored in blob, i.e. information can be stripped out
	 * or added.
	 */
	msg := message.Query {
		Pid:             pid,
		Credentials:	 c.credentials,
		UrlQuery:        keys["url-query"],
		Guid:            string(c.id),
		Manifest:        c.manifest,
		StorageEndpoint: c.root.datasource.GetEndpoint(),
		StorageKind:     c.root.datasource.GetKind(),
		Function:        "slice",
		Args:            args,
		Opts:            opts,
	}
	query, err := c.root.sched.MakeQuery(&msg)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		return nil, NewIllegalInputError(fmt.Sprintf("Failed to construct query: %v", err.Error()))
	}

	key, err := c.root.keyring.Sign(pid)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		return nil, NewInternalError("Signing the token failed")
	}

	// TODO: Why async? Measure performance to determine if it is worth
	// keeping it async?
	// If we make this call synchronous we can report errors immediately
	// to client and avoid potentially messy cleanup later
	//
	// TODO: The first part of Schedule() writes header-information. If
	// we do that part synchronously, we avoid the timing issue mentioned
	// in Result.Status(). The loop could be done in a goroutine... (still
	// think it would be useful to measure time spent in this loop though)
	go func () {
		err := c.root.sched.Schedule(context.Background(), pid, query)
		if err != nil {
			/*
			 * Make scheduling errors fatal to detect them for debugging.
			 * Eventually this should log, maybe cancel the process, and
			 * continue.
			 */
			log.Fatalf("pid=%s, %v", pid, err)
		}
	}()

	return &promise {
		Url: fmt.Sprintf("result/%s", pid),
		Key: key,
	}, nil
}

func (c *cube) CurtainByIndex(
	ctx    context.Context,
	args   struct {
		Coords [][]int32
		Opts   *opts
	},
) (*promise, error) {
	return c.basicCurtain(
		ctx,
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
	return c.basicCurtain(
		ctx,
		curtainargs {
			Kind: "lineno",
			Coords: args.Coords,
		},
		args.Opts,
	)
}

func (c *cube) basicCurtain(
	ctx    context.Context,
	args   curtainargs,
	opts   *opts,
) (*promise, error) {
	keys := ctx.Value("keys").(map[string]string)
	pid := GetPid(ctx)

	msg := message.Query {
		Pid:             pid,
		Credentials:     c.credentials,
		UrlQuery:        keys["url-query"],
		Guid:            string(c.id),
		Manifest:        c.manifest,
		StorageEndpoint: c.root.datasource.GetEndpoint(),
		StorageKind:     c.root.datasource.GetKind(),
		Function:        "curtain",
		Args:            args,
		Opts:            opts,
	}
	query, err := c.root.sched.MakeQuery(&msg)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		return nil, NewIllegalInputError(fmt.Sprintf("Failed to construct query: %v", err.Error()))
	}

	key, err := c.root.keyring.Sign(pid)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		return nil, NewInternalError("Signing the token failed")
	}

	// TODO: See corresponding comment in basicSlice()
	go func () {
		err := c.root.sched.Schedule(context.Background(), pid, query)
		if err != nil {
			/*
			 * Make scheduling errors fatal to detect them for debugging.
			 * Eventually this should log, maybe cancel the process, and
			 * continue.
			 */
			log.Fatalf("pid=%s, %v", pid, err)
		}
	}()

	return &promise {
		Url: fmt.Sprintf("result/%s", pid),
		Key: key,
	}, nil
}

func MakeGraphQL(
	keyring  *auth.Keyring,
	storage  redis.Cmdable,
	datasource AbstractStorage,
	tokens auth.Tokens,
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

    sliceByLineno(dim: Int!, lineno: Int!, opts: Opts): Promise
    sliceByIndex(dim: Int!, index: Int!, opts: Opts): Promise
    curtainByLineno(coords: [[Int!]!]!, opts: Opts): Promise
    curtainByIndex( coords: [[Int!]!]!, opts: Opts): Promise
}
	`
	resolver := &resolver {
		tokens,
		datasource,
		keyring,
		newScheduler(storage),
	}

	s := graphql.MustParseSchema(schema, resolver)
	return &gql {
		schema: s,
	}
}

func (g *gql) Get(ctx *gin.Context) {
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
	query := ctx.Request.URL.Query()
	graphqueryargs := query["query"]
	if len(graphqueryargs) != 1 {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}
	graphquery := graphqueryargs[0]

	opname := ""
	opnameargs := query["operationName"]
	if len(opnameargs) > 1 {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if len(opnameargs) == 1 {
		opname = opnameargs[0]
	}

	variables := make(map[string]interface{})
	variablesargs := query["variables"]
	if len(variables) > 1 {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if len(opnameargs) == 1 {
		err := json.Unmarshal([]byte(variablesargs[0]), &variables)
		if err != nil {
			ctx.AbortWithStatus(http.StatusBadRequest)
			return
		}
	}

	delete(query, "query")
	delete(query, "operationName")
	delete(query, "variables")

	ctx.Request.URL.RawQuery = query.Encode()
	ctx.JSON(http.StatusOK, g.execQuery(ctx, graphquery, opname, variables))
}

func (g *gql) Post(ctx *gin.Context) {
	type body struct {
		Query         string                 `json:"query"`
		OperationName string                 `json:"operationName"`
		Variables     map[string]interface{} `json:"variables"`
	}
	b := body {}
	err := ctx.BindJSON(&b)
	if err != nil {
		log.Printf("pid=%s %v", ctx.GetString("pid"), err)
		return
	}
	ctx.JSON(http.StatusOK, g.execQuery(
		ctx,
		b.Query,
		b.OperationName,
		b.Variables,
	))
}

/*
* In order to trace a request through its whole lifetime we assume
* the context contains a string named "pid". This is a util-method
* with a simple fallback.
*/
func GetPid(ctx context.Context) string {
	pid := ctx.Value("pid")
	if pid == nil {
		return "<WARNING: Unknown pid>"
	}
	return pid.(string)
}

func (g *gql) execQuery(
	ctx    *gin.Context,
	query  string,
	opName string,
	variables map[string]interface{},
) *graphql.Response {
	/*
	* This map comprise the information passed to handlers
	* from requests, primarily used for authentication.
	* Extend this map as needed.
	*/
	keys := map[string]string {
		"Authorization": ctx.GetHeader("Authorization"),// OAuth2.0
		"url-query": ctx.Request.URL.RawQuery,          // Needed for SAAS-auth
	}
	c := context.WithValue(ctx, "keys", keys)
	c  = context.WithValue(c, "pid", ctx.GetString("pid"))
	return g.schema.Exec(c, query, opName, variables)
}
