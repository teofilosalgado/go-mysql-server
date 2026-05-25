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

// GeomCollType represents the GeometryCollection type.
// https://dev.mysql.com/doc/refman/8.0/en/gis-class-point.html
// The type of the returned value is GeomColl.
type GeomCollType struct {
	GeometryType
}

// GeomColl is the value type returned from GeomCollType. Implements GeometryValue.
type GeomColl struct {
	BaseGeometry
}

var _ sql.Type = GeomCollType{}
var _ sql.SpatialColumnType = GeomCollType{}
var _ sql.CollationCoercible = GeomCollType{}
var _ GeometryValue = GeomColl{}

var (
	geomcollValueType = reflect.TypeOf(GeomColl{})
)

// Convert implements Type interface.
func (t GeomCollType) Convert(ctx context.Context, v interface{}) (interface{}, sql.ConvertInRange, error) {
	// Allow null
	if v == nil {
		return nil, sql.InRange, nil
	}
	// Handle conversions
	switch val := v.(type) {
	case []byte:
		// Parse header
		srid, isBig, geomType, err := DeserializeEWKBHeader(val)
		if err != nil {
			return nil, sql.InRange, err
		}
		// Throw error if not marked as geometry collection
		if geomType != WKBGeomCollID {
			return nil, sql.InRange, sql.ErrInvalidGISData.New("GeomCollType.Convert")
		}
		// Parse data section
		geom, _, err := DeserializeGeomColl(val[EWKBHeaderSize:], isBig, srid)
		if err != nil {
			return nil, sql.InRange, err
		}
		return geom, sql.InRange, nil
	case string:
		return t.Convert(ctx, []byte(val))
	case GeomColl:
		if err := t.MatchSRID(val); err != nil {
			return nil, sql.InRange, err
		}
		return val, sql.InRange, nil
	case sql.AnyWrapper:
		unwrapped, err := val.UnwrapAny(ctx)
		if err != nil {
			return nil, sql.InRange, err
		}
		return t.Convert(ctx, unwrapped)
	default:
		return nil, sql.InRange, sql.ErrSpatialTypeConversion.New()
	}
}

// SQL implements Type interface.
func (t GeomCollType) SQL(ctx *sql.Context, dest []byte, v interface{}) (sqltypes.Value, error) {
	if v == nil {
		return sqltypes.NULL, nil
	}

	v, _, err := t.Convert(ctx, v)
	if err != nil {
		return sqltypes.Value{}, nil
	}

	buf := v.(GeomColl).Serialize()

	return sqltypes.MakeTrusted(sqltypes.Geometry, buf), nil
}

// String implements Type interface.
func (t GeomCollType) String() string {
	return "geometrycollection"
}

// ValueType implements Type interface.
func (t GeomCollType) ValueType() reflect.Type {
	return geomcollValueType
}

// Zero implements Type interface.
func (t GeomCollType) Zero() interface{} {
	geosContext := geos.NewContext()
	return MultiPolygon{BaseGeometry{Geometry: geosContext.NewEmptyCollection(geos.TypeIDGeometryCollection)}}
}
