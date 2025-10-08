package gsql

import (
	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"gorm.io/gorm/clause"
)

func Exists(builder *QueryBuilder) field.Expression {
	return existsClause{
		exists: true,
		expr:   builder.ToExpr(),
	}
}

func NotExists(builder *QueryBuilder) field.Expression {
	return existsClause{
		exists: false,
		expr:   builder.ToExpr(),
	}
}

type existsClause struct {
	expr   field.Expression
	exists bool
}

func (e existsClause) Build(builder clause.Builder) {
	if e.exists {
		builder.WriteString(" EXISTS ")
	} else {
		builder.WriteString(" NOT EXISTS ")
	}
	builder.WriteByte('(')
	e.expr.Build(builder)
	builder.WriteByte(')')
}
