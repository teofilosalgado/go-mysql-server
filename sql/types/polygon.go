// Copyright 2022 Dolthub, Inc.
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

package types

import (
	"context"
	"reflect"

	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/geosenv"
	"github.com/dolthub/vitess/go/sqltypes"
)

// PolygonType represents the POLYGON type.
// https://dev.mysql.com/doc/refman/8.0/en/gis-class-polygon.html
// The type of the returned value is Polygon.
type PolygonType struct {
	GeometryType
}

// Polygon is the value type returned from PolygonType. Implements GeometryValue.
type Polygon struct {
	BaseGeometry
}

var _ sql.Type = PolygonType{}
var _ sql.SpatialColumnType = PolygonType{}
var _ sql.CollationCoercible = PolygonType{}
var _ GeometryValue = Polygon{}

var (
	polygonValueType = reflect.TypeOf(Polygon{})
)

// Convert implements Type interface.
func (t PolygonType) Convert(ctx context.Context, v interface{}) (interface{}, sql.ConvertInRange, error) {
	switch buf := v.(type) {
	case nil:
		return nil, sql.InRange, nil
	case []byte:
		polygon, _, err := GeometryType{}.Convert(ctx, buf)
		if sql.ErrInvalidGISData.Is(err) {
			return nil, sql.InRange, sql.ErrInvalidGISData.New("PolygonType.Convert")
		}
		return polygon, sql.InRange, nil
	case string:
		return t.Convert(ctx, []byte(buf))
	case Polygon:
		// TODO
		// The ST_SRID funcion may return a geometry with a different SRID from its original table. The following lines were removed to prevent this issue.
		// if err := t.MatchSRID(buf); err != nil {
		// 	return nil, sql.InRange, err
		// }
		return buf, sql.InRange, nil
	case GeometryValue:
		if buf.GetGeometry().Type() != "Polygon" {
			return nil, sql.InRange, sql.ErrInvalidGISData.New("PolygonType.Convert")
		}
		polygon := Polygon{BaseGeometry: BaseGeometry{Geometry: buf.GetGeometry()}}
		return polygon, sql.InRange, nil
	case sql.AnyWrapper:
		unwrapped, err := buf.UnwrapAny(ctx)
		if err != nil {
			return nil, sql.InRange, err
		}
		return t.Convert(ctx, unwrapped)
	default:
		return nil, sql.InRange, sql.ErrSpatialTypeConversion.New()
	}
}

// SQL implements Type interface.
func (t PolygonType) SQL(ctx *sql.Context, dest []byte, v interface{}) (sqltypes.Value, error) {
	if v == nil {
		return sqltypes.NULL, nil
	}

	v, _, err := t.Convert(ctx, v)
	if err != nil {
		return sqltypes.Value{}, nil
	}

	buf := v.(Polygon).Serialize()

	return sqltypes.MakeTrusted(sqltypes.Geometry, buf), nil
}

// String implements Type interface.
func (t PolygonType) String() string {
	return "polygon"
}

// ValueType implements Type interface.
func (t PolygonType) ValueType() reflect.Type {
	return polygonValueType
}

// Zero implements Type interface.
func (t PolygonType) Zero() interface{} {
	geosContext := geosenv.AcquireContext()
	defer geosenv.ReleaseContext(geosContext)
	return Polygon{BaseGeometry{Geometry: geosContext.NewEmptyPolygon()}}
}

// SetSRID implements SpatialColumnType interface.
func (t PolygonType) SetSRID(v int) sql.Type {
	t.SRID = v
	return t
}
