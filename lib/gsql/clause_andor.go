package gsql

import (
	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"gorm.io/gorm/clause"
)

func And(exprs ...field.Expression) field.Expression {
	return clause.And(exprs...)
}

func Or(exprs ...field.Expression) field.Expression {
	return clause.Or(exprs...)
}
