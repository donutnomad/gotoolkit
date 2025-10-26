package gsql

import (
	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"github.com/samber/lo"
	"gorm.io/datatypes"
	"gorm.io/gorm/clause"
)

func JsonTable(field field.IField, path string) *jsonTableBuilder {
	return &jsonTableBuilder{
		field:   field,
		path:    path,
		columns: nil,
	}
}

type jsonTableColumn struct {
	name      string // name
	fieldType string // VARCHAR(255)
	path      string // '$.name'
	onEmpty   *string
	onErr     *string
}

// Build
// symbol VARCHAR(255) PATH '$.token_symbol'
func (e jsonTableColumn) Build(builder clause.Builder) {
	builder.WriteString(e.name)
	builder.WriteString(" ")
	builder.WriteString(e.fieldType)
	builder.WriteString(" PATH '")
	builder.WriteString(e.path)
	builder.WriteString("'")

	var write = func(s string) {
		builder.WriteString(" ")
		if s == "ERROR" {
			builder.WriteString("ERROR")
		} else if s == "NULL" {
			builder.WriteString("NULL")
		} else {
			builder.WriteString("DEFAULT ")
			builder.AddVar(builder, s)
		}
	}

	if e.onEmpty != nil {
		write(*e.onEmpty)
		builder.WriteString(" ON EMPTY")
	}
	if e.onErr != nil {
		write(*e.onErr)
		builder.WriteString(" ON ERROR")
	}
}

type jsonTableBuilder struct {
	field   field.IField
	path    string
	columns []jsonTableColumn
}

// AddColumn
// https://dev.mysql.com/doc/refman/8.4/en/json-table-functions.html
// onEmptyOnError 支持填入常量 NULL(默认)/ERROR/默认值
func (b *jsonTableBuilder) AddColumn(name, fieldType, path string, onEmptyOnError ...string) *jsonTableBuilder {
	c := jsonTableColumn{
		name:      name,
		fieldType: fieldType,
		path:      path,
	}
	if len(onEmptyOnError) >= 1 {
		c.onEmpty = lo.ToPtr(onEmptyOnError[0])
	}
	if len(onEmptyOnError) >= 2 {
		c.onErr = lo.ToPtr(onEmptyOnError[1])
	}
	b.columns = append(b.columns, c)
	return b
}

func (b *jsonTableBuilder) As(tableName string) ICompactFrom {
	return &jsonTableClause{
		jsonTableBuilder: *b,
		tableName:        tableName,
	}
}

type jsonTableClause struct {
	jsonTableBuilder
	tableName string
}

func (e jsonTableClause) ToExpr() clause.Expression {
	return e
}

func (e jsonTableClause) TableName() string {
	return e.tableName
}

func (e jsonTableClause) NeedBrackets() bool {
	return false
}

// Build
// Joins("JOIN JSON_TABLE(alt.exchange_rules, '$[*]' COLUMNS(symbol VARCHAR(255) PATH '$.token_symbol')) AS t").
func (e jsonTableClause) Build(builder clause.Builder) {
	builder.WriteString("JSON_TABLE(")
	e.field.ToExpr().Build(builder)
	builder.WriteString(", '")
	builder.WriteString(e.path)
	builder.WriteString("'")
	builder.WriteString(" COLUMNS(")
	for idx, c := range e.columns {
		if idx > 0 {
			builder.WriteString(", ")
		}
		c.Build(builder)
	}
	builder.WriteString(")")
	builder.WriteString(")")
}

var _ clause.Expression = (*JSONSetExpression)(nil)

// JSONSetExpression json set expression, implements clause.Expression interface to use as updater
type JSONSetExpression struct {
	datatypes.JSONSetExpression
	length int
	column string
}

// JSONSet update fields of json column
func JSONSet(column string) *JSONSetExpression {
	return &JSONSetExpression{
		*datatypes.JSONSet(column),
		0,
		column,
	}
}

// Set return clause.Expression.
//
//	{
//		"age": 20,
//		"name": "json-1",
//		"orgs": {"orga": "orgv"},
//		"tags": ["tag1", "tag2"]
//	}
//
//	// In MySQL/SQLite, path is `age`, `name`, `orgs.orga`, `tags[0]`, `tags[1]`.
//	DB.UpdateColumn("attr", JSONSet("attr").Set("orgs.orga", 42))
//
//	// In PostgreSQL, path is `{age}`, `{name}`, `{orgs,orga}`, `{tags, 0}`, `{tags, 1}`.
//	DB.UpdateColumn("attr", JSONSet("attr").Set("{orgs, orga}", "bar"))
func (jsonSet *JSONSetExpression) Set(path string, value interface{}) *JSONSetExpression {
	jsonSet.JSONSetExpression.Set(path, value)
	jsonSet.length++
	return jsonSet
}

func (jsonSet *JSONSetExpression) Column() string {
	return jsonSet.column
}

func (jsonSet *JSONSetExpression) Len() int {
	return jsonSet.length
}
