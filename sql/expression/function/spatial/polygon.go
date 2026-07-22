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

	"github.com/dolthub/go-mysql-server/sql/expression"
	"github.com/dolthub/go-mysql-server/sql/geosenv"
	"github.com/dolthub/go-mysql-server/sql/types"

	"github.com/dolthub/go-mysql-server/sql"
)

// Polygon is a function that returns a Polygon.
type Polygon struct {
	expression.NaryExpression
}

var _ sql.FunctionExpression = (*Polygon)(nil)
var _ sql.CollationCoercible = (*Polygon)(nil)

// NewPolygon creates a new polygon expression.
func NewPolygon(ctx *sql.Context, args ...sql.Expression) (sql.Expression, error) {
	if len(args) < 1 {
		return nil, sql.ErrInvalidArgumentNumber.New("Polygon", "1 or more", len(args))
	}
	return &Polygon{expression.NaryExpression{ChildExpressions: args}}, nil
}

// FunctionName implements sql.FunctionExpression
func (p *Polygon) FunctionName() string {
	return "polygon"
}

// Description implements sql.FunctionExpression
func (p *Polygon) Description() string {
	return "returns a new polygon."
}

// Type implements the sql.Expression interface.
func (p *Polygon) Type(ctx *sql.Context) sql.Type {
	return types.PolygonType{}
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*Polygon) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 5
}

func (p *Polygon) String() string {
	var args = make([]string, len(p.ChildExpressions))
	for i, arg := range p.ChildExpressions {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", p.FunctionName(), strings.Join(args, ","))
}

// WithChildren implements the Expression interface.
func (p *Polygon) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	return NewPolygon(ctx, children...)
}

// Eval implements the sql.Expression interface.
func (p *Polygon) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	// Allocate array of lines
	var coordinates = make([][][]float64, len(p.ChildExpressions))

	// Go through each argument
	for i, arg := range p.ChildExpressions {
		// Evaluate argument
		val, err := arg.Eval(ctx, row)
		if err != nil {
			return nil, err
		}
		// Must be of type linestring, throw error otherwise
		gv, err := types.UnwrapGeometry(ctx, val)
		if err != nil {
			return nil, sql.ErrIllegalGISValue.New(val)
		}
		switch v := gv.(type) {
		case types.LineString:
			coordinates[i] = make([][]float64, 2)
			coordinateSequence := v.Geometry.CoordSeq()
			for j := range coordinateSequence.Size() {
				coordinates[i][j] = make([]float64, 2)
				coordinates[i][j][0] = coordinateSequence.X(j)
				coordinates[i][j][1] = coordinateSequence.Y(j)
			}
		default:
			return nil, sql.ErrIllegalGISValue.New(v)
		}
	}
	geosContext := geosenv.AcquireContext()
	defer geosenv.ReleaseContext(geosContext)
	geometry := geosContext.NewPolygon(coordinates)
	return types.Polygon{BaseGeometry: types.BaseGeometry{Geometry: geometry}}, nil
}
