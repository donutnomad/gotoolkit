package field

import (
	"fmt"
	"strings"

	"gorm.io/gorm/clause"
)

// TODO: 需要支持: name, table.name, (SELECT name FROM table LIMIT 1) AS name 或者 没有As (SELECT name FROM table LIMIT 1)

type Base struct {
	tableName  string
	columnName string
	alias      string // 别名
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

func (f Base) ToColumn() clause.Column {
	if f.sql != nil {
		return clause.Column{
			Name: f.sql.Query,
			Raw:  true,
		}
	}
	return clause.Column{
		Table: f.tableName,
		Name:  f.columnName,
		Alias: f.alias,
		Raw:   false,
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
