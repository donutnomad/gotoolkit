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

type ColumnClause struct {
	clause.Column
	Expr clause.Expression
}

func NewColumnClause(f Base) ColumnClause {
	if f.sql != nil {
		return ColumnClause{
			Column: clause.Column{
				Alias: f.alias,
				Raw:   true,
			},
			Expr: f.sql,
		}
	}
	return ColumnClause{
		Column: clause.Column{
			Table: f.tableName,
			Name:  f.columnName,
			Alias: f.alias,
			Raw:   false,
		},
	}
}

func (v ColumnClause) AsColumn() clause.Column {
	return v.Column
}

func (v ColumnClause) Build(builder clause.Builder) {
	writer := builder
	write := func(raw bool, str string) {
		if raw {
			writer.WriteString(str)
		} else {
			writer.WriteQuoted(str)
		}
	}

	if v.Expr != nil {
		writer.WriteByte('(')
		v.Expr.Build(builder)
		writer.WriteByte(')')
		if v.Alias != "" {
			writer.WriteString(" AS ")
			write(v.Raw, v.Alias)
		}
	} else {
		writer.WriteQuoted(v.Column)
	}
}

//func QuoteTo(writer clause.Writer, dialector gorm.Dialector, field interface{}) {
//	write := func(raw bool, str string) {
//		if raw {
//			writer.WriteString(str)
//		} else {
//			dialector.QuoteTo(writer, str)
//		}
//	}
//
//	switch v := field.(type) {
//	case clause.Column:
//		if v.Table != "" {
//			if v.Table == clause.CurrentTable {
//				write(v.Raw, stmt.Table)
//			} else {
//				write(v.Raw, v.Table)
//			}
//			writer.WriteByte('.')
//		}
//
//		if v.Name == clause.PrimaryKey {
//			if stmt.Schema == nil {
//				stmt.DB.AddError(ErrModelValueRequired)
//			} else if stmt.Schema.PrioritizedPrimaryField != nil {
//				write(v.Raw, stmt.Schema.PrioritizedPrimaryField.DBName)
//			} else if len(stmt.Schema.DBNames) > 0 {
//				write(v.Raw, stmt.Schema.DBNames[0])
//			} else {
//				stmt.DB.AddError(ErrModelAccessibleFieldsRequired) //nolint:typecheck,errcheck
//			}
//		} else {
//			write(v.Raw, v.Name)
//		}
//
//		if v.Alias != "" {
//			writer.WriteString(" AS ")
//			write(v.Raw, v.Alias)
//		}
//	case []clause.Column:
//		writer.WriteByte('(')
//		for idx, d := range v {
//			if idx > 0 {
//				writer.WriteByte(',')
//			}
//			stmt.QuoteTo(writer, d)
//		}
//		writer.WriteByte(')')
//	case clause.Expr:
//		v.Build(stmt)
//	case string:
//		dialector.QuoteTo(writer, v)
//	case []string:
//		writer.WriteByte('(')
//		for idx, d := range v {
//			if idx > 0 {
//				writer.WriteByte(',')
//			}
//			dialector.QuoteTo(writer, d)
//		}
//		writer.WriteByte(')')
//	default:
//		dialector.QuoteTo(writer, fmt.Sprint(field))
//	}
//}
