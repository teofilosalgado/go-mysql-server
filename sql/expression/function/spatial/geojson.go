package spatial

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/expression"
	"github.com/dolthub/go-mysql-server/sql/types"
	"github.com/twpayne/go-geos"
)

// AsGeoJSON is a function that returns a point type from a WKT string
type AsGeoJSON struct {
	expression.NaryExpression
}

var _ sql.FunctionExpression = (*AsGeoJSON)(nil)
var _ sql.CollationCoercible = (*AsGeoJSON)(nil)

func NewAsGeoJSON(ctx *sql.Context, args ...sql.Expression) (sql.Expression, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, sql.ErrInvalidArgumentNumber.New("ST_ASGEOJSON", "1, 2, or 3", len(args))
	}
	return &AsGeoJSON{expression.NaryExpression{ChildExpressions: args}}, nil
}

// FunctionName implements sql.FunctionExpression
func (g *AsGeoJSON) FunctionName() string {
	return "st_asgeojson"
}

// Description implements sql.FunctionExpression
func (g *AsGeoJSON) Description() string {
	return "returns a GeoJSON object from the geometry."
}

// Type implements the sql.Expression interface.
func (g *AsGeoJSON) Type(ctx *sql.Context) sql.Type {
	return types.JSON
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (f *AsGeoJSON) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return ctx.GetCollation(), 2
}
func (g *AsGeoJSON) String() string {
	var args = make([]string, len(g.ChildExpressions))
	for i, arg := range g.ChildExpressions {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", g.FunctionName(), strings.Join(args, ","))
}

// WithChildren implements the Expression interface.
func (g *AsGeoJSON) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	return NewAsGeoJSON(ctx, children...)
}

// Eval implements the sql.Expression interface.
func (g *AsGeoJSON) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	// convert spatial type to map, then place inside sql.JSONDocument
	expr0, err := g.ChildExpressions[0].Eval(ctx, row)
	if err != nil {
		return nil, err
	}
	if expr0 == nil {
		return nil, nil
	}

	geometry, err := types.UnwrapGeometry(ctx, expr0)
	if err != nil {
		return nil, sql.ErrInvalidArgumentType.New(g.FunctionName())
	}

	geosContext := geos.NewContext()
	geoJSONWriter := geosContext.NewGeoJSONWriter()
	geoJSONString := geoJSONWriter.WriteGeometry(geometry.GetGeometry(), 0)

	geoJSONMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(geoJSONString), &geoJSONMap)
	if err != nil {
		return nil, sql.ErrInvalidArgumentType.New(g.FunctionName())
	}

	return types.JSONDocument{Val: geoJSONMap}, nil
}

// GeomFromGeoJSON is a function returns a geometry based on a string
type GeomFromGeoJSON struct {
	expression.NaryExpression
}

var _ sql.FunctionExpression = (*GeomFromGeoJSON)(nil)
var _ sql.CollationCoercible = (*GeomFromGeoJSON)(nil)

func NewGeomFromGeoJSON(ctx *sql.Context, args ...sql.Expression) (sql.Expression, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, sql.ErrInvalidArgumentNumber.New("ST_GEOMFROMGEOJSON", "1, 2, or 3", len(args))
	}
	return &GeomFromGeoJSON{expression.NaryExpression{ChildExpressions: args}}, nil
}

// FunctionName implements sql.FunctionExpression
func (g *GeomFromGeoJSON) FunctionName() string {
	return "st_geomfromgeojson"
}

// Description implements sql.FunctionExpression
func (g *GeomFromGeoJSON) Description() string {
	return "returns a GeoJSON object from the geometry."
}

// Type implements the sql.Expression interface.
func (g *GeomFromGeoJSON) Type(ctx *sql.Context) sql.Type {
	return types.GeometryType{}
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*GeomFromGeoJSON) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 4
}

func (g *GeomFromGeoJSON) String() string {
	var args = make([]string, len(g.ChildExpressions))
	for i, arg := range g.ChildExpressions {
		args[i] = arg.String()
	}
	return fmt.Sprintf("ST_GEOMFROMGEOJSON(%s)", strings.Join(args, ","))
}

// WithChildren implements the Expression interface.
func (g *GeomFromGeoJSON) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	return NewGeomFromGeoJSON(ctx, children...)
}

func getIntArg(ctx *sql.Context, row sql.Row, expr sql.Expression) (interface{}, error) {
	x, err := expr.Eval(ctx, row)
	if err != nil {
		return nil, err
	}
	if x == nil {
		return nil, nil
	}
	switch x.(type) {
	case float32, float64:
		return nil, errors.New("received a float when it should be an int")
	}
	x, _, err = types.Int64.Convert(ctx, x)
	if err != nil {
		return nil, err
	}
	return int(x.(int64)), nil
}

// Eval implements the sql.Expression interface.
func (g *GeomFromGeoJSON) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	// Evaluate str argument
	val, err := g.ChildExpressions[0].Eval(ctx, row)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}
	val, _, err = types.LongBlob.Convert(ctx, val)
	if err != nil {
		return nil, err
	}

	val, err = sql.UnwrapAny(ctx, val)
	if err != nil {
		return nil, err
	}

	switch s := val.(type) {
	case string:
		val = s
	case []byte:
		val = string(s)
	}

	geoJSONString, ok := val.(string)
	if !ok {
		return nil, nil
	}
	geosContext := geos.NewContext()
	geoJSONReader := geosContext.NewGeoJSONReader()
	geometry, err := geoJSONReader.ReadGeometry(geoJSONString)

	res := types.BaseGeometry{Geometry: geometry}
	res.Geometry = res.Geometry.SetSRID(4326)
	if len(g.ChildExpressions) == 1 {
		return res, nil
	}

	// Evaluate options argument
	f, err := getIntArg(ctx, row, g.ChildExpressions[1])
	if err != nil {
		return nil, errors.New("incorrect flag value")
	}
	if f == nil {
		return nil, nil
	}
	flag := f.(int)
	if flag < 1 || flag > 4 {
		return nil, sql.ErrInvalidArgumentDetails.New(g.FunctionName(), flag)
	}
	if len(g.ChildExpressions) == 2 {
		return res, nil
	}

	// Evaluate srid argument
	s, err := getIntArg(ctx, row, g.ChildExpressions[2])
	if err != nil {
		return nil, errors.New("incorrect srid value")
	}
	if err = types.ValidateSRID(s.(int), g.FunctionName()); err != nil {
		return nil, err
	}
	srid := s.(int)
	res.Geometry = res.Geometry.SetSRID(srid)

	return res, nil
}
