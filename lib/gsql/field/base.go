package field

import (
	"gorm.io/gorm/clause"
)

type Base struct {
	tableName  string
	columnName string
	alias      string // 别名
	sql        Expression
}

func NewBase(tableName, name string) *Base {
	return &Base{
		tableName:  tableName,
		columnName: name,
	}
}

func NewBaseFromSql(expr Expression, name string) *Base {
	return &Base{
		sql:   expr,
		alias: name,
	}
}

// IsExpr 是否是一个表达式字段
func (f Base) IsExpr() bool {
	return f.sql != nil
}

// ToColumn 转换为clause.Column对象，只有非expr模式才支持导出
func (f Base) ToColumn() clause.Column {
	if f.sql != nil {
		panic("expr field cannot to column")
	}
	return NewColumnClause(f).Column
}

// ToExpr 转换为表达式
func (f Base) ToExpr() Expression {
	return NewColumnClause(f)
}

// Name 返回字段名称
// 对于expr，返回别名
// 对于普通字段，有别名的返回别名，否则返回真实名字
func (f Base) Name() string {
	if f.sql != nil {
		return f.alias
	}
	if len(f.alias) > 0 {
		return f.alias
	}
	return f.columnName
}

// As 创建一个别名字段
func (f Base) As(alias string) IField {
	if f.sql != nil {
		return NewBaseFromSql(f.sql, alias)
	}
	b := NewBase(f.tableName, f.columnName)
	b.alias = alias
	return b
}
