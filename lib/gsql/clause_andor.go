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
	if len(exprs) == 1 {
		if _, ok := exprs[0].(clause.OrConditions); !ok {
			return exprs[0]
		}
	}

	var and clause.AndConditions
	for _, expr := range exprs {
		if v, ok := expr.(clause.AndConditions); ok {
			and.Exprs = append(and.Exprs, v.Exprs...)
		} else {
			and.Exprs = append(and.Exprs, expr)
		}
	}

	return and
}

func Or(exprs ...field.Expression) field.Expression {
	exprs = filterExpr(exprs...)
	if len(exprs) == 0 {
		return empty
	}
	if len(exprs) == 0 {
		return nil
	}
	var or clause.OrConditions
	for _, expr := range exprs {
		if v, ok := expr.(clause.OrConditions); ok {
			or.Exprs = append(or.Exprs, v.Exprs...)
		} else {
			or.Exprs = append(or.Exprs, expr)
		}
	}
	return or
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
