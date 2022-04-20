package catalogue

import (
	"context"
	_ "embed"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	graphql "github.com/graph-gophers/graphql-go"

	psql "github.com/equinor/oneseismic/api/internal/postgres"
	"github.com/equinor/oneseismic/api/internal/util"
)

//go:embed schema.graphql
var schema string

type gql struct {
	schema *graphql.Schema
	client psql.IndexClient
}

func MakeGraphQL(client psql.IndexClient) *gql {
	resolver := &resolver {}

	opts := []graphql.SchemaOpt{
		graphql.UseFieldResolvers(),
		graphql.UseStringDescriptions(),
	}
	_schema := graphql.MustParseSchema(schema, resolver, opts...)

	return &gql {
		schema : _schema,
		client : client,
	}
}

type queryContext struct {
	client psql.IndexClient
}

func (g *gql) Get(ctx *gin.Context) {
	query := ctx.Request.URL.Query()
	gqlquery, err := util.GraphQLQueryFromGet(query)
	if err != nil {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	delete(query, "query")
	delete(query, "operationName")
	delete(query, "variables")

	ctx.Request.URL.RawQuery = query.Encode()
	ctx.JSON(200, g.execQuery(ctx, gqlquery))
}

func (g *gql) Post(ctx *gin.Context) {
	body := util.GraphQLQuery {}
	err := ctx.BindJSON(&body)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	ctx.JSON(200, g.execQuery(ctx, &body))
}

func (g *gql) execQuery(
	ctx   *gin.Context,
	query *util.GraphQLQuery,
) *graphql.Response {
	qctx := queryContext {
		client: g.client,
	}

	c := context.WithValue(ctx, "queryctx", &qctx)
	return g.schema.Exec(c, query.Query, query.OperationName, query.Variables)
}

type resolver struct {
}

func (r *resolver) Manifests(
	ctx context.Context,
	args struct {
		Id         graphql.ID
		First      int32
		Offset     int32
		Where      *psql.ManifestFilter
		Intersects *psql.Geometry
	},
) ([]*psql.Manifest, error) {
	qctx := ctx.Value("queryctx").(*queryContext)
	client := qctx.client

	return client.GetManifests(
		args.Where,
		args.Intersects,
		args.First,
		args.Offset,
	)
}
