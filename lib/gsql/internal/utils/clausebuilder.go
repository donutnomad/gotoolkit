package utils

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"
	"time"

	mysql2 "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

var Dialector = &mysql.Dialector{
	Config: &mysql.Config{
		DSNConfig: &mysql2.Config{
			Loc: time.UTC,
		},
	},
}

type MemoryBuilder struct {
	SQL  strings.Builder
	Vars []any

	error     error
	dialector gorm.Dialector // *mysql.Dialector
	TableExpr *clause.Expr
	Table     string
	Schema    *schema.Schema
}

func NewMemoryBuilder() *MemoryBuilder {
	return &MemoryBuilder{
		SQL:       strings.Builder{},
		dialector: Dialector,
	}
}

func (m *MemoryBuilder) WriteByte(b byte) error {
	return m.SQL.WriteByte(b)
}

func (m *MemoryBuilder) WriteString(s string) (int, error) {
	return m.SQL.WriteString(s)
}

// Quote returns quoted value
func (m *MemoryBuilder) Quote(field any) string {
	var builder strings.Builder
	m.QuoteTo(&builder, field)
	return builder.String()
}

func (m *MemoryBuilder) WriteQuoted(value any) {
	m.QuoteTo(&m.SQL, value)
}

// AddVar add var
func (m *MemoryBuilder) AddVar(writer clause.Writer, vars ...any) {
	writeString := func(str string) {
		_, _ = writer.WriteString(str)
	}
	writeByte := func(b byte) {
		_ = writer.WriteByte(b)
	}

	for idx, v := range vars {
		if idx > 0 {
			writeByte(',')
		}

		switch v := v.(type) {
		case sql.NamedArg:
			m.Vars = append(m.Vars, v.Value)
		case clause.Column, clause.Table:
			m.QuoteTo(writer, v)
		case gorm.Valuer:
			reflectValue := reflect.ValueOf(v)
			if reflectValue.Kind() == reflect.Ptr && reflectValue.IsNil() {
				m.AddVar(writer, nil)
			} else {
				m.AddVar(writer, v.GormValue(context.Background(), &gorm.DB{
					Config: &gorm.Config{
						Dialector: Dialector,
					},
				}))
			}
		case clause.Interface:
			c := clause.Clause{Name: v.Name()}
			v.MergeClause(&c)
			c.Build(m)
		case clause.Expression:
			v.Build(m)
		case driver.Valuer:
			m.Vars = append(m.Vars, v)
			m.bindVarTo(writer, v)
		case []byte:
			m.Vars = append(m.Vars, v)
			m.bindVarTo(writer, v)
		case []any:
			if len(v) > 0 {
				writeByte('(')
				m.AddVar(writer, v...)
				writeByte(')')
			} else {
				writeString("(NULL)")
			}
		default:
			switch rv := reflect.ValueOf(v); rv.Kind() {
			case reflect.Slice, reflect.Array:
				if rv.Len() == 0 {
					writeString("(NULL)")
				} else if rv.Type().Elem() == reflect.TypeOf(uint8(0)) {
					m.Vars = append(m.Vars, v)
					m.bindVarTo(writer, v)
				} else {
					writeByte('(')
					for i := 0; i < rv.Len(); i++ {
						if i > 0 {
							writeByte(',')
						}
						m.AddVar(writer, rv.Index(i).Interface())
					}
					writeByte(')')
				}
			default:
				m.Vars = append(m.Vars, v)
				m.bindVarTo(writer, v)
			}
		}
	}
}

func (m *MemoryBuilder) AddError(err error) error {
	if err != nil {
		if m.error == nil {
			m.error = err
		} else {
			m.error = fmt.Errorf("%v; %w", m.error, err)
		}
	}
	return m.error
}

// QuoteTo write quoted value to writer
func (m *MemoryBuilder) QuoteTo(writer clause.Writer, field any) {
	writeString := func(str string) {
		_, _ = writer.WriteString(str)
	}
	writeByte := func(b byte) {
		_ = writer.WriteByte(b)
	}
	write := func(raw bool, str string) {
		if raw {
			writeString(str)
		} else {
			m.quoteTo(writer, str)
		}
	}

	switch v := field.(type) {
	case clause.Table:
		if v.Name == clause.CurrentTable {
			if m.TableExpr != nil {
				m.TableExpr.Build(m)
			} else {
				write(v.Raw, m.Table)
			}
		} else {
			write(v.Raw, v.Name)
		}

		if v.Alias != "" {
			writeByte(' ')
			write(v.Raw, v.Alias)
		}
	case clause.Column:
		if v.Table != "" {
			if v.Table == clause.CurrentTable {
				write(v.Raw, m.Table)
			} else {
				write(v.Raw, v.Table)
			}
			writeByte('.')
		}

		if v.Name == clause.PrimaryKey {
			if m.Schema == nil {
				_ = m.AddError(gorm.ErrModelValueRequired)
			} else if m.Schema.PrioritizedPrimaryField != nil {
				write(v.Raw, m.Schema.PrioritizedPrimaryField.DBName)
			} else if len(m.Schema.DBNames) > 0 {
				write(v.Raw, m.Schema.DBNames[0])
			} else {
				_ = m.AddError(gorm.ErrModelAccessibleFieldsRequired) //nolint:typecheck,errcheck
			}
		} else {
			write(v.Raw, v.Name)
		}

		if v.Alias != "" {
			writeString(" AS ")
			write(v.Raw, v.Alias)
		}
	case []clause.Column:
		writeByte('(')
		for idx, d := range v {
			if idx > 0 {
				writeByte(',')
			}
			m.QuoteTo(writer, d)
		}
		writeByte(')')
	case clause.Expr:
		v.Build(m)
	case string:
		m.quoteTo(writer, v)
	case []string:
		writeByte('(')
		for idx, d := range v {
			if idx > 0 {
				writeByte(',')
			}
			m.quoteTo(writer, d)
		}
		writeByte(')')
	default:
		m.quoteTo(writer, fmt.Sprint(field))
	}
}

func (m *MemoryBuilder) quoteTo(writer clause.Writer, str string) {
	m.dialector.QuoteTo(writer, str)
}

func (m *MemoryBuilder) bindVarTo(writer clause.Writer, value any) {
	m.dialector.BindVarTo(writer, &gorm.Statement{
		Vars: m.Vars,
	}, value)
}
