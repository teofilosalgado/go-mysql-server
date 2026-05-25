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

// STEquals is a function that returns the STEquals of a LineString
type STEquals struct {
	expression.BinaryExpressionStub
}

var _ sql.FunctionExpression = (*STEquals)(nil)

// NewSTEquals creates a new STEquals expression.
func NewSTEquals(ctx *sql.Context, g1, g2 sql.Expression) sql.Expression {
	return &STEquals{
		expression.BinaryExpressionStub{
			LeftChild:  g1,
			RightChild: g2,
		},
	}
}

// FunctionName implements sql.FunctionExpression
func (s *STEquals) FunctionName() string {
	return "st_equals"
}

// Description implements sql.FunctionExpression
func (s *STEquals) Description() string {
	return "returns 1 or 0 to indicate whether g1 is spatially equal to g2."
}

// Type implements the sql.Expression interface.
func (s *STEquals) Type(ctx *sql.Context) sql.Type {
	return types.Boolean
}

func (s *STEquals) String() string {
	return fmt.Sprintf("ST_EQUALS(%s, %s)", s.LeftChild, s.RightChild)
}

// WithChildren implements the Expression interface.
func (s *STEquals) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	if len(children) != 2 {
		return nil, sql.ErrInvalidChildrenNumber.New(s, len(children), 2)
	}
	return NewSTEquals(ctx, children[0], children[1]), nil
}

// Eval implements the sql.Expression interface.
func (s *STEquals) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	geom1, err := s.LeftChild.Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	geom2, err := s.RightChild.Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	if geom1 == nil || geom2 == nil {
		return nil, nil
	}

	g1, err := types.UnwrapGeometry(ctx, geom1)
	if err != nil {
		return nil, sql.ErrInvalidGISData.New(s.FunctionName())
	}

	g2, err := types.UnwrapGeometry(ctx, geom2)
	if err != nil {
		return nil, sql.ErrInvalidGISData.New(s.FunctionName())
	}

	if g1 == nil || g2 == nil {
		return nil, nil
	}

	return g1.GetGeometry().Equals(g2.GetGeometry()), nil
}
