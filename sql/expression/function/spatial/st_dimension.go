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

	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/expression"
	"github.com/dolthub/go-mysql-server/sql/types"
	"github.com/twpayne/go-geos"
)

// Dimension is a function that converts a spatial type into WKT format (alias for AsText)
type Dimension struct {
	expression.UnaryExpressionStub
}

var _ sql.FunctionExpression = (*Dimension)(nil)
var _ sql.CollationCoercible = (*Dimension)(nil)

// NewDimension creates a new point expression.
func NewDimension(ctx *sql.Context, e sql.Expression) sql.Expression {
	return &Dimension{expression.UnaryExpressionStub{Child: e}}
}

// FunctionName implements sql.FunctionExpression
func (p *Dimension) FunctionName() string {
	return "st_dimension"
}

// Description implements sql.FunctionExpression
func (p *Dimension) Description() string {
	return "returns the dimension of the geometry given."
}

// IsNullable implements the sql.Expression interface.
func (p *Dimension) IsNullable(ctx *sql.Context) bool {
	return p.Child.IsNullable(ctx)
}

// Type implements the sql.Expression interface.
func (p *Dimension) Type(ctx *sql.Context) sql.Type {
	return types.Int32
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*Dimension) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 5
}

func (p *Dimension) String() string {
	return fmt.Sprintf("%s(%s)", p.FunctionName(), p.Child.String())
}

// WithChildren implements the Expression interface.
func (p *Dimension) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	if len(children) != 1 {
		return nil, sql.ErrInvalidChildrenNumber.New(p, len(children), 1)
	}
	return NewDimension(ctx, children[0]), nil
}

func FindDimension(bg types.GeometryValue) interface{} {
	geometryType := bg.GetGeometry().TypeID()
	switch geometryType {
	case geos.TypeIDPoint, geos.TypeIDMultiPoint:
		return 0
	case geos.TypeIDLineString, geos.TypeIDMultiLineString:
		return 1
	case geos.TypeIDPolygon, geos.TypeIDMultiPolygon:
		return 2
	case geos.TypeIDGeometryCollection:
		numGeometries := bg.GetGeometry().NumGeometries()
		maxDimension := 0
		for i := range numGeometries {
			currentGeometry := bg.GetGeometry().Geometry(i)
			currentGeometryDimension := FindDimension(types.BaseGeometry{Geometry: currentGeometry})
			if currentGeometryDimension == nil {
				return nil
			}
			if currentGeometryDimension.(int) > maxDimension {
				maxDimension = currentGeometryDimension.(int)
			}
		}
	default:
		return nil
	}
	return nil
}

// Eval implements the sql.Expression interface.
func (p *Dimension) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
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

	return FindDimension(gv), nil
}
