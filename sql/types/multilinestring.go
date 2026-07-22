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
	"github.com/twpayne/go-geos"
)

// MultiLineStringType represents the MUTILINESTRING type.
// https://dev.mysql.com/doc/refman/8.0/en/gis-class-multilinestring.html
// The type of the returned value is MultiLineString.
type MultiLineStringType struct {
	GeometryType
}

// MultiLineString is the value type returned from MultiLineStringType. Implements GeometryValue.
type MultiLineString struct {
	BaseGeometry
}

var (
	multilinestringValueType = reflect.TypeOf(MultiLineString{})
)

var _ sql.Type = MultiLineStringType{}
var _ sql.SpatialColumnType = MultiLineStringType{}
var _ sql.CollationCoercible = MultiLineStringType{}
var _ GeometryValue = MultiLineString{}

// Convert implements Type interface.
func (t MultiLineStringType) Convert(ctx context.Context, v interface{}) (interface{}, sql.ConvertInRange, error) {
	switch buf := v.(type) {
	case nil:
		return nil, sql.InRange, nil
	case []byte:
		mline, _, err := GeometryType{}.Convert(ctx, buf)
		if sql.ErrInvalidGISData.Is(err) {
			return nil, sql.InRange, sql.ErrInvalidGISData.New("MultiLineString.Convert")
		}
		return mline, sql.InRange, err
	case string:
		return t.Convert(ctx, []byte(buf))
	case MultiLineString:
		// TODO
		// The ST_SRID funcion may return a geometry with a different SRID from its original table. The following lines were removed to prevent this issue.
		// if err := t.MatchSRID(buf); err != nil {
		// 	return nil, sql.InRange, err
		// }
		return buf, sql.InRange, nil
	case GeometryValue:
		if buf.GetGeometry().Type() != "MultiLinestring" {
			return nil, sql.InRange, sql.ErrInvalidGISData.New("MultiLineString.Convert")
		}
		mline := MultiLineString{BaseGeometry: BaseGeometry{Geometry: buf.GetGeometry()}}
		return mline, sql.InRange, nil
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
func (t MultiLineStringType) SQL(ctx *sql.Context, dest []byte, v interface{}) (sqltypes.Value, error) {
	if v == nil {
		return sqltypes.NULL, nil
	}

	v, _, err := t.Convert(ctx, v)
	if err != nil {
		return sqltypes.Value{}, nil
	}

	buf := v.(MultiLineString).Serialize()

	return sqltypes.MakeTrusted(sqltypes.Geometry, buf), nil
}

// String implements Type interface.
func (t MultiLineStringType) String() string {
	return "multilinestring"
}

// ValueType implements Type interface.
func (t MultiLineStringType) ValueType() reflect.Type {
	return multilinestringValueType
}

// Zero implements Type interface.
func (t MultiLineStringType) Zero() interface{} {
	geosContext := geosenv.AcquireContext()
	defer geosenv.ReleaseContext(geosContext)
	return MultiLineString{BaseGeometry{Geometry: geosContext.NewEmptyCollection(geos.TypeIDLineString)}}
}

// SetSRID implements SpatialColumnType interface.
func (t MultiLineStringType) SetSRID(v int) sql.Type {
	t.SRID = v
	return t
}
