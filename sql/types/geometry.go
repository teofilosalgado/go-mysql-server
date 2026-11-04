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
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"reflect"
	"regexp"

	"github.com/dolthub/vitess/go/sqltypes"
	"github.com/dolthub/vitess/go/vt/proto/query"
	"github.com/twpayne/go-geos"
	"gopkg.in/src-d/go-errors.v1"

	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/geosenv"
)

// GeometryType represents the GEOMETRY type.
// https://dev.mysql.com/doc/refman/8.0/en/gis-class-geometry.html
// The type of the returned value is one of the following (each implements GeometryValue): Point, Polygon, LineString.
type GeometryType struct {
	SRID int
}

// GeometryValue is the value type returned from GeometryType, which is an interface over the following types:
// Point, Polygon, LineString, MultiPoint, MultiLineString, MultiPolygon, GeometryCollection.
type GeometryValue interface {
	implementsGeometryValue()
	GetWKB() []byte
	SetWKB([]byte) GeometryValue
	GetGeometry(*geos.Context) (*geos.Geom, error)
	GetSRID() int
	SetSRID(srid int) GeometryValue
	Serialize() []byte
	WriteData(buf []byte) int
	Swap() GeometryValue
	BBox() (float64, float64, float64, float64)
}

type BaseGeometry struct {
	WKB  []byte
	SRID int
}

var _ GeometryValue = BaseGeometry{}

func (bg BaseGeometry) GetWKB() []byte {
	return bg.WKB
}

func (bg BaseGeometry) SetWKB(wkb []byte) GeometryValue {
	bg.WKB = wkb
	return bg
}

func (bg BaseGeometry) GetGeometry(geosContext *geos.Context) (*geos.Geom, error) {
	geometry, err := geosContext.NewGeomFromWKB(bg.WKB)
	if err != nil {
		return nil, err
	}
	geometry.SetSRID(bg.SRID)
	return geometry, nil
}

func (bg BaseGeometry) GetSRID() int {
	return bg.SRID
}

func (bg BaseGeometry) SetSRID(srid int) GeometryValue {
	bg.SRID = srid
	return bg
}

func (bg BaseGeometry) Swap() GeometryValue {
	geosContext := geosenv.AcquireContext()
	defer geosenv.ReleaseContext(geosContext)

	geometry, err := bg.GetGeometry(geosContext)
	if err != nil {
		return bg
	}

	swappedGeomWKT := swapRegex.ReplaceAllString(geometry.ToWKT(), "$2 $1")
	swappedGeom, err := geosContext.NewGeomFromWKT(swappedGeomWKT)
	if err != nil {
		return bg
	}
	bg.WKB = swappedGeom.ToWKB()
	return bg
}

func (bg BaseGeometry) BBox() (float64, float64, float64, float64) {
	geosContext := geosenv.AcquireContext()
	defer geosenv.ReleaseContext(geosContext)

	geometry, err := bg.GetGeometry(geosContext)
	if err != nil {
		return 0, 0, 0, 0
	}
	bounds := geometry.Bounds()
	return bounds.MinX, bounds.MinY, bounds.MaxX, bounds.MaxY
}

func (bg BaseGeometry) implementsGeometryValue() {}

func (bg BaseGeometry) Serialize() (buf []byte) {
	totalSize := SRIDSize + len(bg.WKB)
	buf = make([]byte, totalSize)
	bg.WriteData(buf)
	return buf
}

func (bg BaseGeometry) WriteData(buf []byte) int {
	binary.LittleEndian.PutUint32(buf[0:SRIDSize], uint32(bg.SRID))
	bytesCopied := copy(buf[SRIDSize:], bg.WKB)
	return SRIDSize + bytesCopied
}

// UnwrapGeometry unwraps a value that may be a sql.AnyWrapper (e.g. adaptive/out-of-band storage)
// and returns the underlying GeometryValue. If the value is already a GeometryValue, it is returned
// directly. Returns ErrNotGeometry if the value cannot be converted.
func UnwrapGeometry(ctx context.Context, v interface{}) (GeometryValue, error) {
	if gv, ok := v.(GeometryValue); ok {
		return gv, nil
	}
	unwrapped, err := sql.UnwrapAny(ctx, v)
	if err != nil {
		return nil, err
	}
	if gv, ok := unwrapped.(GeometryValue); ok {
		return gv, nil
	}
	return nil, ErrNotGeometry.New(v)
}

var _ sql.Type = GeometryType{}
var _ sql.SpatialColumnType = GeometryType{}
var _ sql.CollationCoercible = GeometryType{}

var (
	ErrNotGeometry    = errors.NewKind("Value of type %T is not a geometry")
	geometryValueType = reflect.TypeOf((*GeometryValue)(nil)).Elem()
	swapRegex         = regexp.MustCompile(`([-\d.]+)\s+([-\d.]+)`)
)

const (
	CartesianSRID  = uint32(0)
	GeoSpatialSRID = uint32(4326)
)

const (
	SRIDSize       = 4
	EndianSize     = 1
	TypeSize       = 4
	EWKBHeaderSize = SRIDSize + EndianSize + TypeSize
	WKBHeaderSize  = EndianSize + TypeSize

	PointSize             = 16
	CountSize             = 4
	GeometryMaxByteLength = 4*(1024*1024*1024) - 1
)

// Type IDs
const (
	WKBUnknown = iota
	WKBPointID
	WKBLineID
	WKBPolyID
	WKBMultiPointID
	WKBMultiLineID
	WKBMultiPolyID
	WKBGeomCollID
)

func deserializeWKB(mySQLBytes []byte, wkbGeometryTypeId int) []byte {
	wkb := make([]byte, WKBHeaderSize+len(mySQLBytes))
	wkb[0] = 1
	binary.LittleEndian.PutUint32(wkb[1:WKBHeaderSize], uint32(wkbGeometryTypeId))
	copy(wkb[WKBHeaderSize:], mySQLBytes)
	return wkb
}

// DeserializeEWKBHeader parses the header portion of a byte array in EWKB format to extract endianness and type
func DeserializeEWKBHeader(buf []byte) (srid uint32, bigEndian bool, typ uint32, err error) {
	// Must be right length
	if len(buf) < EWKBHeaderSize {
		return 0, false, 0, sql.ErrInvalidGISData.New("DeserializeEWKBHeader")
	}
	srid = binary.LittleEndian.Uint32(buf) // First 4 bytes is SRID always in little endian
	buf = buf[SRIDSize:]                   // Shift pointer over
	bigEndian = buf[0] == 0                // Next byte is endianness
	buf = buf[EndianSize:]                 // Shift pointer over
	if bigEndian {                         // Next 4 bytes is type
		typ = binary.BigEndian.Uint32(buf)
	} else {
		typ = binary.LittleEndian.Uint32(buf)
	}

	return
}

// DeserializeWKBHeader parses the header potion of a byte array in WKB format
// There is no SRID
func DeserializeWKBHeader(buf []byte) (bigEndian bool, typ uint32, err error) {
	// Must be right length
	if len(buf) < (EndianSize + TypeSize) {
		return false, 0, sql.ErrInvalidGISData.New("DeserializeWKBHeader")
	}

	bigEndian = buf[0] == 0 // First byte is byte order
	buf = buf[EndianSize:]  // Shift pointer over
	if bigEndian {          // Next 4 bytes is geometry type
		typ = binary.BigEndian.Uint32(buf)
	} else {
		typ = binary.LittleEndian.Uint32(buf)
	}

	return
}

// DeserializePoint parses the data portion of a byte array in WKB format to a Point object
func DeserializePoint(buf []byte, isBig bool, srid uint32) (Point, int, error) {
	wkb := deserializeWKB(buf, WKBPointID)
	return Point{BaseGeometry{WKB: wkb, SRID: int(srid)}}, len(buf), nil
}

// DeserializeLine parses the data portion of a byte array in WKB format to a LineString object
func DeserializeLine(buf []byte, isBig bool, srid uint32) (LineString, int, error) {
	wkb := deserializeWKB(buf, WKBLineID)
	return LineString{BaseGeometry{WKB: wkb, SRID: int(srid)}}, len(buf), nil
}

// DeserializePoly parses the data portion of a byte array in WKB format to a Polygon object
func DeserializePoly(buf []byte, isBig bool, srid uint32) (Polygon, int, error) {
	wkb := deserializeWKB(buf, WKBPolyID)
	return Polygon{BaseGeometry{WKB: wkb, SRID: int(srid)}}, len(buf), nil
}

// DeserializeMPoint parses the data portion of a byte array in WKB format to a MultiPoint object
func DeserializeMPoint(buf []byte, isBig bool, srid uint32) (MultiPoint, int, error) {
	wkb := deserializeWKB(buf, WKBMultiPointID)
	return MultiPoint{BaseGeometry{WKB: wkb, SRID: int(srid)}}, len(buf), nil
}

// DeserializeMLine parses the data portion of a byte array in WKB format to a MultiLineString object
func DeserializeMLine(buf []byte, isBig bool, srid uint32) (MultiLineString, int, error) {
	wkb := deserializeWKB(buf, WKBMultiLineID)
	return MultiLineString{BaseGeometry{WKB: wkb, SRID: int(srid)}}, len(buf), nil
}

// DeserializeMPoly parses the data portion of a byte array in WKB format to a MultiPolygon object
func DeserializeMPoly(buf []byte, isBig bool, srid uint32) (MultiPolygon, int, error) {
	wkb := deserializeWKB(buf, WKBMultiPolyID)
	return MultiPolygon{BaseGeometry{WKB: wkb, SRID: int(srid)}}, len(buf), nil
}

// DeserializeGeomColl parses the data portion of a byte array in WKB format to a GeometryCollection object
func DeserializeGeomColl(buf []byte, isBig bool, srid uint32) (GeomColl, int, error) {
	wkb := deserializeWKB(buf, WKBGeomCollID)
	return GeomColl{BaseGeometry{WKB: wkb, SRID: int(srid)}}, len(buf), nil
}

// WriteEWKBHeader will write EWKB header to the given buffer
func WriteEWKBHeader(buf []byte, srid, typ uint32) {
	binary.LittleEndian.PutUint32(buf, srid) // always write SRID in little endian
	buf = buf[SRIDSize:]                     // shift
	buf[0] = 1                               // always write in little endian
	buf = buf[EndianSize:]                   // shift
	binary.LittleEndian.PutUint32(buf, typ)  // write geometry type
}

// Compare implements Type interface.
func (t GeometryType) Compare(s context.Context, a interface{}, b interface{}) (int, error) {
	if hasNulls, res := CompareNulls(a, b); hasNulls {
		return res, nil
	}

	aa, err := UnwrapGeometry(s, a)
	if err != nil {
		return 0, err
	}

	bb, err := UnwrapGeometry(s, b)
	if err != nil {
		return 0, err
	}

	return bytes.Compare(aa.Serialize(), bb.Serialize()), nil
}

// Convert implements Type interface.
func (t GeometryType) Convert(ctx context.Context, v interface{}) (interface{}, sql.ConvertInRange, error) {
	if v == nil {
		return nil, sql.InRange, nil
	}
	switch val := v.(type) {
	case []byte:
		srid, isBig, geomType, err := DeserializeEWKBHeader(val)
		if err != nil {
			return nil, sql.InRange, err
		}
		val = val[EWKBHeaderSize:]

		var geom interface{}
		switch geomType {
		case WKBPointID:
			geom, _, err = DeserializePoint(val, isBig, srid)
		case WKBLineID:
			geom, _, err = DeserializeLine(val, isBig, srid)
		case WKBPolyID:
			geom, _, err = DeserializePoly(val, isBig, srid)
		case WKBMultiPointID:
			geom, _, err = DeserializeMPoint(val, isBig, srid)
		case WKBMultiLineID:
			geom, _, err = DeserializeMLine(val, isBig, srid)
		case WKBMultiPolyID:
			geom, _, err = DeserializeMPoly(val, isBig, srid)
		case WKBGeomCollID:
			geom, _, err = DeserializeGeomColl(val, isBig, srid)
		default:
			return nil, sql.InRange, sql.ErrInvalidGISData.New("GeometryType.Convert")
		}
		if err != nil {
			return nil, sql.InRange, err
		}
		return geom, sql.InRange, nil
	case string:
		return t.Convert(ctx, []byte(val))
	case GeometryValue:
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

// Equals implements the Type interface.
func (t GeometryType) Equals(otherType sql.Type) (ok bool) {
	_, ok = otherType.(GeometryType)
	return
}

// MaxTextResponseByteLength implements the Type interface
func (t GeometryType) MaxTextResponseByteLength(*sql.Context) uint32 {
	return GeometryMaxByteLength
}

// Promote implements the Type interface.
func (t GeometryType) Promote() sql.Type {
	return t
}

// SQL implements Type interface.
func (t GeometryType) SQL(ctx *sql.Context, dest []byte, v interface{}) (sqltypes.Value, error) {
	if v == nil {
		return sqltypes.NULL, nil
	}

	v, _, err := t.Convert(ctx, v)
	if err != nil {
		return sqltypes.Value{}, nil
	}

	buf := v.(GeometryValue).Serialize()

	return sqltypes.MakeTrusted(sqltypes.Geometry, buf), nil
}

// String implements Type interface.
func (t GeometryType) String() string {
	return "geometry"
}

// Type implements Type interface.
func (t GeometryType) Type() query.Type {
	return sqltypes.Geometry
}

// ValueType implements Type interface.
func (t GeometryType) ValueType() reflect.Type {
	return geometryValueType
}

// Zero implements Type interface.
func (t GeometryType) Zero() interface{} {
	// MySQL throws an error for INSERT IGNORE, UPDATE IGNORE, etc. if the geometry type cannot be parsed:
	// ERROR 1416 (22003): Cannot get geometry object from data you send to the GEOMETRY field
	// So, we don't implement a zero type for this function.
	return nil
}

// CollationCoercibility implements sql.CollationCoercible interface.
func (GeometryType) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 5
}

// GetSpatialTypeSRID implements SpatialColumnType interface.
func (t GeometryType) GetSpatialTypeSRID() (int, bool) {
	return t.SRID, t.SRID != 0
}

// SetSRID implements SpatialColumnType interface.
func (t GeometryType) SetSRID(v int) sql.Type {
	t.SRID = v
	return t
}

// MatchSRID implements SpatialColumnType interface
func (t GeometryType) MatchSRID(v interface{}) error {
	if t.SRID == 0 {
		return nil
	}
	// if matched with SRID value of row value
	var srid int
	switch val := v.(type) {
	case GeometryValue:
		srid = val.GetSRID()
	default:
		return ErrNotGeometry.New(v)
	}
	if t.SRID == srid {
		return nil
	}
	return sql.ErrNotMatchingSRID.New(srid, t.SRID)
}

func ValidateSRID(srid int, funcName string) error {
	if srid < 0 || srid > math.MaxUint32 {
		return sql.ErrInvalidSRID.New(funcName)
	}
	if _, ok := SupportedSRIDs[uint32(srid)]; !ok {
		return sql.ErrNoSRID.New(srid)
	}

	return nil
}
