package gsql

import "C"
import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"

	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// INSERT INTO ... VALUES
// INSERT INTO ... ON DUPLICATE KEY UPDATE

// INSERT IGNORE ... VALUES
// * 如果插入的行没有导致唯一性约束（主键或唯一索引）冲突，则正常插入。
// * 如果插入的行导致唯一性约束冲突，则 忽略该行，不会插入，也不会返回任何错误。

// INSERT INTO ... SELECT

// REPLACE INTO ... SELECT
// * DELETE + INSERT, 先删除后插入，id会自增
// * 如果新数据某些列没有提供值，它们将使用默认值，旧数据中未提供的列的值会丢失
// * 会触发 DELETE 和 INSERT 相关的触发器
// * 通常性能较低，特别是索引多或有复杂触发器时

type InsertBuilder[T any] struct {
	ignore        bool
	selectColumns []field.IField
}

type insertBuilderWithValues[T any] struct {
	selectColumns            []field.IField
	onDuplicateUpdateColumns []field.IField
	values                   *[]T
	onDuplicateUpdate        bool
	ignore                   bool
}

func InsertInto[T any, TableTypes interface {
	ModelType() T
}](_ TableTypes, columns ...field.IField) *InsertBuilder[T] {
	return &InsertBuilder[T]{
		selectColumns: columns,
	}
}

func InsertIgnore[T any, TableTypes interface {
	ModelType() T
}](_ TableTypes, columns ...field.IField) *InsertBuilder[T] {
	return &InsertBuilder[T]{
		ignore:        true,
		selectColumns: columns,
	}
}

func (b *InsertBuilder[T]) Value(value T) *insertBuilderWithValues[T] {
	return &insertBuilderWithValues[T]{
		selectColumns: b.selectColumns,
		values:        &[]T{value},
		ignore:        b.ignore,
	}
}

func (b *InsertBuilder[T]) Values(values *[]T) *insertBuilderWithValues[T] {
	return &insertBuilderWithValues[T]{
		selectColumns: b.selectColumns,
		values:        values,
		ignore:        b.ignore,
	}
}

func (b *insertBuilderWithValues[T]) DuplicateUpdate(columns ...field.IField) *insertBuilderWithValues[T] {
	b.onDuplicateUpdate = true
	b.onDuplicateUpdateColumns = columns
	return b
}

func (b *insertBuilderWithValues[T]) Exec(db IDB) error {
	_, err := b.ExecWithResult(db)
	return err
}

func (b *insertBuilderWithValues[T]) ExecWithResult(db IDB) (int64, error) {
	var tx = db.Model(lo.Empty[T]())
	if len(b.selectColumns) > 0 {
		addSelects(tx.Statement, b.selectColumns)
		//tx = tx.Select(lo.Map(b.selectColumns, func(item field.IField, index int) string {
		//	return item.Name()
		//}))
	}
	if b.ignore {
		tx = tx.Clauses(clause.Insert{
			Modifier: "IGNORE",
		})
	} else if b.onDuplicateUpdate {
		tx = tx.Clauses(clause.OnConflict{
			UpdateAll: true,
		})
	}

	var allowColumns = lo.Map(b.onDuplicateUpdateColumns, func(item field.IField, index int) string {
		return item.Name()
	})
	if len(allowColumns) > 0 {
		tx.Statement.Dest = b.values
		tx.Statement.SQL.Reset()
		tx.Statement.Vars = nil

		tx = processerExec(createClauses, tx)
		if tx.Error != nil {
			return 0, tx.Error
		}
		stmt := tx.Statement

		stmt.SQL.Grow(180)
		stmt.AddClauseIfNotExists(clause.Insert{})
		stmt.AddClause(callbacks.ConvertToCreateValues(stmt))
		if v, ok := stmt.Clauses[clause.OnConflict{}.Name()]; ok {
			if v1, ok := v.Expression.(clause.OnConflict); ok {
				var doUpdates clause.Set
				for _, item := range v1.DoUpdates {
					if _, ok := item.Value.(time.Time); ok {
						doUpdates = append(doUpdates, item)
					} else if slices.Contains(allowColumns, item.Column.Name) {
						doUpdates = append(doUpdates, item)
					}
				}
				v1.DoUpdates = doUpdates
				v1.MergeClause(&v)
				stmt.Clauses[clause.OnConflict{}.Name()] = v
			}
		}

		stmt.Build(createClauses...)
	}

	ret := tx.Create(b.values)
	return ret.RowsAffected, ret.Error
}

func processerExec(pClauses []string, db *gorm.DB) *gorm.DB {
	// call scopes
	//for len(db.Statement.scopes) > 0 {
	//	db = db.executeScopes()
	//}

	var (
		curTime           = time.Now()
		stmt              = db.Statement
		resetBuildClauses bool
	)

	if len(stmt.BuildClauses) == 0 {
		stmt.BuildClauses = pClauses
		resetBuildClauses = true
	}

	if optimizer, ok := db.Statement.Dest.(gorm.StatementModifier); ok {
		optimizer.ModifyStatement(stmt)
	}

	// assign model values
	if stmt.Model == nil {
		stmt.Model = stmt.Dest
	} else if stmt.Dest == nil {
		stmt.Dest = stmt.Model
	}

	// parse model values
	if stmt.Model != nil {
		if err := stmt.Parse(stmt.Model); err != nil && (!errors.Is(err, schema.ErrUnsupportedDataType) || (stmt.Table == "" && stmt.TableExpr == nil && stmt.SQL.Len() == 0)) {
			if errors.Is(err, schema.ErrUnsupportedDataType) && stmt.Table == "" && stmt.TableExpr == nil {
				db.AddError(fmt.Errorf("%w: Table not set, please set it like: db.Model(&user) or db.Table(\"users\")", err))
			} else {
				db.AddError(err)
			}
		}
	}

	// assign stmt.ReflectValue
	if stmt.Dest != nil {
		stmt.ReflectValue = reflect.ValueOf(stmt.Dest)
		for stmt.ReflectValue.Kind() == reflect.Ptr {
			if stmt.ReflectValue.IsNil() && stmt.ReflectValue.CanAddr() {
				stmt.ReflectValue.Set(reflect.New(stmt.ReflectValue.Type().Elem()))
			}

			stmt.ReflectValue = stmt.ReflectValue.Elem()
		}
		if !stmt.ReflectValue.IsValid() {
			db.AddError(gorm.ErrInvalidValue)
		}
	}

	//for _, f := range p.fns {
	//	f(db)
	//}
	//
	//if stmt.SQL.Len() > 0 {
	//	db.Logger.Trace(stmt.Context, curTime, func() (string, int64) {
	//		sql, vars := stmt.SQL.String(), stmt.Vars
	//		if filter, ok := db.Logger.(ParamsFilter); ok {
	//			sql, vars = filter.ParamsFilter(stmt.Context, stmt.SQL.String(), stmt.Vars...)
	//		}
	//		return db.Dialector.Explain(sql, vars...), db.RowsAffected
	//	}, db.Error)
	//}
	//
	//if !stmt.DB.DryRun {
	//	stmt.SQL.Reset()
	//	stmt.Vars = nil
	//}
	//
	//if resetBuildClauses {
	//	stmt.BuildClauses = nil
	//}

	_ = curTime
	_ = resetBuildClauses
	return db
}

//////////////////////// select /////////////////////////////

func (b *InsertBuilder[T]) Select(q interface{ ToExpr() clause.Expr }) *insertBuilderWithSelect[T] {
	return &insertBuilderWithSelect[T]{
		selectColumns: b.selectColumns,
		ignore:        b.ignore,
		query:         q.ToExpr(),
	}
}

type insertBuilderWithSelect[T any] struct {
	ignore                   bool
	onDuplicateUpdate        bool
	onDuplicateUpdateColumns []field.IField
	query                    clause.Expr
	selectColumns            []field.IField
}

func (b *insertBuilderWithSelect[T]) DuplicateUpdate(columns ...field.IField) *insertBuilderWithSelect[T] {
	b.onDuplicateUpdate = true
	b.onDuplicateUpdateColumns = columns
	return b
}

func (b *insertBuilderWithSelect[T]) Exec(db IDB) error {
	_, err := b.ExecWithResult(db)
	return err
}

func (b *insertBuilderWithSelect[T]) ExecWithResult(db IDB) (int64, error) {
	var tx = db.Model(lo.Empty[T]())
	if len(b.selectColumns) > 0 {
		tx = tx.Select(lo.Map(b.selectColumns, func(item field.IField, index int) string {
			return item.Name()
		}))
	}
	if b.ignore {
		tx = tx.Clauses(clause.Insert{
			Modifier: "IGNORE",
		})
	} else if b.onDuplicateUpdate {
		tx = tx.Clauses(clause.OnConflict{
			UpdateAll: true,
		})
	}

	var def = []T{lo.Empty[T]()}

	tx.Statement.Dest = &def
	tx.Statement.SQL.Reset()
	tx.Statement.Vars = nil

	tx = processerExec(createClauses, tx)
	if tx.Error != nil {
		return 0, tx.Error
	}
	stmt := tx.Statement

	stmt.SQL.Grow(180)
	stmt.AddClauseIfNotExists(clause.Insert{})

	values := valuesWhere{Columns: make([]clause.Column, 0, len(stmt.Schema.DBNames))}
	selectColumns, restricted := stmt.SelectAndOmitColumns(true, false)
	for _, field := range stmt.Schema.FieldsWithDefaultDBValue {
		if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && !restricted) {
			values.Columns = append(values.Columns, clause.Column{Name: field.DBName})
		}
	}
	for _, db := range stmt.Schema.DBNames {
		if field := stmt.Schema.FieldsByDBName[db]; !field.HasDefaultValue || field.DefaultValueInterface != nil {
			if v, ok := selectColumns[db]; (ok && v) || (!ok && (!restricted || field.AutoCreateTime > 0 || field.AutoUpdateTime > 0)) {
				values.Columns = append(values.Columns, clause.Column{Name: db})
			}
		}
	}
	values.query = b.query
	stmt.AddClause(values)

	var allowColumns = lo.Map(b.onDuplicateUpdateColumns, func(item field.IField, index int) string {
		return item.Name()
	})
	if len(allowColumns) > 0 {
		//if v, ok := stmt.Clauses[clause.OnConflict{}.Name()]; ok {
		//	if v1, ok := v.Expression.(clause.OnConflict); ok {
		//		var doUpdates clause.Set
		//		for _, item := range v1.DoUpdates {
		//			if _, ok := item.Value.(time.Time); ok {
		//				doUpdates = append(doUpdates, item)
		//			} else if slices.Contains(allowColumns, item.Column.Name) {
		//				doUpdates = append(doUpdates, item)
		//			}
		//		}
		//		v1.DoUpdates = doUpdates
		//		v1.MergeClause(&v)
		//		stmt.Clauses[clause.OnConflict{}.Name()] = v
		//	}
		//}
	}

	stmt.Build(createClauses...)

	ret := tx.Create(&def)
	return ret.RowsAffected, ret.Error
}

type valuesWhere struct {
	Columns []clause.Column
	query   clause.Expression
}

// Name from clause name
func (valuesWhere) Name() string {
	return "VALUES"
}

// Build from clause
func (values valuesWhere) Build(builder clause.Builder) {
	writer := &safeWriter{builder}
	if len(values.Columns) > 0 {
		writer.WriteByte('(')
		for idx, column := range values.Columns {
			if idx > 0 {
				writer.WriteByte(',')
			}
			writer.WriteQuoted(column)
		}
		writer.WriteByte(')')
		writer.WriteByte(' ')
		values.query.Build(builder)
	} else {
		writer.WriteString("DEFAULT VALUES")
	}
}

// MergeClause merge values clauses
func (values valuesWhere) MergeClause(clause *clause.Clause) {
	clause.Name = ""
	clause.Expression = values
}
