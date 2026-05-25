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
	"reflect"
	"strings"

	"github.com/dolthub/go-mysql-server/sql"
	"github.com/dolthub/go-mysql-server/sql/expression"
	"github.com/dolthub/go-mysql-server/sql/types"
)

// Distance is a function that returns the shortest distance between two geometries
type Distance struct {
	expression.NaryExpression
}

var _ sql.FunctionExpression = (*Distance)(nil)
var _ sql.CollationCoercible = (*Distance)(nil)

// NewDistance creates a new Distance expression.
func NewDistance(ctx *sql.Context, args ...sql.Expression) (sql.Expression, error) {
	if len(args) != 2 && len(args) != 3 {
		return nil, sql.ErrInvalidArgumentNumber.New("ST_DISTANCE", "2 or 3", len(args))
	}
	return &Distance{expression.NaryExpression{ChildExpressions: args}}, nil
}

// FunctionName implements sql.FunctionExpression
func (d *Distance) FunctionName() string {
	return "st_distance"
}

// Description implements sql.FunctionExpression
func (d *Distance) Description() string {
	return "returns the distance between g1 and g2."
}

// Type implements the sql.Expression interface.
func (d *Distance) Type(ctx *sql.Context) sql.Type {
	return types.Float64
}

// CollationCoercibility implements the interface sql.CollationCoercible.
func (*Distance) CollationCoercibility(ctx *sql.Context) (collation sql.CollationID, coercibility byte) {
	return sql.Collation_binary, 5
}

func (d *Distance) String() string {
	var args = make([]string, len(d.ChildExpressions))
	for i, arg := range d.ChildExpressions {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", d.FunctionName(), strings.Join(args, ","))
}

// WithChildren implements the Expression interface.
func (d *Distance) WithChildren(ctx *sql.Context, children ...sql.Expression) (sql.Expression, error) {
	return NewDistance(ctx, children...)
}

// Eval implements the sql.Expression interface.
func (d *Distance) Eval(ctx *sql.Context, row sql.Row) (interface{}, error) {
	g1, err := d.ChildExpressions[0].Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	g2, err := d.ChildExpressions[1].Eval(ctx, row)
	if err != nil {
		return nil, err
	}

	if g1 == nil || g2 == nil {
		return nil, nil
	}

	geom1, err := types.UnwrapGeometry(ctx, g1)
	if err != nil {
		return nil, sql.ErrInvalidGISData.New(d.FunctionName())
	}

	geom2, err := types.UnwrapGeometry(ctx, g2)
	if err != nil {
		return nil, sql.ErrInvalidGISData.New(d.FunctionName())
	}

	distance := geom1.GetGeometry().Distance(geom2.GetGeometry())
	if len(d.ChildExpressions) == 2 {
		return distance, nil
	}

	unit, err := d.ChildExpressions[2].Eval(ctx, row)
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
