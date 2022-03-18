package postgres

import (
	"testing"
	"fmt"
	"strings"
)


var (
	point = Point{ X: 1, Y: 2 }
	
	linestring = Linestring{
		Coords: []coordinate{
			{ X: 2, Y: 2 },
			{ X: 3, Y: 2 },
		},
	}
	
	polygon = Polygon{
		Coords: []coordinate{
			{ X: 4, Y: 4 },
			{ X: 4, Y: 5 },
			{ X: 5, Y: 5 },
			{ X: 5, Y: 4 },
			{ X: 4, Y: 4 },
		},
	}
)

/* 
 * The query-strings that we assert on in these tests are rather long by
 * nature. This function makes the error message on assertion failures a
 * little more readable with some line-splitting
 */
func fmtErrorMsg(expected string, got string) (string) {
	return fmt.Sprintf("\nexpected:\n\t%v\ngot:\n\t%v", expected, got)
}

/*
 * Creating the expected query-strings is a hassle because they are so long.
 * String concatenation and string builder are both ugly, while string literals
 * preserve newlines and tabs, which are not part of the actual queries. But
 * they are the nicest of the 3 to write in the tests. This function takes a
 * string literal and clears it of any newlines or tabs.
 */
func whitespaceCorrection(str string) (string) {
	str =  strings.ReplaceAll(str, "\t", "")
	return strings.ReplaceAll(str, "\n", "")
}

func TestGisGeometryPoint(t *testing.T) {
	expected := "WHERE ST_Intersects(cdp, 'POINT(1.000000 2.000000)')"

	geom := Geometry{
		Point: &point,
	}

	result := Where(nil, &geom, "", "cdp")
	if result != expected {
		t.Error(fmtErrorMsg(expected, result))
	}
}

func TestGisGeometryLinestring(t *testing.T) {
	raw := `
		WHERE ST_Intersects(cdp, 
			'LINESTRING(2.000000 2.000000, 3.000000 2.000000)'
		)
	`
	expected := whitespaceCorrection(raw)

	geom := Geometry{
		Linestring: &linestring,
	}

	result := Where(nil, &geom, "", "cdp")
	if result != expected {
		t.Error(fmtErrorMsg(expected, result))
	}
}

func TestGisGeometryPolygon(t *testing.T) {
	raw := `
		WHERE ST_Intersects(cdp, 'POLYGON((
			4.000000 4.000000, 
			4.000000 5.000000, 
			5.000000 5.000000, 
			5.000000 4.000000, 
			4.000000 4.000000
		))
	')
	`
	expected := whitespaceCorrection(raw)

	geom := Geometry{
		Polygon: &polygon,
	}

	result := Where(nil, &geom, "", "cdp")
	if result != expected {
		t.Error(fmtErrorMsg(expected, result))
	}
}

func TestMultipleGIS(t *testing.T) {
	/*
	 * Where multiple geometries are defined, they should form an OR-operation
	 */
	raw := `WHERE 
		ST_Intersects(cdp, 'POINT(1.000000 2.000000)') OR 
		ST_Intersects(cdp, 'LINESTRING(2.000000 2.000000, 3.000000 2.000000)')
	`
	expected := whitespaceCorrection(raw)

	geom := Geometry{
		Point:      &point,
		Linestring: &linestring,
	}
	
	result := Where(nil, &geom, "", "cdp")

	if result != expected{
		t.Error(fmtErrorMsg(expected, result))
	}
}

func TestJsonEqualSingle(t *testing.T) {
	expected := `WHERE manifest @> '{"guid":"some-guid"}'`
	
	var guid string = "some-guid"
	filter := ManifestFilter{
		Eq: &Manifest{Guid: &guid},
	}
	
	result := Where(&filter, nil, "manifest", "")
	
	if result != expected{
		t.Error(fmtErrorMsg(expected, result))
	}
}

func TestJsonEqualMulti(t *testing.T) {
	expected := `WHERE manifest @> '{"format-version":1,"guid":"some-guid"}'`
	
	var guid string = "some-guid"
	var formatVersion int32 = 1
	filter := ManifestFilter{
		Eq: &Manifest{
			Guid:          &guid,
			FormatVersion: &formatVersion,
		},
	}
	
	result := Where(&filter, nil, "manifest", "")
	
	if result != expected{
		t.Error(fmtErrorMsg(expected, result))
	}
}

func TestJsonEqualNested(t *testing.T) {
	expected := `WHERE manifest @> '{"attributes":[{"type":"cdpx"}]}'`
	
	attrType := "cdpx"
	attrs := []Attribute{{ Type: &attrType }}

	filter := ManifestFilter{
		Eq: &Manifest{Attributes: &attrs},
	}
	
	result := Where(&filter, nil, "manifest", "")
	
	if result != expected{
		t.Error(fmtErrorMsg(expected, result))
	}
}

func TestJsonOperator(t *testing.T) {
	raw := `WHERE (
		manifest @> '{"guid":"some-guid"}' %s 
		manifest @> '{"format-version":1}'
	)`
	template := whitespaceCorrection(raw)

	var guid string = "some-guid"
	f1 := ManifestFilter{
		Eq: &Manifest{Guid: &guid},
	}
	
	var formatVersion int32 = 1
	f2 := ManifestFilter{
		Eq: &Manifest{FormatVersion: &formatVersion},
	}

	makeFilter := func(operator string) (*ManifestFilter) {
		switch operator {
			case "AND": return &ManifestFilter{And: &[]ManifestFilter{f1, f2}}
			case "OR":  return &ManifestFilter{Or:  &[]ManifestFilter{f1, f2}}
			default:
				return nil
		}
	}

	for _, op := range []string{"AND", "OR"} {
		expected := fmt.Sprintf(template, op)

		filter := makeFilter(op)
		result := Where(filter, nil, "manifest", "")
		
		if result != expected{
			t.Error(fmtErrorMsg(expected, result))
		}
	}
}

/* 
 *  This query is rather complex and nonsensical, but it tests a number of
 *  things:
 *
 *  1) intersects and where are combined by an AND-operation
 *  2) and/or can be nested
 *  3) defining more than one and/or/eq at the same 'level' combines the terms
 *     with an AND-operation. In this case the top-level 'or' and 'eq'.
 *
 * The GraphQL query:
 *
 *    manifests(
 *      intersects: { point: { x: 1, y: 2 } },
 *      where: {
 *        eq : { attributes: [{ type: $type }] },
 *        or: [{ 
 *          and: [
 *            { eq: { guid: "some-guid" } },
 *            { eq: { formatVersion: 1 } }
 *          ]},
 *          { eq: { formatVersion: 2 } },
 *        ]
 *      }
 *    ) {
 *      ...
 *    } 
 */
func TestComplex(t *testing.T) {
	raw := `WHERE 
		ST_Intersects(cdp, 'POINT(1.000000 2.000000)') AND 
		manifest @> '{"attributes":[{"type":"cdpx"}]}' AND 
		(
			(
				manifest @> '{"guid":"some-guid"}' AND 
				manifest @> '{"format-version":1}'
			) OR 
			manifest @> '{"format-version":2}'
		)`
	
	expected := whitespaceCorrection(raw)
		
	var guid string = "some-guid"
	f1 := ManifestFilter{
		Eq: &Manifest{Guid: &guid},
	}
	
	var formatVersion1 int32 = 1
	f2 := ManifestFilter{
		Eq: &Manifest{FormatVersion: &formatVersion1},
	}
	
	var formatVersion2 int32 = 2
	f3 := ManifestFilter{
		Eq: &Manifest{FormatVersion: &formatVersion2},
	}
	
	attrType := "cdpx"
	f4 := []Attribute{{ Type: &attrType }}

	and := ManifestFilter{And: &[]ManifestFilter{f1, f2}}
	filter  := ManifestFilter{
		Or: &[]ManifestFilter{and, f3},
		Eq: &Manifest{Attributes: &f4},
	}
		
	geom := Geometry {
		Point: &point,
	}

	result := Where(&filter, &geom, "manifest", "cdp")
		
	if result != expected{
		t.Error(fmtErrorMsg(expected, result))
	}
}
