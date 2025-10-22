package gsql

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"regexp"
	"strings"

	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TableName(name string) Table {
	return Table{Name: name}
}

func TableName2(name string) Table2 {
	return Table2{Name: name}
}

func MapPattern[OUT any, IN any](input field.Pattern[IN]) field.Pattern[OUT] {
	return field.NewPatternWith[OUT](input.Base)
}

func MapComparable[OUT any, IN any](input field.Comparable[IN]) field.Comparable[OUT] {
	return field.NewComparableWith[OUT](input.Base)
}

func Field(sql string, args ...any) field.IField {
	return field.NewBaseFromSql(Expr(sql, args...), "")
}

func FieldExpr(expr field.Expression, alias string) field.IField {
	return field.NewBaseFromSql(expr, alias)
}

func Expr(sql string, args ...any) field.Expression {
	return clause.Expr{
		SQL:  sql,
		Vars: args,
	}
}

func DefineTempTable[Model any, ModelT any](types ModelT, builder *QueryBuilder) templateTable[ModelT, Model] {
	return DefineTable[Model, ModelT](fmt.Sprintf("%s%d", "temp_", rand.N(32)), types, builder)
}

func DefineTempTableAny[ModelT any](types ModelT, builder *QueryBuilder) templateTable[ModelT, any] {
	return DefineTable[any, ModelT](fmt.Sprintf("%s%d", "temp_", rand.N(32)), types, builder)
}

func DefineTable[Model any, ModelT any](tableName string, types ModelT, builder field.IToExpr) templateTable[ModelT, Model] {
	if v, ok := builder.(*QueryBuilder); ok {
		if len(v.selects) == 0 {
			b := v.Clone()
			b.selects = append(b.selects, Star)
			builder = b
		}
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

	return templateTable[ModelT, Model]{
		Fields:    *ty,
		tableName: tableName,
		expr:      builder.ToExpr(),
	}
}

var (
	createClauses = []string{"INSERT", "VALUES", "ON CONFLICT"}
	queryClauses  = []string{"CTE", "SELECT", "FROM", "WHERE", "GROUP BY", "ORDER BY", "LIMIT", "FOR"}
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
		if name != "DUAL" {
			name = quote(name)
		}
		expr = &clause.Expr{SQL: name}
		table = tables[1]
	} else if name != "" {
		if name != "DUAL" {
			name = quote(name)
		}
		expr = &clause.Expr{SQL: name}
		table = name
	}
	return
}

func addSelects(stmt *gorm.Statement, selects []field.IField) {
	if len(selects) == 0 {
		return
	}
	var m = make(map[string]struct{}, len(selects))
	for _, s := range selects {
		name := s.Name()
		if len(name) == 0 {
			continue
		}
		_, ok := m[name]
		if ok {
			panic(fmt.Sprintf("conflict select field name: `%s`, check your select fields", name))
		}
		m[name] = struct{}{}
	}

	stmt.AddClause(clause.Select{
		Distinct: stmt.Distinct,
		Expression: columnClause{
			commaExpr: clause.CommaExpression{
				Exprs: lo.Map(selects, func(item field.IField, index int) clause.Expression {
					return item.ToExpr()
				}),
			},
			distinct: stmt.Distinct,
		},
	})
}

type columnClause struct {
	commaExpr clause.CommaExpression
	distinct  bool
}

func (c columnClause) Build(builder clause.Builder) {
	if c.distinct {
		_, _ = builder.WriteString("DISTINCT ")
	}
	c.commaExpr.Build(builder)
}

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

type order struct {
	field field.IField
	asc   bool
}

type safeWriter struct {
	builder clause.Builder
}

func (w *safeWriter) WriteByte(b byte) {
	_ = w.builder.WriteByte(b)
}
func (w *safeWriter) WriteString(b string) {
	_, _ = w.builder.WriteString(b)
}
func (w *safeWriter) WriteQuoted(f any) {
	w.builder.WriteQuoted(f)
}
func (w *safeWriter) AddVar(writer *safeWriter, args ...any) {
	w.builder.AddVar(writer.builder, args...)
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
	expr      clause.Expression
}

func (t templateTable[T, Model]) ModelType() *Model {
	var def Model
	return &def
}

func (t templateTable[T, Model]) TableName() string {
	return t.tableName
}

func (t templateTable[T, Model]) ToExpr() clause.Expression {
	return t.expr
}
