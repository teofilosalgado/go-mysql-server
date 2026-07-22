// Copyright 2020-2021 Dolthub, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spatial

import (
	"fmt"
	"strings"

	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/expression"
	"github.com/dolthub/go-mysql-server/sql/geosenv"
	"github.com/dolthub/go-mysql-server/sql/types"
	"github.com/twpayne/go-geos"
)

// AsWKT is a function that converts a spatial type into WKT format (alias for AsText)
type AsWKT struct {
	expression.UnaryExpressionStub
}

var _ sql.FunctionExpression = (*AsWKT)(nil)
var _ sql.CollationCoercible = (*AsWKT)(nil)

// NewAsWKT creates a new point expression.
func NewAsWKT(ctx *sql.Context, e sql.Expression) sql.Expression {
	return &AsWKT{expression.UnaryExpressionStub{Child: e}}
}

// FunctionName implements sql.FunctionExpression
func (p *AsWKT) FunctionName() string {
	return "st_aswkb"
}

// Description implements sql.FunctionExpression
func (p *AsWKT) Description() string {
	return "returns binary representation of given spatial type."
}

// IsNullable implements the sql.Expression interface.
func (p *AsWKT) IsNullable(ctx *sql.Context) bool {
	return p.Child.IsNullable(ctx)
}

// Type implements the sql.Expression interface.
func (p *AsWKT) Type(ctx *sql.Context) sql.Type {
	return types.LongText
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*AsWKT) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return ctx.GetCollation(), 4
}

func (p *AsWKT) String() string {
	return fmt.Sprintf("%s(%s)", p.FunctionName(), p.Child.String())
}

// WithChildren implements the Expression interface.
func (p *AsWKT) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	if len(children) != 1 {
		return nil, sql.ErrInvalidChildrenNumber.New(p, len(children), 1)
	}
	return NewAsWKT(ctx, children[0]), nil
}

// Eval implements the sql.Expression interface.
func (p *AsWKT) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	// Evaluate argument
	v, err := p.Child.Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	// Return nil if argument is nil
	if v == nil {
		return nil, nil
	}

	gv, err := types.UnwrapGeometry(ctx, v)
	if err != nil {
		return nil, sql.ErrInvalidArgument.New(p.FunctionName())
	}

	return gv.GetGeometry().ToWKT(), nil
}

// GeomFromText is a function that returns a point type from a WKT string
type GeomFromText struct {
	expression.NaryExpression
}

var _ sql.FunctionExpression = (*GeomFromText)(nil)
var _ sql.CollationCoercible = (*GeomFromText)(nil)

// NewGeomFromText creates a new point expression.
func NewGeomFromText(ctx *sql.Context, args ...sql.Expression) (sql.Expression, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, sql.ErrInvalidArgumentNumber.New("ST_GEOMFROMTEXT", "1, 2, or 3", len(args))
	}
	return &GeomFromText{expression.NaryExpression{ChildExpressions: args}}, nil
}

// FunctionName implements sql.FunctionExpression
func (g *GeomFromText) FunctionName() string {
	return "st_geomfromtext"
}

// Description implements sql.FunctionExpression
func (g *GeomFromText) Description() string {
	return "returns a new point from a WKT string."
}

// Type implements the sql.Expression interface.
func (g *GeomFromText) Type(ctx *sql.Context) sql.Type {
	// TODO: return type is determined after Eval, use Geometry for now?
	return types.GeometryType{}
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*GeomFromText) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 4
}

func (g *GeomFromText) String() string {
	var args = make([]string, len(g.ChildExpressions))
	for i, arg := range g.ChildExpressions {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", g.FunctionName(), strings.Join(args, ","))
}

// WithChildren implements the Expression interface.
func (g *GeomFromText) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	return NewGeomFromText(ctx, children...)
}

func WKTToPoint(geometry *geos.Geom) (types.Point, error) {
	return types.Point{BaseGeometry: types.BaseGeometry{Geometry: geometry}}, nil
}

func WKTToLineString(geometry *geos.Geom) (types.LineString, error) {
	return types.LineString{BaseGeometry: types.BaseGeometry{Geometry: geometry}}, nil
}

func WKTToPolygon(geometry *geos.Geom) (types.Polygon, error) {
	return types.Polygon{BaseGeometry: types.BaseGeometry{Geometry: geometry}}, nil
}

func WKTToMultiPoint(geometry *geos.Geom) (types.MultiPoint, error) {
	return types.MultiPoint{BaseGeometry: types.BaseGeometry{Geometry: geometry}}, nil
}

func WKTToMultiLineString(geometry *geos.Geom) (types.MultiLineString, error) {
	return types.MultiLineString{BaseGeometry: types.BaseGeometry{Geometry: geometry}}, nil
}

func WKTToMultiPolygon(geometry *geos.Geom) (types.MultiPolygon, error) {
	return types.MultiPolygon{BaseGeometry: types.BaseGeometry{Geometry: geometry}}, nil
}

func WKTToGeomColl(geometry *geos.Geom) (types.GeomColl, error) {
	return types.GeomColl{BaseGeometry: types.BaseGeometry{Geometry: geometry}}, nil
}

// WKTToGeom expects a string in WKT format, and converts it to a geometry type
func WKTToGeom(ctx *sql.Context, row sql.Row, exprs []sql.Expression, expectedGeomType string) (types.GeometryValue, error) {
	rawWKT, err := exprs[0].Eval(ctx, row)
	if err != nil {
		return nil, err
	}
	parsedWKT, ok := rawWKT.(string)
	if !ok {
		return nil, sql.ErrInvalidGISData.New()
	}

	rawSRID, err := exprs[1].Eval(ctx, row)
	if err != nil {
		return nil, err
	}
	parsedSrid, ok := rawSRID.(int16)
	if !ok {
		return nil, sql.ErrInvalidGISData.New()
	}

	geosContext := geosenv.AcquireContext()
	defer geosenv.ReleaseContext(geosContext)
	geomFromWKT, err := geosContext.NewGeomFromWKT(parsedWKT)
	if err != nil {
		return nil, err
	}
	geomFromWKT = geomFromWKT.SetSRID(int(parsedSrid))

	if expectedGeomType != "" && geomFromWKT.Type() != expectedGeomType {
		return nil, sql.ErrInvalidGISData.New()
	}

	switch geomFromWKT.Type() {
	case "Point":
		return WKTToPoint(geomFromWKT)
	case "LineString":
		return WKTToLineString(geomFromWKT)
	case "Polygon":
		return WKTToPolygon(geomFromWKT)
	case "MultiPoint":
		return WKTToMultiPoint(geomFromWKT)
	case "MultiLinestring":
		return WKTToMultiLineString(geomFromWKT)
	case "MultiPolygon":
		return WKTToMultiPolygon(geomFromWKT)
	case "GeometryCollection":
		return WKTToGeomColl(geomFromWKT)
	default:
		return nil, sql.ErrInvalidGISData.New()
	}
}

// Eval implements the sql.Expression interface.
func (g *GeomFromText) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	geom, err := WKTToGeom(ctx, row, g.ChildExpressions, "")
	if sql.ErrInvalidGISData.Is(err) {
		return nil, sql.ErrInvalidGISData.New(g.FunctionName())
	}
	return geom, err
}

// PointFromText is a function that returns a Point type from a WKT string
type PointFromText struct {
	expression.NaryExpression
}

var _ sql.FunctionExpression = (*PointFromText)(nil)
var _ sql.CollationCoercible = (*PointFromText)(nil)

// NewPointFromText creates a new point expression.
func NewPointFromText(ctx *sql.Context, args ...sql.Expression) (sql.Expression, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, sql.ErrInvalidArgumentNumber.New("ST_POINTFROMTEXT", "1, 2, or 3", len(args))
	}
	return &PointFromText{expression.NaryExpression{ChildExpressions: args}}, nil
}

// FunctionName implements sql.FunctionExpression
func (p *PointFromText) FunctionName() string {
	return "st_pointfromtext"
}

// Description implements sql.FunctionExpression
func (p *PointFromText) Description() string {
	return "returns a new point from a WKT string."
}

// Type implements the sql.Expression interface.
func (p *PointFromText) Type(ctx *sql.Context) sql.Type {
	return types.PointType{}
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*PointFromText) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 4
}

func (p *PointFromText) String() string {
	var args = make([]string, len(p.ChildExpressions))
	for i, arg := range p.ChildExpressions {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", p.FunctionName(), strings.Join(args, ","))
}

// WithChildren implements the Expression interface.
func (p *PointFromText) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	return NewPointFromText(ctx, children...)
}

// Eval implements the sql.Expression interface.
func (p *PointFromText) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	point, err := WKTToGeom(ctx, row, p.ChildExpressions, "Point")
	if sql.ErrInvalidGISData.Is(err) {
		return nil, sql.ErrInvalidGISData.New(p.FunctionName())
	}
	return point, err
}

// LineFromText is a function that returns a LineString type from a WKT string
type LineFromText struct {
	expression.NaryExpression
}

var _ sql.FunctionExpression = (*LineFromText)(nil)
var _ sql.CollationCoercible = (*LineFromText)(nil)

// NewLineFromText creates a new point expression.
func NewLineFromText(ctx *sql.Context, args ...sql.Expression) (sql.Expression, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, sql.ErrInvalidArgumentNumber.New("ST_LINEFROMTEXT", "1 or 2", len(args))
	}
	return &LineFromText{expression.NaryExpression{ChildExpressions: args}}, nil
}

// FunctionName implements sql.FunctionExpression
func (l *LineFromText) FunctionName() string {
	return "st_linefromtext"
}

// Description implements sql.FunctionExpression
func (l *LineFromText) Description() string {
	return "returns a new line from a WKT string."
}

// Type implements the sql.Expression interface.
func (l *LineFromText) Type(ctx *sql.Context) sql.Type {
	return types.LineStringType{}
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*LineFromText) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 4
}

func (l *LineFromText) String() string {
	var args = make([]string, len(l.ChildExpressions))
	for i, arg := range l.ChildExpressions {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", l.FunctionName(), strings.Join(args, ","))
}

// WithChildren implements the Expression interface.
func (l *LineFromText) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	return NewLineFromText(ctx, children...)
}

// Eval implements the sql.Expression interface.
func (l *LineFromText) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	line, err := WKTToGeom(ctx, row, l.ChildExpressions, "LineString")
	if sql.ErrInvalidGISData.Is(err) {
		return nil, sql.ErrInvalidGISData.New(l.FunctionName())
	}
	return line, err
}

// PolyFromText is a function that returns a Polygon type from a WKT string
type PolyFromText struct {
	expression.NaryExpression
}

var _ sql.FunctionExpression = (*PolyFromText)(nil)
var _ sql.CollationCoercible = (*PolyFromText)(nil)

// NewPolyFromText creates a new polygon expression.
func NewPolyFromText(ctx *sql.Context, args ...sql.Expression) (sql.Expression, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, sql.ErrInvalidArgumentNumber.New("ST_POLYFROMTEXT", "1, 2, or 3", len(args))
	}
	return &PolyFromText{expression.NaryExpression{ChildExpressions: args}}, nil
}

// FunctionName implements sql.FunctionExpression
func (p *PolyFromText) FunctionName() string {
	return "st_polyfromtext"
}

// Description implements sql.FunctionExpression
func (p *PolyFromText) Description() string {
	return "returns a new polygon from a WKT string."
}

// Type implements the sql.Expression interface.
func (p *PolyFromText) Type(ctx *sql.Context) sql.Type {
	return types.PolygonType{}
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*PolyFromText) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 4
}

func (p *PolyFromText) String() string {
	var args = make([]string, len(p.ChildExpressions))
	for i, arg := range p.ChildExpressions {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", p.FunctionName(), strings.Join(args, ","))
}

// WithChildren implements the Expression interface.
func (p *PolyFromText) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	return NewPolyFromText(ctx, children...)
}

// Eval implements the sql.Expression interface.
func (p *PolyFromText) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	poly, err := WKTToGeom(ctx, row, p.ChildExpressions, "Polygon")
	if sql.ErrInvalidGISData.Is(err) {
		return nil, sql.ErrInvalidGISData.New(p.FunctionName())
	}
	return poly, err
}

// MultiPoint is a function that returns a MultiPoint type from a WKT string
type MPointFromText struct {
	expression.NaryExpression
}

var _ sql.FunctionExpression = (*MPointFromText)(nil)
var _ sql.CollationCoercible = (*MPointFromText)(nil)

// NewMPointFromText creates a new MultiPoint expression.
func NewMPointFromText(ctx *sql.Context, args ...sql.Expression) (sql.Expression, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, sql.ErrInvalidArgumentNumber.New("ST_MULTIPOINTFROMTEXT", "1 or 2", len(args))
	}
	return &MPointFromText{expression.NaryExpression{ChildExpressions: args}}, nil
}

// FunctionName implements sql.FunctionExpression
func (p *MPointFromText) FunctionName() string {
	return "st_mpointfromtext"
}

// Description implements sql.FunctionExpression
func (p *MPointFromText) Description() string {
	return "returns a new multipoint from a WKT string."
}

// Type implements the sql.Expression interface.
func (p *MPointFromText) Type(ctx *sql.Context) sql.Type {
	return types.MultiPointType{}
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*MPointFromText) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 4
}

func (p *MPointFromText) String() string {
	var args = make([]string, len(p.ChildExpressions))
	for i, arg := range p.ChildExpressions {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", p.FunctionName(), strings.Join(args, ","))
}

// WithChildren implements the Expression interface.
func (p *MPointFromText) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	return NewMPointFromText(ctx, children...)
}

// Eval implements the sql.Expression interface.
func (p *MPointFromText) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	line, err := WKTToGeom(ctx, row, p.ChildExpressions, "MultiPoint")
	if sql.ErrInvalidGISData.Is(err) {
		return nil, sql.ErrInvalidGISData.New(p.FunctionName())
	}
	return line, err
}

// MLineFromText is a function that returns a MultiLineString type from a WKT string
type MLineFromText struct {
	expression.NaryExpression
}

var _ sql.FunctionExpression = (*MLineFromText)(nil)
var _ sql.CollationCoercible = (*MLineFromText)(nil)

// NewMLineFromText creates a new multilinestring expression.
func NewMLineFromText(ctx *sql.Context, args ...sql.Expression) (sql.Expression, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, sql.ErrInvalidArgumentNumber.New("ST_MLINEFROMTEXT", "1 or 2", len(args))
	}
	return &MLineFromText{expression.NaryExpression{ChildExpressions: args}}, nil
}

// FunctionName implements sql.FunctionExpression
func (l *MLineFromText) FunctionName() string {
	return "st_mlinefromtext"
}

// Description implements sql.FunctionExpression
func (l *MLineFromText) Description() string {
	return "returns a new multi line from a WKT string."
}

// Type implements the sql.Expression interface.
func (l *MLineFromText) Type(ctx *sql.Context) sql.Type {
	return types.MultiLineStringType{}
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*MLineFromText) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 4
}

func (l *MLineFromText) String() string {
	var args = make([]string, len(l.ChildExpressions))
	for i, arg := range l.ChildExpressions {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", l.FunctionName(), strings.Join(args, ","))
}

// WithChildren implements the Expression interface.
func (l *MLineFromText) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	return NewMLineFromText(ctx, children...)
}

// Eval implements the sql.Expression interface.
func (l *MLineFromText) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	mline, err := WKTToGeom(ctx, row, l.ChildExpressions, "MultilineString")
	if sql.ErrInvalidGISData.Is(err) {
		return nil, sql.ErrInvalidGISData.New(l.FunctionName())
	}
	return mline, err
}

// MPolyFromText is a function that returns a MultiPolygon type from a WKT string
type MPolyFromText struct {
	expression.NaryExpression
}

var _ sql.FunctionExpression = (*MPolyFromText)(nil)
var _ sql.CollationCoercible = (*MPolyFromText)(nil)

// NewMPolyFromText creates a new multilinestring expression.
func NewMPolyFromText(ctx *sql.Context, args ...sql.Expression) (sql.Expression, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, sql.ErrInvalidArgumentNumber.New("ST_MPOLYFROMTEXT", "1 or 2", len(args))
	}
	return &MPolyFromText{expression.NaryExpression{ChildExpressions: args}}, nil
}

// FunctionName implements sql.FunctionExpression
func (p *MPolyFromText) FunctionName() string {
	return "st_mpolyfromtext"
}

// Description implements sql.FunctionExpression
func (p *MPolyFromText) Description() string {
	return "returns a new multipolygon from a WKT string."
}

// Type implements the sql.Expression interface.
func (p *MPolyFromText) Type(ctx *sql.Context) sql.Type {
	return types.MultiPolygonType{}
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*MPolyFromText) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 4
}

func (p *MPolyFromText) String() string {
	var args = make([]string, len(p.ChildExpressions))
	for i, arg := range p.ChildExpressions {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", p.FunctionName(), strings.Join(args, ","))
}

// WithChildren implements the Expression interface.
func (p *MPolyFromText) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	return NewMPolyFromText(ctx, children...)
}

// Eval implements the sql.Expression interface.
func (p *MPolyFromText) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	mpoly, err := WKTToGeom(ctx, row, p.ChildExpressions, "MultiPolygon")
	if sql.ErrInvalidGISData.Is(err) {
		return nil, sql.ErrInvalidGISData.New(p.FunctionName())
	}
	return mpoly, err
}

// GeomCollFromText is a function that returns a MultiPolygon type from a WKT string
type GeomCollFromText struct {
	expression.NaryExpression
}

var _ sql.FunctionExpression = (*GeomCollFromText)(nil)
var _ sql.CollationCoercible = (*GeomCollFromText)(nil)

// NewGeomCollFromText creates a new multilinestring expression.
func NewGeomCollFromText(ctx *sql.Context, args ...sql.Expression) (sql.Expression, error) {
	if len(args) < 1 || len(args) > 3 {
		return nil, sql.ErrInvalidArgumentNumber.New("ST_GeomCollFromText", "1 or 2", len(args))
	}
	return &MPolyFromText{expression.NaryExpression{ChildExpressions: args}}, nil
}

// FunctionName implements sql.FunctionExpression
func (p *GeomCollFromText) FunctionName() string {
	return "st_geomcollfromtext"
}

// Description implements sql.FunctionExpression
func (p *GeomCollFromText) Description() string {
	return "returns a new geometry collection from a WKT string."
}

// Type implements the sql.Expression interface.
func (p *GeomCollFromText) Type(ctx *sql.Context) sql.Type {
	return types.GeomCollType{}
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*GeomCollFromText) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 4
}

func (p *GeomCollFromText) String() string {
	var args = make([]string, len(p.ChildExpressions))
	for i, arg := range p.ChildExpressions {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", p.FunctionName(), strings.Join(args, ","))
}

// WithChildren implements the Expression interface.
func (p *GeomCollFromText) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	return NewGeomFromText(ctx, children...)
}

// Eval implements the sql.Expression interface.
func (p *GeomCollFromText) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	geom, err := WKTToGeom(ctx, row, p.ChildExpressions, "GeometryCollection")
	if sql.ErrInvalidGISData.Is(err) {
		return nil, sql.ErrInvalidGISData.New(p.FunctionName())
	}
	return geom, err
}
