package field

import (
	"fmt"

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

func (f comparableImpl[T]) EqF(other IComparable[T]) Expression {
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

func (f comparableImpl[T]) NotField(other IComparable[T]) Expression {
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

func (f comparableImpl[T]) GtField(other IComparable[T]) Expression {
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

func (f comparableImpl[T]) GteField(other IComparable[T]) Expression {
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

func (f comparableImpl[T]) LtField(other IComparable[T]) Expression {
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

func (f comparableImpl[T]) LteField(other IComparable[T]) Expression {
	return f.operateField(other, "<=")
}

func (f comparableImpl[T]) operateValue(value any, operator string) Expression {
	if f.IsExpr() {
		panic("[comparableImpl] cannot operate on expr")
	}

	var expr clause.Expression
	var column = f.ToColumn()
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
		expr = clause.IN{Column: column, Values: []any{value}}
	case "NOT IN":
		expr = clause.Not(clause.IN{Column: column, Values: []any{value}})
	default:
		panic(fmt.Sprintf("invalid operator %s", operator))
	}
	return expr
}

func (f comparableImpl[T]) operateField(other IComparable[T], operator string) Expression {
	if other.IsExpr() {
		panic("[comparableImpl] cannot operate on expr")
	}
	return f.operateValue(other.ToColumn(), operator)
}
