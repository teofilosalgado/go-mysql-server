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

	"github.com/dolthub/go-mysql-server/sql/types"

	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/expression"
)

// MBRIntersects is a function that returns 1 or 0 to indicate whether the minimum bounding rectangles of the two geometries g1 and g2 intersect.
type MBRIntersects struct {
	expression.BinaryExpressionStub
}

var _ sql.FunctionExpression = (*MBRIntersects)(nil)
var _ sql.CollationCoercible = (*MBRIntersects)(nil)

// NewMBRIntersects creates a new MBRIntersects expression.
func NewMBRIntersects(ctx *sql.Context, g1, g2 sql.Expression) sql.Expression {
	return &MBRIntersects{
		expression.BinaryExpressionStub{
			LeftChild:  g1,
			RightChild: g2,
		},
	}
}

// FunctionName implements sql.FunctionExpression
func (i *MBRIntersects) FunctionName() string {
	return "mbrintersects"
}

// Description implements sql.FunctionExpression
func (i *MBRIntersects) Description() string {
	return "Returns 1 or 0 to indicate whether the minimum bounding rectangles of the two geometries g1 and g2 intersect."
}

// Type implements the sql.Expression interface.
func (i *MBRIntersects) Type(ctx *sql.Context) sql.Type {
	return types.Boolean
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*MBRIntersects) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 5
}

func (i *MBRIntersects) String() string {
	return fmt.Sprintf("%s(%s,%s)", i.FunctionName(), i.LeftChild, i.RightChild)
}

func (i *MBRIntersects) DebugString(ctx *sql.Context) string {
	return fmt.Sprintf("%s(%s,%s)", i.FunctionName(), sql.DebugString(ctx, i.LeftChild), sql.DebugString(ctx, i.RightChild))
}

// WithChildren implements the Expression interface.
func (i *MBRIntersects) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	if len(children) != 2 {
		return nil, sql.ErrInvalidChildrenNumber.New(i, len(children), 2)
	}
	return NewMBRIntersects(ctx, children[0], children[1]), nil
}

// Eval implements the sql.Expression interface.
func (i *MBRIntersects) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	geom1, err := i.LeftChild.Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	geom2, err := i.RightChild.Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	if geom1 == nil || geom2 == nil {
		return nil, nil
	}

	g1, err := types.UnwrapGeometry(ctx, geom1)
	if err != nil {
		return nil, sql.ErrInvalidGISData.New(i.FunctionName())
	}

	g2, err := types.UnwrapGeometry(ctx, geom2)
	if err != nil {
		return nil, sql.ErrInvalidGISData.New(i.FunctionName())
	}

	if g1 == nil || g2 == nil {
		return nil, nil
	}

	return g1.GetGeometry().Envelope().Intersects(g2.GetGeometry().Envelope()), nil
}
