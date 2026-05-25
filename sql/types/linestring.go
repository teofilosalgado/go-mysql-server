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

// LineStringType represents the LINESTRING type.
// https://dev.mysql.com/doc/refman/8.0/en/gis-class-linestring.html
// The type of the returned value is LineString.
type LineStringType struct {
	GeometryType
}

// LineString is the value type returned from LineStringType. Implements GeometryValue.
type LineString struct {
	BaseGeometry
}

var _ sql.Type = LineStringType{}
var _ sql.SpatialColumnType = LineStringType{}
var _ sql.CollationCoercible = LineStringType{}
var _ GeometryValue = LineString{}

var (
	lineStringValueType = reflect.TypeOf(LineString{})
)

// Convert implements Type interface.
func (t LineStringType) Convert(ctx context.Context, v interface{}) (interface{}, sql.ConvertInRange, error) {
	switch buf := v.(type) {
	case nil:
		return nil, sql.InRange, nil
	case []byte:
		line, _, err := GeometryType{}.Convert(ctx, buf)
		if sql.ErrInvalidGISData.Is(err) {
			return nil, sql.InRange, sql.ErrInvalidGISData.New("LineStringType.Convert")
		}
		return line, sql.InRange, err
	case string:
		return t.Convert(ctx, []byte(buf))
	case LineString:
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
func (t LineStringType) SQL(ctx *sql.Context, dest []byte, v interface{}) (sqltypes.Value, error) {
	if v == nil {
		return sqltypes.NULL, nil
	}

	v, _, err := t.Convert(ctx, v)
	if err != nil {
		return sqltypes.Value{}, nil
	}

	buf := v.(LineString).Serialize()

	return sqltypes.MakeTrusted(sqltypes.Geometry, buf), nil
}

// String implements Type interface.
func (t LineStringType) String() string {
	return "linestring"
}

// ValueType implements Type interface.
func (t LineStringType) ValueType() reflect.Type {
	return lineStringValueType
}

// Zero implements Type interface.
func (t LineStringType) Zero() interface{} {
	geosContext := geos.NewContext()
	return LineString{BaseGeometry{Geometry: geosContext.NewEmptyLineString()}}
}
