// Copyright 2023 Dolthub, Inc.
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

// Within is a function that true if left is spatially within right
type Within struct {
	expression.BinaryExpressionStub
}

var _ sql.FunctionExpression = (*Within)(nil)
var _ sql.CollationCoercible = (*Within)(nil)

// NewWithin creates a new Within expression.
func NewWithin(ctx *sql.Context, g1, g2 sql.Expression) sql.Expression {
	return &Within{
		expression.BinaryExpressionStub{
			LeftChild:  g1,
			RightChild: g2,
		},
	}
}

// FunctionName implements sql.FunctionExpression
func (w *Within) FunctionName() string {
	return "st_within"
}

// Description implements sql.FunctionExpression
func (w *Within) Description() string {
	return "returns 1 or 0 to indicate whether g1 is spatially within g2. This tests the opposite relationship as st_contains()."
}

// Type implements the sql.Expression interface.
func (w *Within) Type(ctx *sql.Context) sql.Type {
	return types.Boolean
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*Within) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 5
}

func (w *Within) String() string {
	return fmt.Sprintf("%s(%s,%s)", w.FunctionName(), w.LeftChild, w.RightChild)
}

// WithChildren implements the Expression interface.
func (w *Within) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	if len(children) != 2 {
		return nil, sql.ErrInvalidChildrenNumber.New(w, len(children), 2)
	}
	return NewWithin(ctx, children[0], children[1]), nil
}

// Eval implements the sql.Expression interface.
func (w *Within) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	geom1, err := w.LeftChild.Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	geom2, err := w.RightChild.Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	if geom1 == nil || geom2 == nil {
		return nil, nil
	}

	g1, err := types.UnwrapGeometry(ctx, geom1)
	if err != nil {
		return nil, sql.ErrInvalidGISData.New(w.FunctionName())
	}

	g2, err := types.UnwrapGeometry(ctx, geom2)
	if err != nil {
		return nil, sql.ErrInvalidGISData.New(w.FunctionName())
	}

	if g1 == nil || g2 == nil {
		return nil, nil
	}

	return g1.GetGeometry().Within(g2.GetGeometry()), nil
}
