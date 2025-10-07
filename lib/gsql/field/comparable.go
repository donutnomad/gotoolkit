package field

import (
	"fmt"
	"slices"

	"github.com/samber/mo"
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
		return Expression{}
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
		return Expression{}
	}
	return f.Not(value.MustGet())
}

func (f comparableImpl[T]) NotField(other IComparable[T]) Expression {
	return f.operateField(other, "!=")
}

func (f comparableImpl[T]) In(values ...T) Expression {
	if len(values) == 0 {
		return Expression{}
	}
	return f.operateValue(sliceToAny(values), "IN")
}

func (f comparableImpl[T]) NotIn(values ...T) Expression {
	if len(values) == 0 {
		return Expression{}
	}
	return f.operateValue(sliceToAny(values), "NOT IN")
}

func (f comparableImpl[T]) Gt(value T) Expression {
	return f.operateValue(value, ">")
}

func (f comparableImpl[T]) GtOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return Expression{}
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
		return Expression{}
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
		return Expression{}
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
		return Expression{}
	}
	return f.Lte(value.MustGet())
}

func (f comparableImpl[T]) LteField(other IComparable[T]) Expression {
	return f.operateField(other, "<=")
}

func (f comparableImpl[T]) operateValue(value any, operator string) Expression {
	_, args := f.Column().Unpack()
	column := ExtractColumn(f)
	return Expression{Query: fmt.Sprintf("%s %s ?", column, operator), Args: slices.Concat(args, []any{value})}
}

func (f comparableImpl[T]) operateField(other IComparable[T], operator string) Expression {
	_ = requireNoArgs(f.Column().Unpack())
	_ = requireNoArgs(other.Column().Unpack())
	column1 := ExtractColumn(f)
	column2 := ExtractColumn(other)
	return Expression{Query: fmt.Sprintf("%s %s %s", column1, operator, column2)}
}
