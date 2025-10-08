package gsql

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	mysql2 "github.com/go-sql-driver/mysql"
	"github.com/samber/lo"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
)

type ScopeFuncG[Model any] func(b *QueryBuilderG[Model])

type QueryBuilderG[T any] struct {
	selects  []field.IField
	from     interface{ TableName() string }
	joins    []JoinClause
	wheres   []clause.Expression
	orders   []order
	offset   int
	limit    int
	unscoped bool
	distinct bool
}

func SelectG[T any](fields ...field.IField) *baseQueryBuilderG[T] {
	return baseQueryBuilderG[T]{}.Select(fields...)
}

func PluckG[T any, Field interface {
	field.IField
	field.IFieldType[T]
}](f Field) *baseQueryBuilderG[T] {
	return SelectG[T](f)
}

func FromG[T any, Table interface {
	TableName() string
	ModelType() *T
}](from Table) *QueryBuilderG[T] {
	return baseQueryBuilderG[T]{}.Select().From(from)
}

type baseQueryBuilderG[T any] struct {
	selects []field.IField
}

func (baseQueryBuilderG[T]) Select(fields ...field.IField) *baseQueryBuilderG[T] {
	var b = &baseQueryBuilderG[T]{}
	for _, f := range fields {
		b.selects = append(b.selects, f)
	}
	return b
}

func (b baseQueryBuilderG[T]) From(table interface {
	TableName() string
}) *QueryBuilderG[T] {
	return &QueryBuilderG[T]{
		selects: b.selects,
		from:    table,
	}
}

func (b *QueryBuilderG[T]) Join(clauses ...JoinClause) *QueryBuilderG[T] {
	b.joins = append(b.joins, clauses...)
	return b
}

func (b *QueryBuilderG[T]) Where(exprs ...field.Expression) *QueryBuilderG[T] {
	for _, expr := range exprs {
		b.wheres = append(b.wheres, expr)
	}
	return b
}

func (b *QueryBuilderG[T]) ToSQL() string {
	expr := b.ToExpr()
	return dialector.Explain(expr.SQL, expr.Vars...)
}

func (b *QueryBuilderG[T]) String() string {
	return b.ToSQL()
}

func (b *QueryBuilderG[T]) ToExpr() clause.Expr {
	tx := &gorm.DB{
		Config: &gorm.Config{
			ClauseBuilders: map[string]clause.ClauseBuilder{},
			Dialector:      dialector,
		},
		Statement: &gorm.Statement{
			Clauses:      map[string]clause.Clause{},
			BuildClauses: queryClauses,
		},
	}
	if b.unscoped {
		tx = tx.Unscoped()
	}
	tx.Statement.DB = tx
	b.buildStmt(tx.Statement, quote())
	callbacks.BuildQuerySQL(tx)
	return clause.Expr{SQL: tx.Statement.SQL.String(), Vars: tx.Statement.Vars}
}

func (b *QueryBuilderG[T]) Clone() *QueryBuilderG[T] {
	return &QueryBuilderG[T]{
		selects:  slices.Clone(b.selects),
		from:     b.from,
		joins:    slices.Clone(b.joins),
		wheres:   slices.Clone(b.wheres),
		orders:   slices.Clone(b.orders),
		offset:   b.offset,
		limit:    b.limit,
		unscoped: b.unscoped,
	}
}

func (b *QueryBuilderG[T]) Order(column field.IField, asc ...bool) *QueryBuilderG[T] {
	b.orders = append(b.orders, order{column, optional(asc, true)})
	return b
}

func (b *QueryBuilderG[T]) Paginate(p Paginate) *QueryBuilderG[T] {
	page := max(1, p.Page)
	pageSize := max(1, p.PageSize)
	b.Offset((page - 1) * pageSize)
	b.Limit(pageSize)
	return b
}

func (b *QueryBuilderG[T]) Offset(offset int) *QueryBuilderG[T] {
	b.offset = offset
	return b
}

func (b *QueryBuilderG[T]) Limit(limit int) *QueryBuilderG[T] {
	b.limit = limit
	return b
}

func (b *QueryBuilderG[T]) Scope(fn ScopeFuncG[T]) *QueryBuilderG[T] {
	return b.Scopes(fn)
}

func (b *QueryBuilderG[T]) Scopes(fns ...ScopeFuncG[T]) *QueryBuilderG[T] {
	for _, fn := range fns {
		fn(b)
	}
	return b
}

func (b *QueryBuilderG[T]) Unscoped() *QueryBuilderG[T] {
	b.unscoped = true
	return b
}

func (b *QueryBuilderG[T]) Distinct() *QueryBuilderG[T] {
	b.distinct = true
	return b
}

func (b *QueryBuilderG[T]) Create(db IDB, value *T) DBResult {
	builder := b.Clone()
	builder.from = nil
	builder.selects = nil
	builder.wheres = nil
	builder.from = TableName2("")
	ret := builder.build(db).Create(value)
	return DBResult{
		ret.Error,
		ret.RowsAffected,
	}
}

func (b *QueryBuilderG[T]) Update(db IDB, values any) DBResult {
	ret := b.build(db).Updates(values)
	return DBResult{
		ret.Error,
		ret.RowsAffected,
	}
}

func (b *QueryBuilderG[T]) Delete(db IDB) DBResult {
	var dest T
	ret := b.build(db).Delete(&dest)
	return DBResult{
		ret.Error,
		ret.RowsAffected,
	}
}

func (b *QueryBuilderG[T]) Count(db IDB) (count int64, _ error) {
	ret := b.build(db).Count(&count)
	return count, ret.Error
}

func (b *QueryBuilderG[T]) Exist(db IDB) (bool, error) {
	var count int64
	tx := b.Clone().Limit(1).build(db).Count(&count)
	return count > 0, tx.Error
}

func (b *QueryBuilderG[T]) Take(db IDB) (*T, error) {
	return b.firstLast(db, false, false)
}

func (b *QueryBuilderG[T]) First(db IDB) (*T, error) {
	return b.firstLast(db, true, false)
}

func (b *QueryBuilderG[T]) Last(db IDB) (*T, error) {
	return b.firstLast(db, true, true)
}

func (b *QueryBuilderG[T]) Find(db IDB) ([]*T, error) {
	var dest []*T
	ret := b.build(db).Find(&dest)
	if ret.RowsAffected == 0 {
		return nil, nil
	} else if ret.Error != nil {
		return nil, ret.Error
	}
	return dest, ret.Error
}

func (b *QueryBuilderG[T]) AsField(asName string) field.IField {
	if len(b.selects) == 0 {
		panic("selects is empty")
		//if v, ok := b.from.(interface{ ModelType() *T }); ok {
		//
		//} else {
		//	panic("")
		//}
	} else {
		b.selects = b.selects[0:1]
	}
	e := b.ToExpr()
	return field.NewBaseFromSql(clause.Expr{
		SQL:  e.SQL,
		Vars: e.Vars,
	}, asName)
}

func (b *QueryBuilderG[T]) firstLast(db IDB, order, desc bool) (*T, error) {
	var dest T
	err := firstLast(b, db, order, desc, &dest)
	return &dest, err
}

func firstLast[T any](b *QueryBuilderG[T], db IDB, order, desc bool, dest any) error {
	tx := b.Clone().Limit(1).build(db)
	stmt := tx.GetStatement()
	stmt.RaiseErrorOnNotFound = true

	if lo.IsNil(stmt.Model) {
		if v, ok := b.from.(interface{ ModelTypeAny() any }); ok {
			stmt.Model = v.ModelTypeAny()
		}
	}

	if order && !lo.IsNil(stmt.Model) {
		stmt.AddClause(clause.OrderBy{
			Columns: []clause.OrderByColumn{
				{
					Column: clause.Column{Table: clause.CurrentTable, Name: clause.PrimaryKey},
					Desc:   desc,
				},
			},
		})
	}

	if err := tx.Find(dest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = nil
		}
		return err
	}
	return nil
}

func optional[T any](args []T, def T) T {
	if len(args) == 0 {
		return def
	}
	return args[0]
}

func quote() func(field string) string {
	return func(field string) string {
		var writer strings.Builder
		dialector.QuoteTo(&writer, field)
		return writer.String()
	}
}

func (b *QueryBuilderG[T]) build(db IDB) IDB {
	tx := db.Table("")
	if b.unscoped {
		tx = tx.Unscoped()
	}
	b.buildStmt(tx.Statement, quote())
	return NewDefaultGormDB(tx)
}

func (b *QueryBuilderG[T]) buildStmt(stmt *gorm.Statement, quote func(field string) string) {
	stmt.Distinct = b.distinct
	if v, ok := b.from.(ICompactFrom); ok {
		stmt.TableExpr = &clause.Expr{SQL: "(?) AS " + v.TableName(), Vars: []any{v.Expr()}}
		stmt.Table = v.TableName()
	} else {
		tn := b.from.TableName()
		if v, ok := b.from.(interface{ Alias() string }); ok {
			alias := v.Alias()
			if tn != alias && len(alias) > 0 {
				tn = fmt.Sprintf("%s AS %s", tn, alias)
			}
		}
		stmt.TableExpr, stmt.Table = txTable(quote, tn)
	}
	addSelects(stmt, b.selects)
	if len(b.wheres) > 0 {
		stmt.AddClause(clause.Where{Exprs: b.wheres})
	}
	//for _, where := range b.wheres {
	//	if conds := stmt.BuildCondition(where.Query, where.Args...); len(conds) > 0 {
	//	}
	//}
	for _, join := range b.joins {
		_from := stmt.Clauses["FROM"]
		fromClause := clause.From{}
		if v, ok := _from.Expression.(clause.From); ok {
			fromClause = v
		}
		fromClause.Joins = append(fromClause.Joins, clause.Join{Expression: join})
		_from.Expression = fromClause
		stmt.Clauses["FROM"] = _from
	}
	if b.offset > 0 {
		stmt.AddClause(clause.Limit{Offset: b.offset})
	}
	if b.limit > 0 {
		stmt.AddClause(clause.Limit{Limit: &b.limit})
	}
	var orderBy clause.OrderBy
	for _, order := range b.orders {
		c := order.field
		if c.IsExpr() {
			continue
		}
		orderBy.Columns = append(orderBy.Columns, clause.OrderByColumn{
			Column: c.ToColumn(),
			Desc:   !order.asc,
		})
	}
	if len(orderBy.Columns) > 0 {
		stmt.AddClause(orderBy)
	}
}

////////////////////////////////////////////////

var dialector = mysql.Dialector{
	Config: &mysql.Config{
		DSNConfig: &mysql2.Config{
			Loc: time.UTC,
		},
	},
}

//func (b *QueryBuilderG[T]) ToFieldG(asName string) field.Pattern[T] {
//	e := b.ToExpr()
//	base := field.NewBaseFromSql(clause.Expr{
//		SQL:  e.SQL,
//		Vars: e.Vars,
//	}, asName, b.from.TableName())
//	return field.NewPatternWith[T](*base)
//}

//// Scan 执行查询
//func (b *QueryBuilderG[T]) Scan(db IDB) (*T, error) {
//	var def T
//	ret := b.build(db).Scan(&def)
//	if ret.RowsAffected == 0 {
//		return nil, nil
//	} else if ret.Error != nil {
//		return nil, ret.Error
//	}
//	return &def, nil
//}

//func (b *QueryBuilderG[T]) Pluck(db IDB, column interface {
//	field.IField
//	field.IFieldType[T]
//}) ([]T, error) {
//	var name = field.ExtractColumn(column)
//	builder := b.Clone()
//	builder.selects = nil
//
//	var dest []T
//	err := builder.build(db).Pluck(name, &dest).Error
//	return dest, err
//}
