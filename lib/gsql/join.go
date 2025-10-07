package gsql

import (
	"slices"

	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"gorm.io/gorm/clause"
)

type JoinClause struct {
	JoinType string
	Table    interface{ TableName() string }
	On       field.Expression
}
type joiner struct {
	joinType string
	table    interface{ TableName() string }
}

func LeftJoin(table interface{ TableName() string }) joiner {
	return joiner{joinType: "LEFT JOIN", table: table}
}

func RightJoin(table interface{ TableName() string }) joiner {
	return joiner{joinType: "RIGHT JOIN", table: table}
}

func InnerJoin(table interface{ TableName() string }) joiner {
	return joiner{joinType: "INNER JOIN", table: table}
}

func (j joiner) On(expr field.Expression) JoinClause {
	return JoinClause{
		JoinType: j.joinType,
		Table:    j.table,
		On:       expr,
	}
}

func (j JoinClause) And(expr field.Expression) JoinClause {
	return JoinClause{
		JoinType: j.JoinType,
		Table:    j.Table,
		On:       And(j.On, expr),
	}
}

func (j JoinClause) Or(expr field.Expression) JoinClause {
	return JoinClause{
		JoinType: j.JoinType,
		Table:    j.Table,
		On:       Or(j.On, expr),
	}
}

func (j JoinClause) Build() clause.Expr {
	var table = j.Table.TableName()
	var expr = clause.Expr{}
	if v, ok := j.Table.(ICompactFrom); ok {
		q := v.Query()
		expr.SQL = j.JoinType + " (?) AS `" + v.TableName() + "` ON ?"
		expr.Vars = slices.Concat(expr.Vars, []any{q.ToExpr(), j.On.ToExpr()})
	} else {
		expr.SQL = j.JoinType + " `" + table + "` ON ?"
		expr.Vars = slices.Concat(expr.Vars, []any{j.On.ToExpr()})
	}
	return expr
}
