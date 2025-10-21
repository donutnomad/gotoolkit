package field

import (
	"fmt"

	"github.com/donutnomad/gotoolkit/lib/gsql/internal/utils"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"gorm.io/gorm/clause"
)

// =, !=, IN, NOT IN, >, >=, <, <=
type comparableImpl[T any] struct {
	IField
}

func (f comparableImpl[T]) Eq(value T) Expression {
	return f.operateValue(value, "=")
}

func (f comparableImpl[T]) EqOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return emptyExpression
	}
	return f.Eq(value.MustGet())
}

func (f comparableImpl[T]) EqF(other IField) Expression {
	return f.operateField(other, "=")
}

func (f comparableImpl[T]) Not(value T) Expression {
	return f.operateValue(value, "!=")
}

func (f comparableImpl[T]) NotOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return emptyExpression
	}
	return f.Not(value.MustGet())
}

func (f comparableImpl[T]) NotF(other IField) Expression {
	return f.operateField(other, "!=")
}

func (f comparableImpl[T]) In(values ...T) Expression {
	if len(values) == 0 {
		return emptyExpression
	}
	return f.operateValue(lo.ToAnySlice(values), "IN")
}

func (f comparableImpl[T]) NotIn(values ...T) Expression {
	if len(values) == 0 {
		return emptyExpression
	}
	return f.operateValue(lo.ToAnySlice(values), "NOT IN")
}

func (f comparableImpl[T]) Gt(value T) Expression {
	return f.operateValue(value, ">")
}

func (f comparableImpl[T]) GtOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return emptyExpression
	}
	return f.Gt(value.MustGet())
}

func (f comparableImpl[T]) GtF(other IField) Expression {
	return f.operateField(other, ">")
}

func (f comparableImpl[T]) Gte(value T) Expression {
	return f.operateValue(value, ">=")
}

func (f comparableImpl[T]) GteOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return emptyExpression
	}
	return f.Gte(value.MustGet())
}

func (f comparableImpl[T]) GteF(other IField) Expression {
	return f.operateField(other, ">=")
}

func (f comparableImpl[T]) Lt(value T) Expression {
	return f.operateValue(value, "<")
}

func (f comparableImpl[T]) LtOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return emptyExpression
	}
	return f.Lt(value.MustGet())
}

func (f comparableImpl[T]) LtF(other IField) Expression {
	return f.operateField(other, "<")
}

func (f comparableImpl[T]) Lte(value T) Expression {
	return f.operateValue(value, "<=")
}

func (f comparableImpl[T]) LteOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return emptyExpression
	}
	return f.Lte(value.MustGet())
}

func (f comparableImpl[T]) LteF(other IField) Expression {
	return f.operateField(other, "<=")
}

// Between
// opFrom: >=,>,=,<=,<, default: >=
// opTo: >=,>,=,<=,<, default: <
func (f comparableImpl[T]) Between(from, to *T, op ...string) Expression {
	return f.BetweenOpt(Range[T]{
		From: mo.PointerToOption(from),
		To:   mo.PointerToOption(to),
	}, op...)
}

// BetweenOpt
// opFrom: >=,>,=,<=,<, default: >=
// opTo: >=,>,=,<=,<, default: <
func (f comparableImpl[T]) BetweenOpt(rng interface {
	FromValue() *T
	ToValue() *T
}, op ...string) Expression {
	var opFrom = ">="
	var opTo = "<"
	if len(op) > 0 {
		opFrom = op[0]
	}
	if len(op) > 1 {
		opTo = op[1]
	}
	var opFunc = func(op string, val *T) clause.Expression {
		if val == nil {
			return nil
		}
		value := mo.Some(*val)
		switch op {
		case ">=":
			return f.GteOpt(value)
		case ">":
			return f.GtOpt(value)
		case "=":
			return f.EqOpt(value)
		case "<=":
			return f.LteOpt(value)
		case "<":
			return f.LtOpt(value)
		default:
			panic(fmt.Sprintf("invalid operation %q", op))
		}
	}
	exprs := []clause.Expression{opFunc(opFrom, rng.FromValue()), opFunc(opTo, rng.ToValue())}
	return clause.And(lo.Filter(exprs, func(item clause.Expression, index int) bool {
		return item != nil
	})...)
}

func (f comparableImpl[T]) operateValue(value any, operator string) Expression {
	return f.operateValue2(f.ToColumn(), value, operator)
}

func (f comparableImpl[T]) operateValue2(column any, value any, operator string) Expression {
	var expr clause.Expression
	switch operator {
	case "=":
		expr = clause.Eq{Column: column, Value: value}
	case "!=":
		expr = clause.Neq{Column: column, Value: value}
	case ">":
		expr = clause.Gt{Column: column, Value: value}
	case ">=":
		expr = clause.Gte{Column: column, Value: value}
	case "<":
		expr = clause.Lt{Column: column, Value: value}
	case "<=":
		expr = clause.Lte{Column: column, Value: value}
	case "IN":
		expr = utils.IN{Column: column, Values: []any{value}}
	case "NOT IN":
		expr = clause.Not(utils.IN{Column: column, Values: []any{value}})
	default:
		panic(fmt.Sprintf("invalid operator %s", operator))
	}
	return expr
}

func (f comparableImpl[T]) operateField(other IField, operator string) Expression {
	return f.operateValue2(f.ToColumn(), other.ToExpr(), operator)
}
