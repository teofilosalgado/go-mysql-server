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
	"github.com/dolthub/vitess/go/sqltypes"
	"github.com/twpayne/go-geos"
)

// MultiPointType represents the MULTIPOINT type.
// https://dev.mysql.com/doc/refman/8.0/en/gis-class-multipoint.html
// The type of the returned value is MultiPoint.
type MultiPointType struct {
	GeometryType
}

// MultiPoint is the value type returned from MultiPointType. Implements GeometryValue.
type MultiPoint struct {
	BaseGeometry
}

var _ sql.Type = MultiPointType{}
var _ sql.SpatialColumnType = MultiPointType{}
var _ sql.CollationCoercible = MultiPointType{}
var _ GeometryValue = MultiPoint{}

var (
	multiPointValueType = reflect.TypeOf(MultiPoint{})
)

// Convert implements Type interface.
func (t MultiPointType) Convert(ctx context.Context, v interface{}) (interface{}, sql.ConvertInRange, error) {
	switch buf := v.(type) {
	case nil:
		return nil, sql.InRange, nil
	case []byte:
		multipoint, _, err := GeometryType{}.Convert(ctx, buf)
		if sql.ErrInvalidGISData.Is(err) {
			return nil, sql.InRange, sql.ErrInvalidGISData.New("MultiPointType.Convert")
		}
		return multipoint, sql.InRange, nil
	case string:
		return t.Convert(ctx, []byte(buf))
	case MultiPoint:
		// TODO
		// The ST_SRID funcion may return a geometry with a different SRID from its original table. The following lines were removed to prevent this issue.
		// if err := t.MatchSRID(buf); err != nil {
		// 	return nil, sql.InRange, err
		// }
		return buf, sql.InRange, nil
	case GeometryValue:
		if buf.GetGeometry().Type() != "MultiPoint" {
			return nil, sql.InRange, sql.ErrInvalidGISData.New("MultiPointType.Convert")
		}
		multipoint := MultiPoint{BaseGeometry: BaseGeometry{Geometry: buf.GetGeometry()}}
		return multipoint, sql.InRange, nil
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
func (t MultiPointType) SQL(ctx *sql.Context, dest []byte, v interface{}) (sqltypes.Value, error) {
	if v == nil {
		return sqltypes.NULL, nil
	}

	v, _, err := t.Convert(ctx, v)
	if err != nil {
		return sqltypes.Value{}, nil
	}

	buf := v.(MultiPoint).Serialize()

	return sqltypes.MakeTrusted(sqltypes.Geometry, buf), nil
}

// String implements Type interface.
func (t MultiPointType) String() string {
	return "multipoint"
}

// ValueType implements Type interface.
func (t MultiPointType) ValueType() reflect.Type {
	return multiPointValueType
}

// Zero implements Type interface.
func (t MultiPointType) Zero() interface{} {
	geosContext := geos.NewContext()
	return MultiPoint{BaseGeometry{Geometry: geosContext.NewEmptyCollection(geos.TypeIDPoint)}}
}

// SetSRID implements SpatialColumnType interface.
func (t MultiPointType) SetSRID(v int) sql.Type {
	t.SRID = v
	return t
}
