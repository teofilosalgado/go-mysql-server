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

// IsSimple is a function that returns whether a geometry value is simple.
// A geometry is simple if it has no anomalous geometric points, such as self-intersection or self-tangency.
type IsSimple struct {
	expression.UnaryExpressionStub
}

var _ sql.FunctionExpression = (*IsSimple)(nil)
var _ sql.CollationCoercible = (*IsSimple)(nil)

// NewIsSimple creates a new IsSimple expression.
func NewIsSimple(ctx *sql.Context, e sql.Expression) sql.Expression {
	return &IsSimple{expression.UnaryExpressionStub{Child: e}}
}

// FunctionName implements sql.FunctionExpression
func (s *IsSimple) FunctionName() string {
	return "st_issimple"
}

// Description implements sql.FunctionExpression
func (s *IsSimple) Description() string {
	return "returns whether the geometry value is simple (has no anomalous geometric points)."
}

// IsNullable implements the sql.Expression interface.
func (s *IsSimple) IsNullable(ctx *sql.Context) bool {
	return s.Child.IsNullable(ctx)
}

// Type implements the sql.Expression interface.
func (s *IsSimple) Type(ctx *sql.Context) sql.Type {
	return types.Boolean
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*IsSimple) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 5
}

func (s *IsSimple) String() string {
	return fmt.Sprintf("%s(%s)", s.FunctionName(), s.Child.String())
}

// WithChildren implements the Expression interface.
func (s *IsSimple) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	if len(children) != 1 {
		return nil, sql.ErrInvalidChildrenNumber.New(s, len(children), 1)
	}
	return NewIsSimple(ctx, children[0]), nil
}

// Eval implements the sql.Expression interface.
func (s *IsSimple) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	val, err := s.Child.Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	if val == nil {
		return nil, nil
	}

	gv, err := types.UnwrapGeometry(ctx, val)
	if err != nil {
		return nil, sql.ErrInvalidGISData.New(s.FunctionName())
	}

	return gv.GetGeometry().IsSimple(), nil
}
