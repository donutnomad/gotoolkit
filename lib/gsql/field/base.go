package field

import (
	"fmt"
	"strings"
)

type Base struct {
	tableName  string
	columnName string
	sql        *Expression
}

func NewBase(tableName, name string) *Base {
	return &Base{
		tableName:  tableName,
		columnName: name,
	}
}

func NewBaseFromSql(expr Expression) *Base {
	return &Base{
		sql: &expr,
	}
}

func (f Base) Name() string {
	column := ExtractColumn(&f)
	if idx := strings.Index(column, "."); idx >= 0 {
		return column[idx+1:]
	}
	return column
}

func (f Base) Column() Expression {
	if f.sql == nil {
		var query string
		if f.tableName == "" {
			query = WrapIfReserved(f.columnName)
		} else {
			query = fmt.Sprintf("%s.%s", WrapIfReserved(f.tableName), WrapIfReserved(f.columnName))
		}
		return Expression{Query: query}
	} else {
		return *f.sql
	}
}

func (f Base) As(alias string) IField {
	if f.sql == nil {
		return NewBaseFromSql(Expression{
			Query: AS(f.Column().Query, alias),
		})
	}
	expr := f.sql.ToExpr()
	return NewBaseFromSql(Expression{
		Query: AS(expr.SQL, alias),
		Args:  expr.Vars,
	})
}
