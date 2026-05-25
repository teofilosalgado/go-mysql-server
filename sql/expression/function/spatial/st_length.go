// Copyright 2020-2022 Dolthub, Inc.
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
	"reflect"
	"strings"

	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/expression"
	"github.com/dolthub/go-mysql-server/sql/types"
)

// STLength is a function that returns the STLength of a LineString
type STLength struct {
	expression.NaryExpression
}

var _ sql.FunctionExpression = (*STLength)(nil)
var _ sql.CollationCoercible = (*STLength)(nil)

// NewSTLength creates a new STLength expression.
func NewSTLength(ctx *sql.Context, args ...sql.Expression) (sql.Expression, error) {
	if len(args) != 1 && len(args) != 2 {
		return nil, sql.ErrInvalidArgumentNumber.New("ST_LENGTH", "1 or 2", len(args))
	}
	return &STLength{expression.NaryExpression{ChildExpressions: args}}, nil
}

// FunctionName implements sql.FunctionExpression
func (s *STLength) FunctionName() string {
	return "st_length"
}

// Description implements sql.FunctionExpression
func (s *STLength) Description() string {
	return "returns the length of the given linestring. If given a unit argument, will return the length in those units"
}

// Type implements the sql.Expression interface.
func (s *STLength) Type(ctx *sql.Context) sql.Type {
	return types.Float64
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*STLength) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 5
}

func (s *STLength) String() string {
	var args = make([]string, len(s.ChildExpressions))
	for i, arg := range s.ChildExpressions {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", s.FunctionName(), strings.Join(args, ","))
}

// WithChildren implements the Expression interface.
func (s *STLength) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	return NewSTLength(ctx, children...)
}

// Eval implements the sql.Expression interface.
func (s *STLength) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	ls, err := s.ChildExpressions[0].Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	if ls == nil {
		return nil, nil
	}

	gv, err := types.UnwrapGeometry(ctx, ls)
	if err != nil {
		return nil, sql.ErrInvalidGISData.New(s.FunctionName())
	}

	if len(s.ChildExpressions) == 1 {
		return gv.GetGeometry().Length(), nil
	}

	unit, err := s.ChildExpressions[1].Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	unit, _, err = types.Text.Convert(ctx, unit)
	if err != nil {
		return nil, sql.ErrInvalidType.New(reflect.TypeOf(unit))
	}

	// TODO: Implement unit argument
	return nil, nil
}
