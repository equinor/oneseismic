package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/go-redis/redis/v8"

	"github.com/equinor/oneseismic/api/internal/auth"
	"github.com/equinor/oneseismic/api/internal/message"
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
	pid  := keys["pid"]
	auth := keys["Authorization"]

	token, err := r.tokens.GetOnbehalf(auth)
	if err != nil {
		log.Printf("Cube(1) - pid=%s, %v", pid, err)
		return nil, NewIllegalAccessError("Unable to get on-behalf token")
	}

	doc, err := r.source.FetchManifest(
		ctx,
		token,
		string(args.Id),
	)
	// TODO: inspect error and determine if cached token should be evicted?
	if err != nil {
		log.Printf("Cube(2) - Cube ID=%s %v", args.Id, err)
		return nil, err
	}

	manifest := make(map[string]interface{})
	err = json.Unmarshal(doc, &manifest)
	if err != nil {
		log.Printf("Cube(3) - pid=%s %v", pid, err)
		return nil, NewInternalError("Failed converting to manifest")
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
	token, err := c.root.tokens.GetOnbehalf(auth)
	if err != nil {
		log.Printf("basicSlice(1) pid=%s, %v", pid, err)
		return nil, NewIllegalAccessError("Unable to get on-behalf token")
	}

	msg := message.Query {
		Pid:             pid,
		Token:           token,
		Guid:            string(c.id),
		Manifest:        c.manifest,
		StorageEndpoint: c.root.BasicEndpoint.source.GetUrl().String(),
		Function:        "slice",
		Args:            args,
		Opts:            opts,
	}
	query, err := c.root.sched.MakeQuery(&msg)
	if err != nil {
		log.Printf("basicSlice(2) pid=%s, %v", pid, err)
		return nil, NewIllegalInputError(fmt.Sprintf("Failed to construct query: %v", err.Error()))
	}

	key, err := c.root.keyring.Sign(pid)
	if err != nil {
		log.Printf("basicSlice(3) pid=%s, %v", pid, err)
		return nil, NewInternalError("Signing the token failed")
	}

	// TODO: Why async?
	// If we make this call synchronous we can report errors immediately
	// to client and avoid potentially messy cleanup later
	go func () {
		err := c.root.sched.Schedule(context.Background(), pid, query)
		if err != nil {
			/*
			 * Make scheduling errors fatal to detect them for debugging.
			 * Eventually this should log, maybe cancel the process, and
			 * continue.
			 */
			log.Fatalf("basicSlice(4) pid=%s, %v", pid, err)
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

	token, err := c.root.tokens.GetOnbehalf(auth)
	if err != nil {
		log.Printf("basicCurtain(1) pid=%s, %v", pid, err)
		return nil, NewIllegalAccessError("Unable to get on-behalf token")
	}

	msg := message.Query {
		Pid:             pid,
		Token:           token,
		Guid:            string(c.id),
		Manifest:        c.manifest,
		StorageEndpoint: c.root.BasicEndpoint.source.GetUrl().String(),
		Function:        "curtain",
		Args:            args,
	}
	query, err := c.root.sched.MakeQuery(&msg)
	if err != nil {
		log.Printf("basicCurtain(2) pid=%s, %v", pid, err)
		return nil, NewIllegalInputError(fmt.Sprintf("Failed to construct query: %v", err.Error()))
	}

	key, err := c.root.keyring.Sign(pid)
	if err != nil {
		log.Printf("basicCurtain(3) pid=%s, %v", pid, err)
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
	tokens   auth.Tokens,
	datasource AbstractStorage,
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
			storage,
			tokens,
			datasource,
		),
	}


	s := graphql.MustParseSchema(schema, resolver)
	return &gql {
		schema: s,
	}
}

func (g *gql) Get(ctx *gin.Context) {
	query  := ctx.Query("query")
	opName := ctx.Query("operationName")

	// TODO: parse the ?variables=... to this map
	//variables := ctx.Query("variables")
	variables := make(map[string]interface{})
	ctx.JSON(200, g.execQuery(ctx, query, opName, variables))
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
		log.Printf("Post(1) pid=%s %v", ctx.GetString("pid"), err)
		return
	}
	//log.Printf("query=%v  opName=%v  vars=%v", b.Query, b.OperationName, b.Variables)
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
	}
	c := context.WithValue(ctx, "keys", keys)
	response := g.schema.Exec(c, query, opName, variables)
	//log.Printf("response=%v,  data=%v  error=%v", response, response.Data, response.Errors)
	return response
}
