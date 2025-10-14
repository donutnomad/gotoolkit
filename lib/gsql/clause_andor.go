package gsql

import (
	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"gorm.io/gorm/clause"
)

var empty = clause.Expr{}

func And(exprs ...field.Expression) field.Expression {
	exprs = filterExpr(exprs...)
	if len(exprs) == 0 {
		return empty
	}
	return clause.And(exprs...)
}

func Or(exprs ...field.Expression) field.Expression {
	exprs = filterExpr(exprs...)
	if len(exprs) == 0 {
		return empty
	}
	return clause.Or(exprs...)
}

func filterExpr(input ...field.Expression) []field.Expression {
	var output = make([]field.Expression, 0, len(input))
	for _, expr := range input {
		if v, ok := expr.(clause.Expr); ok {
			if len(v.SQL) == 0 {
				continue
			}
		}
		output = append(output, expr)
	}
	return output
}
