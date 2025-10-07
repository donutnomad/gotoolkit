package gsql

import (
	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"gorm.io/gorm/clause"
)

type JoinClause struct {
	JoinType string
	Table    ITableName
	On       field.Expression
}

type joiner struct {
	joinType string
	table    ITableName
}

func LeftJoin(table ITableName) joiner {
	return joiner{joinType: "LEFT JOIN", table: table}
}

func RightJoin(table ITableName) joiner {
	return joiner{joinType: "RIGHT JOIN", table: table}
}

func InnerJoin(table ITableName) joiner {
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

func (j JoinClause) Build(builder clause.Builder) {
	writer := &safeWriter{builder}

	writer.WriteString(j.JoinType)
	writer.WriteByte(' ')

	var tableName = j.Table.TableName()
	if v, ok := j.Table.(ICompactFrom); ok {
		writer.WriteByte('(')
		writer.AddVar(writer, v.Expr())
		writer.WriteByte(')')
		writer.WriteString(" AS ")
	}

	writer.WriteQuoted(tableName)
	writer.WriteString(" ON ")
	writer.AddVar(writer, j.On)
}

//func (j JoinClause) Build(builder clause.Builder) {
//	var table = j.Table.TableName()
//	var expr = clause.Expr{
//		SQL: j.JoinType,
//	}
//	if v, ok := j.Table.(ICompactFrom); ok {
//		expr.SQL += " (?) AS `" + v.TableName()
//		expr.Vars = []any{v.Expr()}
//	} else {
//		expr.SQL += " `" + table
//	}
//	expr.SQL += "` ON ?"
//	expr.Vars = append(expr.Vars, j.On)
//	expr.Build(builder)
//}
