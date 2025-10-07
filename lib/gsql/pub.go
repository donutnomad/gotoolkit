package gsql

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"regexp"
	"strings"

	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"gorm.io/gorm/clause"
)

type Table struct {
	Name string
}

type Table2 struct {
	Name string
}

func (t Table2) TableName() string {
	return t.Name
}

func (t Table) Ptr() *Table {
	return &t
}

func (t *Table) TableName() string {
	return t.Name
}

func TableName(name string) Table {
	return Table{Name: name}
}

func TableName2(name string) Table2 {
	return Table2{Name: name}
}

func tableNameFn(name string) tableNameImpl {
	return tableNameImpl{name: name}
}

type tableNameImpl struct {
	name string
}

func (t tableNameImpl) TableName() string {
	return t.name
}

type templateTable[T any, Model any] struct {
	Fields    T
	tableName string
	query     QueryBuilder
}

func (t templateTable[T, Model]) ModelType() *Model {
	var def Model
	return &def
}

func (t templateTable[T, Model]) TableName() string {
	return t.tableName
}

func (t templateTable[T, Model]) Query() QueryBuilder {
	return t.query
}

type ICompactFrom interface {
	Query() QueryBuilder
	TableName() string
}

func DefineTempTable[Model any, ModelT any](types ModelT, builder *QueryBuilder) templateTable[ModelT, Model] {
	return DefineTable[Model, ModelT](fmt.Sprintf("%s%d", "temp_", rand.N(32)), types, builder)
}

func DefineTempTableAny[T any](types T, builder *QueryBuilder) templateTable[T, any] {
	return DefineTable[any, T](fmt.Sprintf("%s%d", "temp_", rand.N(32)), types, builder)
}

func DefineTable[Model any, T any](tableName string, types T, builder *QueryBuilder) templateTable[T, Model] {
	b := builder.Clone()

	if len(b.selects) == 0 {
		b.selects = append(b.selects, Star.Column())
	}

	var newTable = reflect.ValueOf(tableNameFn(tableName))
	var ty = &types

	rv := reflect.ValueOf(ty)
	if rv.Kind() != reflect.Ptr {
		panic("input must be a pointer")
	}
	if rv.IsNil() {
		panic("input pointer is nil")
	}

	// 解引用获取指针指向的结构体
	rv = rv.Elem()

	if rv.Kind() != reflect.Struct {
		panic("input must be pointer to struct")
	}

	for i := 0; i < rv.NumField(); i++ {
		fieldValue := rv.Field(i)
		fieldType := rv.Type().Field(i)

		if !fieldType.IsExported() || !fieldValue.CanSet() {
			continue
		}

		fieldV2 := fieldValue.Addr().Interface()
		if v, ok := fieldV2.(interface{ WithTable(tableName string) }); ok {
			v.WithTable(tableName)
		} else {
			withTableMethod := fieldValue.MethodByName("WithTable")
			if !withTableMethod.IsValid() {
				continue
			}
			results := withTableMethod.Call([]reflect.Value{newTable})
			if len(results) > 0 {
				fieldValue.Set(results[0])
			}
		}
	}

	return templateTable[T, Model]{
		Fields:    *ty,
		tableName: tableName,
		query:     *b,
	}
}

func And(exprs ...field.Expression) field.Expression {
	return buildAndOr(exprs, "AND")
}

func Or(exprs ...field.Expression) field.Expression {
	return buildAndOr(exprs, "OR")
}

func buildAndOr(exprs []field.Expression, operator string) field.Expression {
	var query strings.Builder
	var args []any
	query.WriteString("(")
	for i, expr := range exprs {
		if i > 0 {
			query.WriteString(" ")
			query.WriteString(operator)
			query.WriteString(" ")
		}
		query.WriteString(expr.Query)
		args = append(args, expr.Args...)
	}
	query.WriteString(")")
	return field.Expression{
		Query: query.String(),
		Args:  args,
	}
}

var (
	createClauses = []string{"INSERT", "VALUES", "ON CONFLICT"}
	queryClauses  = []string{"SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "LIMIT", "FOR"}
	updateClauses = []string{"UPDATE", "SET", "WHERE"}
	deleteClauses = []string{"DELETE", "FROM", "WHERE"}
)

var tableRegexp = regexp.MustCompile(`(?i)(?:.+? AS (\w+)\s*(?:$|,)|^\w+\s+(\w+)$)`)

func txTable(quote func(field string) string, name string, args ...any) (expr *clause.Expr, table string) {
	if strings.Contains(name, " ") || strings.Contains(name, "`") || len(args) > 0 {
		expr = &clause.Expr{SQL: name, Vars: args}
		if results := tableRegexp.FindStringSubmatch(name); len(results) == 3 {
			if results[1] != "" {
				table = results[1]
			} else {
				table = results[2]
			}
		}
	} else if tables := strings.Split(name, "."); len(tables) == 2 {
		expr = &clause.Expr{SQL: quote(name)}
		table = tables[1]
	} else if name != "" {
		expr = &clause.Expr{SQL: quote(name)}
		table = name
	}
	return
}

type order struct {
	field field.IField
	asc   bool
}
