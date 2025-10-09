package gsql

import (
	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"gorm.io/gorm/clause"
)

func LeftJoin(table ITableName) joiner {
	return joiner{joinType: "LEFT JOIN", table: table}
}

func RightJoin(table ITableName) joiner {
	return joiner{joinType: "RIGHT JOIN", table: table}
}

func InnerJoin(table ITableName) joiner {
	return joiner{joinType: "INNER JOIN", table: table}
}

type JoinClause struct {
	JoinType string
	Table    ITableName
	On       field.Expression
	hasOn    bool
}

type joiner struct {
	joinType string
	table    ITableName
}

func (j joiner) On(expr field.Expression) JoinClause {
	return JoinClause{
		JoinType: j.joinType,
		Table:    j.table,
		On:       expr,
		hasOn:    true,
	}
}

func (j joiner) OnEmpty() JoinClause {
	return JoinClause{
		JoinType: j.joinType,
		Table:    j.table,
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
		writer.AddVar(writer, v.ToExpr())
		writer.WriteByte(')')
		writer.WriteString(" AS ")
	}

	writer.WriteQuoted(tableName)
	if j.hasOn {
		writer.WriteString(" ON ")
		writer.AddVar(writer, j.On)
	}
}
