package field

import (
	"gorm.io/gorm/clause"
)

// FieldFlag 字段标志位
type FieldFlag uint32

const (
	FlagNone          FieldFlag = 0
	FlagPrimaryKey    FieldFlag = 1 << 0 // 主键
	FlagUniqueIndex   FieldFlag = 1 << 1 // 唯一索引
	FlagIndex         FieldFlag = 1 << 2 // 普通索引
	FlagAutoIncrement FieldFlag = 1 << 3 // 自增
)

type Base struct {
	tableName  string
	columnName string
	alias      string // 别名
	sql        Expression
	flags      FieldFlag // 字段标志
}

func NewBase(tableName, name string, flags ...FieldFlag) *Base {
	var flag FieldFlag = FlagNone
	if len(flags) > 0 {
		flag = flags[0]
	}
	return &Base{
		tableName:  tableName,
		columnName: name,
		flags:      flag,
	}
}

// Flags 返回字段标志
func (f Base) Flags() FieldFlag {
	return f.flags
}

// HasFlag 判断是否有某个标志
func (f Base) HasFlag(flag FieldFlag) bool {
	return f.flags&flag != 0
}

// IsPrimaryKey 是否为主键
func (f Base) IsPrimaryKey() bool {
	return f.HasFlag(FlagPrimaryKey)
}

// IsUniqueIndex 是否为唯一索引
func (f Base) IsUniqueIndex() bool {
	return f.HasFlag(FlagUniqueIndex)
}

func NewBaseFromSql(expr Expression, alias string) *Base {
	return &Base{
		sql:   expr,
		alias: alias,
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

func (f Base) FullName() string {
	return fieldName(f.tableName, f.columnName)
}

func (f Base) Alias() string {
	return f.alias
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
