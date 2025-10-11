package field

import (
	"database/sql/driver"
	"reflect"
	"strings"

	"github.com/samber/mo"
	"gorm.io/gorm/clause"
)

type patternImpl[T any] struct {
	IField
}

func (f patternImpl[T]) Like(value T, escape ...byte) Expression {
	return f.operateValue(value, "LIKE", optional(escape, 0), func(value string) string { return value })
}

func (f patternImpl[T]) LikeOpt(value mo.Option[T], escape ...byte) Expression {
	if value.IsAbsent() {
		return emptyExpression
	}
	return f.Like(value.MustGet(), escape...)
}

func (f patternImpl[T]) NotLike(value T, escape ...byte) Expression {
	return f.operateValue(value, "NOT LIKE", optional(escape, 0), func(value string) string {
		return value
	})
}

func (f patternImpl[T]) NotLikeOpt(value mo.Option[T], escape ...byte) Expression {
	if value.IsAbsent() {
		return emptyExpression
	}
	return f.NotLike(value.MustGet(), escape...)
}

func (f patternImpl[T]) Contains(value T) Expression {
	return f.operateValue(value, "LIKE", 0, func(value string) string { return "%" + value + "%" })
}

func (f patternImpl[T]) ContainsOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return emptyExpression
	}
	return f.Contains(value.MustGet())
}

func (f patternImpl[T]) HasPrefix(value T) Expression {
	return f.operateValue(value, "LIKE", 0, func(value string) string { return value + "%" })
}

func (f patternImpl[T]) HasPrefixOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return emptyExpression
	}
	return f.HasPrefix(value.MustGet())
}

func (f patternImpl[T]) HasSuffix(value T) Expression {
	return f.operateValue(value, "LIKE", 0, func(value string) string { return "%" + value })
}

func (f patternImpl[T]) HasSuffixOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return emptyExpression
	}
	return f.HasSuffix(value.MustGet())
}

func (f patternImpl[T]) anyToString(value any) string {
	var valueString string
	for {
		switch v := value.(type) {
		case string:
			return v
		case driver.Valuer:
			v1, err := v.Value()
			if err != nil {
				panic(err)
			}
			value = v1
			continue
		default:
		}
		valueOf := reflect.ValueOf(value)
		if valueOf.Kind() == reflect.String {
			valueString = valueOf.String()
			break
		} else {
			panic("value must be string")
		}
	}
	return valueString
}

func (f patternImpl[T]) operateValue(value any, operator string, escape byte, valueFormatter func(value string) string) Expression {
	var expr clause.Expression = clause.Like{
		Column: f.ToColumn(),
		Value: escapeClause{
			value:  valueFormatter(f.anyToString(value)),
			escape: escape,
		},
	}
	if strings.HasPrefix(operator, "NOT") {
		expr = clause.Not(expr)
	}
	return expr
}
