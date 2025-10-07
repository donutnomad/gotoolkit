package field

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"slices"

	"github.com/samber/mo"
)

type patternImpl[T any] struct {
	IField
}

func (f patternImpl[T]) NotLike(value T) Expression {
	return f.operateValue(value, "NOT LIKE", "", func(value string) string {
		return value
	})
}

func (f patternImpl[T]) NotLikeOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return Expression{}
	}
	return f.NotLike(value.MustGet())
}

func (f patternImpl[T]) Like(value T) Expression {
	return f.operateValue(value, "LIKE", "", func(value string) string { return value })
}

func (f patternImpl[T]) LikeOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return Expression{}
	}
	return f.Like(value.MustGet())
}

func (f patternImpl[T]) Contains(value T) Expression {
	return f.operateValue(value, "LIKE", "", func(value string) string { return "%" + value + "%" })
}

func (f patternImpl[T]) ContainsOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return Expression{}
	}
	return f.Contains(value.MustGet())
}

func (f patternImpl[T]) HasPrefix(value T) Expression {
	return f.operateValue(value, "LIKE", "", func(value string) string { return value + "%" })
}

func (f patternImpl[T]) HasPrefixOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return Expression{}
	}
	return f.HasPrefix(value.MustGet())
}

func (f patternImpl[T]) HasSuffix(value T) Expression {
	return f.operateValue(value, "LIKE", "", func(value string) string { return "%" + value })
}

func (f patternImpl[T]) HasSuffixOpt(value mo.Option[T]) Expression {
	if value.IsAbsent() {
		return Expression{}
	}
	return f.HasSuffix(value.MustGet())
}

func (f patternImpl[T]) operateValue(value any, operator string, escape string, valueFormatter func(value string) string) Expression {
	var valueString string
	for {
		switch v := value.(type) {
		case string:
			valueString = v
			break
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
	query, args := f.Column().Unpack()
	queryStr := fmt.Sprintf("%s %s ?", query, operator)
	if escape != "" {
		queryStr = fmt.Sprintf("%s ESCAPE '%s'", queryStr, escape)
	}
	return Expression{Query: queryStr, Args: slices.Concat(args, []any{valueFormatter(valueString)})}
}
