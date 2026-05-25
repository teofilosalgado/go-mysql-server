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

// MultiPolygonType represents the MULTIPOLYGON type.
// https://dev.mysql.com/doc/refman/8.0/en/gis-class-multipolygon.html
// The type of the returned value is MultiPolygon.
type MultiPolygonType struct {
	GeometryType
}

// MultiPolygon is the value type returned from MultiPolygonType. Implements GeometryValue.
type MultiPolygon struct {
	BaseGeometry
}

var (
	multipolygonValueType = reflect.TypeOf(MultiPolygon{})
)

var _ sql.Type = MultiPolygonType{}
var _ sql.SpatialColumnType = MultiPolygonType{}
var _ sql.CollationCoercible = MultiPolygonType{}
var _ GeometryValue = MultiPolygon{}

// Convert implements Type interface.
func (t MultiPolygonType) Convert(ctx context.Context, v interface{}) (interface{}, sql.ConvertInRange, error) {
	switch buf := v.(type) {
	case nil:
		return nil, sql.InRange, nil
	case []byte:
		mpoly, _, err := GeometryType{}.Convert(ctx, buf)
		if sql.ErrInvalidGISData.Is(err) {
			return nil, sql.InRange, sql.ErrInvalidGISData.New("MultiPolygon.Convert")
		}
		return mpoly, sql.InRange, err
	case string:
		return t.Convert(ctx, []byte(buf))
	case MultiPolygon:
		if err := t.MatchSRID(buf); err != nil {
			return nil, sql.InRange, err
		}
		return buf, sql.InRange, nil
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
func (t MultiPolygonType) SQL(ctx *sql.Context, dest []byte, v interface{}) (sqltypes.Value, error) {
	if v == nil {
		return sqltypes.NULL, nil
	}

	v, _, err := t.Convert(ctx, v)
	if err != nil {
		return sqltypes.Value{}, nil
	}

	buf := v.(MultiPolygon).Serialize()

	return sqltypes.MakeTrusted(sqltypes.Geometry, buf), nil
}

// String implements Type interface.
func (t MultiPolygonType) String() string {
	return "multipolygon"
}

// ValueType implements Type interface.
func (t MultiPolygonType) ValueType() reflect.Type {
	return multipolygonValueType
}

// Zero implements Type interface.
func (t MultiPolygonType) Zero() interface{} {
	geosContext := geos.NewContext()
	return MultiPolygon{BaseGeometry{Geometry: geosContext.NewEmptyCollection(geos.TypeIDMultiPolygon)}}
}
