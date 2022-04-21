package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/equinor/oneseismic/api/internal"
)

type Schema struct {
	Table string
	Cols  Columns
}

type Columns struct {
	Manifest string
	Geometry string
}

type coordinate struct {
	X float64
	Y float64
}

type Point coordinate

type Linestring struct {
	Coords []coordinate
}

type Polygon struct {
	Coords []coordinate
}

type Geometry struct {
	Point      *Point
	Linestring *Linestring
	Polygon    *Polygon
}

type Manifest struct {
	FormatVersion  *int32        `json:"format-version,omitempty"`
	UploadFilename *string       `json:"upload-filename,omitempty"`
	Guid           *string       `json:"guid,omitempty"`
	Data           *[]Data       `json:"data,omitempty"`
	Attributes     *[]Attribute  `json:"attributes,omitempty"`
	LineNumbers    *[][]int32    `json:"line-numbers,omitempty"`
	LineLabels     *[]string     `json:"line-labels,omitempty"`
	SampleValueMin *float64      `json:"sample-value-min,omitempty"`
	SampleValueMax *float64      `json:"sample-value-max,omitempty"`
}

type Data struct {
	FileExtension *string    `json:"file-extention,omitempty"`
	Filters       *[]string  `json:"filters,omitempty"`
	Shapes        *[][]int32 `json:"shapes,omitempty"`
	Prefix        *string    `json:"prefix,omitempty"`
	Resolution    *string    `json:"resolution,omitempty"`
}

type Attribute struct {
	Type          *string    `json:"type,omitempty"`
	Layout        *string    `json:"layout,omitempty"`
	FileExtension *string    `json:"file-extention,omitempty"`
	Labels        *[]string  `json:"labels,omitempty"`
	Shapes        *[][]int32 `json:"shapes,omitempty"`
	Prefix        *string    `json:"prefix,omitempty"`
}

type ManifestFilter struct {
	Eq  *Manifest
	Or  *[]ManifestFilter
	And *[]ManifestFilter
}

type operator string

const (
	And operator = " AND "
	Or  operator = " OR "
)


/*
 * Stupid helper to join together terms by some operator
 */
func join(op operator, terms ...*string) (*string) {
	// Remove all nil values before combining
	_terms := make([]string, 0)
	for _, elem := range terms {
		if elem != nil {
			_terms = append(_terms, *elem)
		}
	}

	if len(_terms) == 0 {
		return nil
	}

	out := strings.Join(_terms, string(op))
	return &out
}

/*
 *  Map a manifest-instance (filter) to SQL using the jsonb containment
 *  operator ( @> )
 */
func equalityFilter(filter *Manifest, dbColumn string) (*string) {
	if filter == nil {
		return nil
	}

	b, _ := json.Marshal(filter)

	eq := fmt.Sprintf("%s @> '%s'", dbColumn, string(b))
	return &eq
}

func fmtCoordinates(coords []coordinate) (*string) {
	if len(coords) == 0 {
		return nil
	}

	cs := make([]string, 0)
	for _, c := range coords {
		// TODO utm precision
		cs = append(cs, fmt.Sprintf("%f %f", c.X, c.Y))
	}
	out := strings.Join(cs, ", ")
	return &out
}

/*
 * Map the recursive ManifestFilter into SQL syntax.
 */
func jsonbFilter(
	filter *ManifestFilter,
	dbColumn string,
	op operator,
) (*string) {
	if filter == nil {
		return nil
	}

	reduce := func(filters *[]ManifestFilter, op operator) (*string) {
		if filters == nil {
			return nil
		}

		processed := make([]string, 0)

		for _, filter := range *filters {
			x := jsonbFilter(&filter, dbColumn, op)
			if x != nil {
				processed = append(processed, *x)
			}
		}

		if len(processed) == 0 {
			return nil
		}

		out := fmt.Sprintf("(%s)", strings.Join(processed, string(op)))
		return &out
	}

	filters := make([]string, 0)

	eq := equalityFilter(filter.Eq, dbColumn)
	if eq != nil {
		filters = append(filters, *eq)
	}

	or := reduce(filter.Or, Or)
	if or != nil {
		filters = append(filters, *or)
	}

	and := reduce(filter.And, And)
	if and != nil {
		filters = append(filters, *and)
	}

	if len(filters) == 0 {
		return nil
	}

	out := fmt.Sprintf("%s", strings.Join(filters, string(op)))
	return &out
}

func gisPoint(p *Point) (*string) {
	if p == nil {
		return nil
	}

	// TODO utm precision
	out := fmt.Sprintf("POINT(%f %f)", p.X, p.Y)
	return &out
}

func gisLinestring(line *Linestring) (*string) {
	if line == nil {
		return nil
	}

	coords := fmtCoordinates(line.Coords)
	if coords == nil {
		return nil
	}

	out := fmt.Sprintf("LINESTRING(%s)", *coords)
	return &out
}

func gisPolygon(poly *Polygon) (*string) {
	if poly == nil {
		return nil
	}

	coords := fmtCoordinates(poly.Coords)
	if coords == nil {
		return nil
	}

	out := fmt.Sprintf("POLYGON((%s))", *coords)
	return &out
}

func gisIntersects(column string, geom *string) (*string) {
	if geom == nil {
		return nil
	}
	out := fmt.Sprintf("ST_Intersects(%s, '%s')", column, *geom)
	return &out
}

/*
 * Map a Geometry instance into one or more PostGIS ST_Intersects functions,
 * combined by defaultOp if applicable
 */
func gisGeometry(geom *Geometry, column string, defaultOp operator) (*string) {
	if geom == nil {
		return nil
	}

	point      := gisIntersects(column, gisPoint(geom.Point))
	linestring := gisIntersects(column, gisLinestring(geom.Linestring))
	polygon    := gisIntersects(column, gisPolygon(geom.Polygon))

	return join(defaultOp, point, linestring, polygon)
}

/*
 * Create a pqx Connection Pool, which is concurrency-safe. The caller is
 * responsible for closing the pool when done with it.
 */
func MakeConnectionPool(
	connstring string,
	logger     pgx.Logger,
) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connstring)
    if err != nil {
		msg := fmt.Sprintf("Unable to parse config: %v\n", err)
		log.Println(msg)
		return nil, internal.InternalError(msg)
    }

    config.ConnConfig.Logger = logger
    pool, err := pgxpool.ConnectConfig(context.Background(), config)
    if err != nil {
		msg := fmt.Sprintf("Unable to connect to database: %v\n", err)
		log.Println(msg)
		return nil, internal.InternalError(msg)
    }
    return pool, nil
}

/*
 * Create the SQL WHERE-clause from the ManifestFilter and Geometry.
 *
 * The 2 filters are joined together by an OR operation in case both are
 * present. If both are nil, the empty string is returned.
 *
 * The entire WHERE-clause is generated as a single string. Ideally we would
 * use SQL arguments here, but there is a hard limit of 10 when using SQL
 * arguments, which a complex json query can exceed.
 */
func Where(
	jsonFilter *ManifestFilter,
	geomFilter *Geometry,
	jsonColumn string,
	geomColumn string,
) (string) {
	/*
	 * Default to an OR operation in the case where multiple fields on
	 * geomFilter are set. I.e. if both Point and Polygon are defined, the
	 * resulting PostGIS query will be the intersection of the Polygon OR the
	 * Point.
	 */
	geom := gisGeometry(geomFilter, geomColumn, Or)

	/*
	 * Default to an AND operation when multiple manifest-fields are defined.
	 */
	json := jsonbFilter(jsonFilter, jsonColumn, And)

	var out string
	where := join(And, geom, json)
	if where != nil {
		out = fmt.Sprintf("WHERE %s", *where)
	}

	return out
}

/*
 * Execute a prepared query and scan the resulting rows into manifest objects
 */
func ExecQuery(
	connPool *pgxpool.Pool,
	query    string,
	args     ...interface{},
) ([]*Manifest, error) {
	rows, err := connPool.Query(context.Background(), query, args...)

	if err != nil {
		msg := fmt.Sprintf("Query failed with: %v", err)
		log.Println(msg)
		return nil, internal.QueryError(msg)
	}

	var manifests []*Manifest
	for rows.Next() {
		var m Manifest
		err := rows.Scan(&m)

		if err != nil {
			log.Fatal(err)
		}

		manifests = append(manifests, &m)
	}

	return manifests, nil
}
