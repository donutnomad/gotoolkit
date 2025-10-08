package gsql

import (
	"slices"

	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"gorm.io/gorm/clause"
)

type QueryBuilder QueryBuilderG[any]

func Select(fields ...field.IField) *baseQueryBuilder {
	return baseQueryBuilder{}.Select(fields...)
}

func Pluck(f field.IField) *baseQueryBuilder {
	return Select(f)
}

type baseQueryBuilder struct {
	selects []field.IField
}

func (baseQueryBuilder) Select(fields ...field.IField) *baseQueryBuilder {
	var b = &baseQueryBuilder{}
	for _, f := range fields {
		b.selects = append(b.selects, f)
	}
	return b
}

func (b baseQueryBuilder) From(table interface{ TableName() string }) *QueryBuilder {
	return &QueryBuilder{
		selects: b.selects,
		from:    table,
	}
}

func (b *QueryBuilder) as() *QueryBuilderG[any] {
	return (*QueryBuilderG[any])(b)
}

func (b *QueryBuilder) Join(clauses ...JoinClause) *QueryBuilder {
	b.as().Join(clauses...)
	return b
}

func (b *QueryBuilder) Where(exprs ...field.Expression) *QueryBuilder {
	b.as().Where(exprs...)
	return b
}

func (b *QueryBuilder) ToSQL() string {
	return b.as().ToSQL()
}

func (b *QueryBuilder) String() string {
	return b.ToSQL()
}

func (b *QueryBuilder) ToExpr() clause.Expression {
	return b.as().ToExpr()
}

func (b *QueryBuilder) Clone() *QueryBuilder {
	return &QueryBuilder{
		selects: slices.Clone(b.selects),
		from:    b.from,
		joins:   slices.Clone(b.joins),
		wheres:  slices.Clone(b.wheres),
	}
}

func (b *QueryBuilder) build(db IDB) IDB {
	return b.as().build(db)
}

func (b *QueryBuilder) Order(column field.IField, asc ...bool) *QueryBuilder {
	b.as().Order(column, asc...)
	return b
}

type Paginate struct {
	Page     int
	PageSize int
}

func (b *QueryBuilder) Paginate(p Paginate) *QueryBuilder {
	page := max(1, p.Page)
	pageSize := max(1, p.PageSize)
	b.Offset((page - 1) * pageSize)
	b.Limit(pageSize)
	return b
}

func (b *QueryBuilder) Offset(offset int) *QueryBuilder {
	b.as().Offset(offset)
	return b
}

func (b *QueryBuilder) Limit(limit int) *QueryBuilder {
	b.as().Limit(limit)
	return b
}

type ScopeFunc func(b *QueryBuilder)

func (b *QueryBuilder) Scope(fn ScopeFunc) *QueryBuilder {
	return b.Scopes(fn)
}

func (b *QueryBuilder) Scopes(fns ...ScopeFunc) *QueryBuilder {
	for _, fn := range fns {
		fn(b)
	}
	return b
}

func (b *QueryBuilder) Unscoped() *QueryBuilder {
	b.unscoped = true
	return b
}

func (b *QueryBuilder) Distinct() *QueryBuilder {
	b.distinct = true
	return b
}

func (b *QueryBuilder) Create(db IDB, value any) DBResult {
	builder := b.Clone()
	builder.selects = nil
	builder.wheres = nil
	builder.from = TableName2("")
	ret := builder.build(db).Create(value)
	return DBResult{
		ret.Error,
		ret.RowsAffected,
	}
}

func (b *QueryBuilder) Update(db IDB, value any) DBResult {
	ret := b.build(db).Updates(value)
	return DBResult{
		ret.Error,
		ret.RowsAffected,
	}
}

func (b *QueryBuilder) Delete(db IDB, dest any) DBResult {
	ret := b.build(db).Delete(&dest)
	return DBResult{
		ret.Error,
		ret.RowsAffected,
	}
}

func (b *QueryBuilder) Count(db IDB) (count int64, _ error) {
	ret := b.build(db).Count(&count)
	return count, ret.Error
}

func (b *QueryBuilder) Exist(db IDB) (bool, error) {
	var count int64
	tx := b.Clone().Limit(1).build(db).Count(&count)
	return count > 0, tx.Error
}

func (b *QueryBuilder) Take(db IDB, dest any) error {
	return firstLast(b.as(), db, false, false, dest)
}

func (b *QueryBuilder) First(db IDB, dest any) error {
	return firstLast(b.as(), db, true, false, dest)
}

func (b *QueryBuilder) Last(db IDB, dest any) error {
	return firstLast(b.as(), db, true, true, dest)
}

func (b *QueryBuilder) Find(db IDB, dest any) error {
	ret := b.build(db).Find(dest)
	if ret.RowsAffected == 0 {
		return nil
	} else if ret.Error != nil {
		return ret.Error
	}
	return ret.Error
}

// AsF as field
func (b *QueryBuilder) AsF(asName ...string) field.IField {
	if len(b.selects) == 0 {
		panic("selects is empty")
	} else {
		b.selects = b.selects[0:1]
	}
	return FieldExpr(b.ToExpr(), optional(asName, ""))
}
