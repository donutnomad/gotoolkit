package gsql

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"github.com/donutnomad/gotoolkit/lib/gsql/internal/utils"
	"github.com/samber/lo"
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
	// group by / having
	groupBy []field.IField
	having  []clause.Expression
	// locking (FOR UPDATE/SHARE ... NOWAIT/SKIP LOCKED)
	locking *clause.Locking
	// table hints on FROM
	fromIndexHints []indexHint
	fromPartitions []string
	// CTE (Common Table Expressions)
	cte *CTEClause
}

func SelectG[T any](fields ...field.IField) *baseQueryBuilderG[T] {
	return (&baseQueryBuilderG[T]{}).Select(fields...)
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
	return (&baseQueryBuilderG[T]{}).Select().From(from)
}

type baseQueryBuilderG[T any] struct {
	selects []field.IField
	cte     *CTEClause
}

func (b *baseQueryBuilderG[T]) Select(fields ...field.IField) *baseQueryBuilderG[T] {
	for _, f := range fields {
		if v, ok := f.(field.BaseFields); ok {
			b.selects = append(b.selects, v...)
		} else {
			b.selects = append(b.selects, f)
		}
	}
	return b
}

func (b *baseQueryBuilderG[T]) From(table interface {
	TableName() string
}) *QueryBuilderG[T] {
	qb := &QueryBuilderG[T]{
		selects: b.selects,
		from:    table,
		cte:     b.cte,
	}
	return qb
}

func (b *QueryBuilderG[T]) Join(clauses ...JoinClause) *QueryBuilderG[T] {
	b.joins = append(b.joins, clauses...)
	return b
}

func (b *QueryBuilderG[T]) Where(exprs ...field.Expression) *QueryBuilderG[T] {
	b.wheres = append(b.wheres, exprs...)
	return b
}

func (b *QueryBuilderG[T]) ToSQL() string {
	expr := b.ToExpr()
	return dialector.Explain(expr.SQL, expr.Vars...)
}

func (b *QueryBuilderG[T]) String() string {
	return b.ToSQL()
}

func (b *QueryBuilderG[T]) Clone() *QueryBuilderG[T] {
	var cte *CTEClause
	if b.cte != nil {
		cte = &CTEClause{
			CTEs:      slices.Clone(b.cte.CTEs),
			Recursive: b.cte.Recursive,
		}
	}
	return &QueryBuilderG[T]{
		selects:        slices.Clone(b.selects),
		from:           b.from,
		joins:          slices.Clone(b.joins),
		wheres:         slices.Clone(b.wheres),
		orders:         slices.Clone(b.orders),
		offset:         b.offset,
		limit:          b.limit,
		unscoped:       b.unscoped,
		groupBy:        slices.Clone(b.groupBy),
		having:         slices.Clone(b.having),
		locking:        b.locking,
		fromIndexHints: slices.Clone(b.fromIndexHints),
		fromPartitions: slices.Clone(b.fromPartitions),
		cte:            cte,
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

// GroupBy adds GROUP BY columns
func (b *QueryBuilderG[T]) GroupBy(cols ...field.IField) *QueryBuilderG[T] {
	b.groupBy = append(b.groupBy, cols...)
	return b
}

// Having adds HAVING expressions
func (b *QueryBuilderG[T]) Having(exprs ...field.Expression) *QueryBuilderG[T] {
	b.having = append(b.having, exprs...)
	return b
}

// ForUpdate sets locking to FOR UPDATE
func (b *QueryBuilderG[T]) ForUpdate() *QueryBuilderG[T] {
	if b.locking == nil {
		b.locking = &clause.Locking{}
	}
	b.locking.Strength = clause.LockingStrengthUpdate
	return b
}

// ForShare sets locking to FOR SHARE
func (b *QueryBuilderG[T]) ForShare() *QueryBuilderG[T] {
	if b.locking == nil {
		b.locking = &clause.Locking{}
	}
	b.locking.Strength = clause.LockingStrengthShare
	return b
}

// Nowait adds NOWAIT option to locking
func (b *QueryBuilderG[T]) Nowait() *QueryBuilderG[T] {
	if b.locking == nil {
		b.locking = &clause.Locking{}
	}
	b.locking.Options = clause.LockingOptionsNoWait
	return b
}

// SkipLocked adds SKIP LOCKED option to locking
func (b *QueryBuilderG[T]) SkipLocked() *QueryBuilderG[T] {
	if b.locking == nil {
		b.locking = &clause.Locking{}
	}
	b.locking.Options = clause.LockingOptionsSkipLocked
	return b
}

// Partition sets PARTITION list for FROM table
func (b *QueryBuilderG[T]) Partition(parts ...string) *QueryBuilderG[T] {
	b.fromPartitions = append(b.fromPartitions, parts...)
	return b
}

// UseIndex appends USE INDEX (idx, ...) hint
func (b *QueryBuilderG[T]) UseIndex(indexes ...string) *QueryBuilderG[T] {
	b.fromIndexHints = append(b.fromIndexHints, indexHint{action: "USE", indexNames: indexes})
	return b
}

// IgnoreIndex appends IGNORE INDEX (idx, ...) hint
func (b *QueryBuilderG[T]) IgnoreIndex(indexes ...string) *QueryBuilderG[T] {
	b.fromIndexHints = append(b.fromIndexHints, indexHint{action: "IGNORE", indexNames: indexes})
	return b
}

// ForceIndex appends FORCE INDEX (idx, ...) hint
func (b *QueryBuilderG[T]) ForceIndex(indexes ...string) *QueryBuilderG[T] {
	b.fromIndexHints = append(b.fromIndexHints, indexHint{action: "FORCE", indexNames: indexes})
	return b
}

// UseIndexForJoin appends USE INDEX FOR JOIN (idx, ...) hint
func (b *QueryBuilderG[T]) UseIndexForJoin(indexes ...string) *QueryBuilderG[T] {
	b.fromIndexHints = append(b.fromIndexHints, indexHint{action: "USE", forTarget: "JOIN", indexNames: indexes})
	return b
}

// IgnoreIndexForJoin appends IGNORE INDEX FOR JOIN (idx, ...) hint
func (b *QueryBuilderG[T]) IgnoreIndexForJoin(indexes ...string) *QueryBuilderG[T] {
	b.fromIndexHints = append(b.fromIndexHints, indexHint{action: "IGNORE", forTarget: "JOIN", indexNames: indexes})
	return b
}

// ForceIndexForJoin appends FORCE INDEX FOR JOIN (idx, ...) hint
func (b *QueryBuilderG[T]) ForceIndexForJoin(indexes ...string) *QueryBuilderG[T] {
	b.fromIndexHints = append(b.fromIndexHints, indexHint{action: "FORCE", forTarget: "JOIN", indexNames: indexes})
	return b
}

// UseIndexForOrderBy appends USE INDEX FOR ORDER BY (idx, ...) hint
func (b *QueryBuilderG[T]) UseIndexForOrderBy(indexes ...string) *QueryBuilderG[T] {
	b.fromIndexHints = append(b.fromIndexHints, indexHint{action: "USE", forTarget: "ORDER BY", indexNames: indexes})
	return b
}

// IgnoreIndexForOrderBy appends IGNORE INDEX FOR ORDER BY (idx, ...) hint
func (b *QueryBuilderG[T]) IgnoreIndexForOrderBy(indexes ...string) *QueryBuilderG[T] {
	b.fromIndexHints = append(b.fromIndexHints, indexHint{action: "IGNORE", forTarget: "ORDER BY", indexNames: indexes})
	return b
}

// ForceIndexForOrderBy appends FORCE INDEX FOR ORDER BY (idx, ...) hint
func (b *QueryBuilderG[T]) ForceIndexForOrderBy(indexes ...string) *QueryBuilderG[T] {
	b.fromIndexHints = append(b.fromIndexHints, indexHint{action: "FORCE", forTarget: "ORDER BY", indexNames: indexes})
	return b
}

// UseIndexForGroupBy appends USE INDEX FOR GROUP BY (idx, ...) hint
func (b *QueryBuilderG[T]) UseIndexForGroupBy(indexes ...string) *QueryBuilderG[T] {
	b.fromIndexHints = append(b.fromIndexHints, indexHint{action: "USE", forTarget: "GROUP BY", indexNames: indexes})
	return b
}

// IgnoreIndexForGroupBy appends IGNORE INDEX FOR GROUP BY (idx, ...) hint
func (b *QueryBuilderG[T]) IgnoreIndexForGroupBy(indexes ...string) *QueryBuilderG[T] {
	b.fromIndexHints = append(b.fromIndexHints, indexHint{action: "IGNORE", forTarget: "GROUP BY", indexNames: indexes})
	return b
}

// ForceIndexForGroupBy appends FORCE INDEX FOR GROUP BY (idx, ...) hint
func (b *QueryBuilderG[T]) ForceIndexForGroupBy(indexes ...string) *QueryBuilderG[T] {
	b.fromIndexHints = append(b.fromIndexHints, indexHint{action: "FORCE", forTarget: "GROUP BY", indexNames: indexes})
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
	tx := b.build(db)
	//ret := tx.Find(&dest)
	ret := Scan(tx, &dest)
	if ret.RowsAffected == 0 {
		return nil, nil
	} else if ret.Error != nil {
		return nil, ret.Error
	}
	return dest, ret.Error
}

// AsF as field
func (b *QueryBuilderG[T]) AsF(asName ...string) field.IField {
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
	return FieldExpr(b.ToExpr(), optional(asName, ""))
}

func (b *QueryBuilderG[T]) firstLast(db IDB, order, desc bool) (*T, error) {
	var dest T
	err := firstLast(b, db, order, desc, &dest)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &dest, nil
}

func (b *QueryBuilderG[T]) ToExpr() clause.Expr {
	tx := &gorm.DB{
		Config: &gorm.Config{
			ClauseBuilders: map[string]clause.ClauseBuilder{
				"CTE": func(c clause.Clause, builder clause.Builder) {
					if cte, ok := c.Expression.(CTEClause); ok {
						cte.Build(builder)
					}
				},
			},
			Dialector: dialector,
		},
		Statement: &gorm.Statement{
			Clauses:      map[string]clause.Clause{},
			BuildClauses: queryClauses,
		},
	}
	tx.Statement.DB = tx
	b.buildStmt(tx.Statement, getQuoteFunc())
	callbacks.BuildQuerySQL(tx)
	return clause.Expr{SQL: tx.Statement.SQL.String(), Vars: tx.Statement.Vars}
}

func (b *QueryBuilderG[T]) build(db IDB) *gorm.DB {
	tx := db.Session(&gorm.Session{
		Initialized: true,
	})
	tx.Config.ClauseBuilders["CTE"] = func(c clause.Clause, builder clause.Builder) {
		if cte, ok := c.Expression.(CTEClause); ok {
			cte.Build(builder)
		}
	}
	b.buildStmt(tx.Statement, getQuoteFunc())
	return tx
}

func (b *QueryBuilderG[T]) buildStmt(stmt *gorm.Statement, quote func(field string) string) {
	if b.unscoped {
		stmt.Unscoped = true
	}
	// 添加 CTE 子句
	if b.cte != nil {
		stmt.AddClause(*b.cte)
	}
	stmt.Distinct = b.distinct
	if v, ok := b.from.(ICompactFrom); ok {
		stmt.TableExpr = &clause.Expr{SQL: "(?) AS " + v.TableName(), Vars: []any{v.ToExpr()}}
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
		// decorate table expr with partition / index hints if present
		if stmt.TableExpr != nil {
			expr := stmt.TableExpr
			suffix := strings.TrimSpace(strings.Join([]string{
				buildPartitionSQL(quote, b.fromPartitions),
				buildIndexHintsSQL(quote, b.fromIndexHints),
			}, " "))
			if len(suffix) > 0 {
				expr.SQL = expr.SQL + " " + suffix
			}
		}
	}
	addSelects(stmt, b.selects)
	if len(b.wheres) > 0 {
		stmt.AddClause(clause.Where{Exprs: b.wheres})
	}
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
			Column: clause.Column{
				Name: c.FullName(),
				Raw:  true,
			},
			Desc: !order.asc,
		})
	}
	if len(orderBy.Columns) > 0 {
		stmt.AddClause(orderBy)
	}
	// GROUP BY / HAVING
	if len(b.groupBy) > 0 || len(b.having) > 0 {
		var cols []clause.Column
		if len(b.groupBy) > 0 {
			cols = make([]clause.Column, 0, len(b.groupBy))
			for _, gb := range b.groupBy {
				cols = append(cols, clause.Column{Name: gb.FullName(), Raw: true})
			}
		}
		stmt.AddClause(clause.GroupBy{Columns: cols, Having: b.having})
	}
	// FOR locking
	if b.locking != nil {
		stmt.AddClause(*b.locking)
	}
}

////////////////////////////////////////////////

var dialector = utils.Dialector

// ---------- table hints helpers ----------
type indexHint struct {
	action     string // USE | IGNORE | FORCE
	forTarget  string // "" | JOIN | ORDER BY | GROUP BY
	indexNames []string
}

func buildIndexHintsSQL(quote func(string) string, hints []indexHint) string {
	if len(hints) == 0 {
		return ""
	}
	var parts []string
	for _, h := range hints {
		if len(h.indexNames) == 0 || h.action == "" {
			continue
		}
		var b strings.Builder
		b.WriteString(h.action)
		b.WriteString(" INDEX")
		if h.forTarget != "" {
			b.WriteString(" FOR ")
			b.WriteString(h.forTarget)
		}
		b.WriteString(" (")
		for i, idx := range h.indexNames {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(quote(idx))
		}
		b.WriteString(")")
		parts = append(parts, b.String())
	}
	return strings.Join(parts, " ")
}

func buildPartitionSQL(quote func(string) string, parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("PARTITION (")
	for i, p := range parts {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(quote(p))
	}
	b.WriteString(")")
	return b.String()
}

func firstLast[T any](b *QueryBuilderG[T], db IDB, order, desc bool, dest any) error {
	tx := b.Clone().Limit(1).build(db)
	stmt := tx.Statement
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

	stmt.RaiseErrorOnNotFound = true
	ret := Scan(tx, dest)
	//if err := tx.Find(dest).Error; err != nil {
	if err := ret.Error; err != nil {
		return err
	}
	return nil
}
