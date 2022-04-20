package catalogue

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	graphql "github.com/graph-gophers/graphql-go"
	"github.com/jackc/pgx/v4/pgxpool"

	psql "github.com/equinor/oneseismic/api/internal/postgres"
	"github.com/equinor/oneseismic/api/internal/util"
)

//go:embed schema.graphql
var schema string

type gql struct {
	schema   *graphql.Schema
	connpool *pgxpool.Pool
	dbschema *dbSchema
}

type dbSchema struct {
	query string
	table string
	cols  Columns
}

type Columns struct {
	manifest string
	geometry string
}

func MakeGraphQL(connpool *pgxpool.Pool, dbschema *dbSchema) *gql {
	resolver := &resolver {}

	opts := []graphql.SchemaOpt{
		graphql.UseFieldResolvers(),
		graphql.UseStringDescriptions(),
	}
	_schema := graphql.MustParseSchema(schema, resolver, opts...)

	return &gql {
		schema   : _schema,
		connpool : connpool,
		dbschema : dbschema,
	}
}

func MakeDBSchema(
	table          string,
	manifestColumn string,
	geometryColumn string,
) (*dbSchema) {
	query := fmt.Sprintf(
		"SELECT %s FROM %s %s LIMIT $1 OFFSET $2",
		manifestColumn,
		table,
		"%s",
	)

	return &dbSchema{
		query: query,
		table: table,
		cols:  Columns {
			manifest: manifestColumn,
			geometry: geometryColumn,
		},
	}
}

type queryContext struct {
	connpool *pgxpool.Pool
	dbschema *dbSchema
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
		connpool: g.connpool,
		dbschema: g.dbschema,
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
	connpool := qctx.connpool
	dbschema := qctx.dbschema

	where := psql.Where(
		args.Where,
		args.Intersects,
		dbschema.cols.manifest,
		dbschema.cols.geometry,
	)

	query := fmt.Sprintf(dbschema.query, where)

	return psql.ExecQuery(
		connpool,
		query,
		args.First,
		args.Offset,
	)
}
