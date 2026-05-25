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

// ConvexHull is a function that returns the convex hull of a geometry value.
type ConvexHull struct {
	expression.UnaryExpressionStub
}

var _ sql.FunctionExpression = (*ConvexHull)(nil)
var _ sql.CollationCoercible = (*ConvexHull)(nil)

// NewConvexHull creates a new ConvexHull expression.
func NewConvexHull(ctx *sql.Context, e sql.Expression) sql.Expression {
	return &ConvexHull{expression.UnaryExpressionStub{Child: e}}
}

// FunctionName implements sql.FunctionExpression
func (c *ConvexHull) FunctionName() string {
	return "st_convexhull"
}

// Description implements sql.FunctionExpression
func (c *ConvexHull) Description() string {
	return "returns the convex hull of the geometry value."
}

// IsNullable implements the sql.Expression interface.
func (c *ConvexHull) IsNullable(ctx *sql.Context) bool {
	return c.Child.IsNullable(ctx)
}

// Type implements the sql.Expression interface.
func (c *ConvexHull) Type(ctx *sql.Context) sql.Type {
	return types.GeometryType{}
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*ConvexHull) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 4
}

func (c *ConvexHull) String() string {
	return fmt.Sprintf("%s(%s)", c.FunctionName(), c.Child.String())
}

// WithChildren implements the Expression interface.
func (c *ConvexHull) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	if len(children) != 1 {
		return nil, sql.ErrInvalidChildrenNumber.New(c, len(children), 1)
	}
	return NewConvexHull(ctx, children[0]), nil
}

// Eval implements the sql.Expression interface.
func (c *ConvexHull) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
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

	return types.Polygon{BaseGeometry: types.BaseGeometry{Geometry: gv.GetGeometry().ConvexHull()}}, nil
}
