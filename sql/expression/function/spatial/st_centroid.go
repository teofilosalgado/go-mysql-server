// Copyright 2025 Dolthub, Inc.
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
)

// Centroid is a function that returns the mathematical centroid of a geometry.
type Centroid struct {
	expression.UnaryExpressionStub
}

var _ sql.FunctionExpression = (*Centroid)(nil)
var _ sql.CollationCoercible = (*Centroid)(nil)

// NewCentroid creates a new Centroid expression.
func NewCentroid(ctx *sql.Context, e sql.Expression) sql.Expression {
	return &Centroid{expression.UnaryExpressionStub{Child: e}}
}

// FunctionName implements sql.FunctionExpression
func (c *Centroid) FunctionName() string {
	return "st_centroid"
}

// Description implements sql.FunctionExpression
func (c *Centroid) Description() string {
	return "returns the mathematical centroid for the geometry value."
}

// IsNullable implements the sql.Expression interface.
func (c *Centroid) IsNullable(ctx *sql.Context) bool {
	return c.Child.IsNullable(ctx)
}

// Type implements the sql.Expression interface.
func (c *Centroid) Type(ctx *sql.Context) sql.Type {
	return types.PointType{}
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*Centroid) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 4
}

func (c *Centroid) String() string {
	return fmt.Sprintf("%s(%s)", c.FunctionName(), c.Child.String())
}

// WithChildren implements the Expression interface.
func (c *Centroid) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	if len(children) != 1 {
		return nil, sql.ErrInvalidChildrenNumber.New(c, len(children), 1)
	}
	return NewCentroid(ctx, children[0]), nil
}

// Eval implements the sql.Expression interface.
func (c *Centroid) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	// Evaluate argument
	v, err := c.Child.Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	// Return nil if argument is nil
	if v == nil {
		return nil, nil
	}

	gv, err := types.UnwrapGeometry(ctx, v)
	if err != nil {
		return nil, sql.ErrInvalidArgument.New(c.FunctionName())
	}

	return types.Point{BaseGeometry: types.BaseGeometry{Geometry: gv.GetGeometry().Centroid()}}, nil
}
