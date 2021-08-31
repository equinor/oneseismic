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
	"github.com/go-redis/redis/v8"
	"github.com/Azure/azure-storage-blob-go/azblob"

	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/message"
	"github.com/equinor/oneseismic/api/internal/util"
)

type gql struct {
	schema *graphql.Schema
}

type resolver struct {
	BasicEndpoint
}
type cube struct {
	id       graphql.ID
	root     *resolver
	manifest map[string]interface{}
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

func credentials(
	tokens auth.Tokens,
	token string,
) (azblob.Credential, error) {
	log.Printf("Credentials token: %v", token)
	if token != "" {
		tok, err := tokens.GetOnbehalf(token)
		if err != nil {
			return nil, err
		}
		return azblob.NewTokenCredential(tok, nil), nil
	}
	return azblob.NewAnonymousCredential(), nil
}

func (r *resolver) Cube(
	ctx context.Context,
	args struct { Id graphql.ID },
) (*cube, error) {
	keys := ctx.Value("keys").(map[string]string)
	pid  := keys["pid"]
	auth := keys["Authorization"]

	creds, err := credentials(r.tokens, auth)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		return nil, errors.New("Unable to get on-behalf token")
	}
	doc, err := getManifest(
		ctx,
		creds,
		keys["url-query"],
		r.endpoint,
		string(args.Id),
	)
	// TODO: inspect error and determine if cached token should be evicted
	if err != nil {
		log.Printf("pid=%s %v", pid, err)
		return nil, err
	}

	manifest, err := manifestAsMap(doc)
	if err != nil {
		log.Printf("pid=%s %v", pid, err)
		return nil, err
	}

	return &cube {
		id:       args.Id,
		root:     r,
		manifest: manifest,
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
			"pid=%s %s/manifest.json broken; no dimensions",
			pid,
			string(c.id),
		)
		return nil, errors.New("internal error; bad document")
	}
	linenos, err := asSliceSliceInt32(doc)
	if err != nil {
		return nil, errors.New("internal error; bad document")
	}
	return linenos, nil
}

/*
 * This is the util.GetManifest function, but tuned for graphql and with
 * gin-specifics removed. Its purpose is to make for a quick migration to a
 * working graphql interface to oneseismic. Expect this function to be removed
 * or drastically change soon.
 */
func getManifest(
	ctx      context.Context,
	cred     azblob.Credential,
	urlquery string,
	endpoint string,
	guid     string,
) ([]byte, error) {
	container, err := url.Parse(fmt.Sprintf("%s/%s", endpoint, guid))
	if err != nil {
		return nil, err
	}

	container.RawQuery = urlquery
	manifest, err := util.FetchManifestWithCredential(ctx, cred, container)
	if err != nil {
		switch e := err.(type) {
		case azblob.StorageError:
			sc := e.Response()
			if sc.StatusCode == http.StatusNotFound {
				// TODO: add guid as a part of the error message?
				return nil, errors.New("Not found")
			}
			return nil, errors.New("Internal error")
		}
		return nil, err
	}
	return manifest, nil
}

func manifestAsMap(doc []byte) (m map[string]interface{}, err error) {
	err = json.Unmarshal(doc, &m)
	return
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
	pid  := keys["pid"]
	auth := keys["Authorization"]
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
	token := ""
	if auth != "" {
		tok, err := c.root.tokens.GetOnbehalf(auth)
		if err != nil {
			log.Printf("pid=%s, %v", pid, err)
			return nil, errors.New("internal error; bad token?")
		}
		token = tok
	}

	msg := message.Query {
		Pid:             pid,
		Token:           token,
		UrlQuery:        keys["url-query"],
		Guid:            string(c.id),
		Manifest:        c.manifest,
		StorageEndpoint: c.root.endpoint,
		Function:        "slice",
		Args:            args,
		Opts:            opts,
	}
	query, err := c.root.sched.MakeQuery(&msg)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		return nil, nil
	}

	key, err := c.root.keyring.Sign(pid)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		return nil, errors.New("internal error")
	}

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
	args   struct { Coords [][]int32 `json:"coords"` },
) (*promise, error) {
	return c.basicCurtain(
		ctx,
		curtainargs {
			Kind: "index",
			Coords: args.Coords,
		},
	)
}

func (c *cube) CurtainByLineno(
	ctx    context.Context,
	args   struct { Coords [][]int32 `json:"coords"` },
) (*promise, error) {
	return c.basicCurtain(
		ctx,
		curtainargs {
			Kind: "lineno",
			Coords: args.Coords,
		},
	)
}

func (c *cube) basicCurtain(
	ctx    context.Context,
	args   curtainargs,
) (*promise, error) {
	keys := ctx.Value("keys").(map[string]string)
	pid  := keys["pid"]
	auth := keys["Authorization"]

	token := ""
	if auth != "" {
		tok, err := c.root.tokens.GetOnbehalf(auth)
		if err != nil {
			log.Printf("pid=%s, %v", pid, err)
			return nil, errors.New("internal error; bad token?")
		}
		token = tok
	}

	msg := message.Query {
		Pid:             pid,
		Token:           token,
		UrlQuery:        keys["url-query"],
		Guid:            string(c.id),
		Manifest:        c.manifest,
		StorageEndpoint: c.root.endpoint,
		Function:        "curtain",
		Args:            args,
	}
	query, err := c.root.sched.MakeQuery(&msg)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		return nil, nil
	}

	key, err := c.root.keyring.Sign(pid)
	if err != nil {
		log.Printf("pid=%s, %v", pid, err)
		return nil, errors.New("internal error")
	}

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
	endpoint string,
	storage  redis.Cmdable,
	tokens   auth.Tokens,
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
    curtainByLineno(coords: [[Int!]!]!): Promise
    curtainByIndex(coords: [[Int!]!]!): Promise
}
	`
	resolver := &resolver {
		MakeBasicEndpoint(
			keyring,
			endpoint,
			storage,
			tokens,
		),
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
	ctx.JSON(200, g.execQuery(ctx, graphquery, opname, variables))
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

	ctx.JSON(200, g.execQuery(
		ctx,
		b.Query,
		b.OperationName,
		b.Variables,
	))
}

func (g *gql) execQuery(
	ctx    *gin.Context,
	query  string,
	opName string,
	variables map[string]interface{},
) *graphql.Response {
	keys := map[string]string {
		"pid": ctx.GetString("pid"),
		"Authorization": ctx.GetHeader("Authorization"),
		"url-query": ctx.Request.URL.RawQuery,
	}
	c := context.WithValue(ctx, "keys", keys)
	return g.schema.Exec(c, query, opName, variables)
}
