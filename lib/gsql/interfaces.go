package gsql

import (
	"context"
	"database/sql"
	"time"

	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ExprTo struct {
	clause.Expression
}

func (e ExprTo) AsF(name ...string) field.IField {
	return FieldExpr(e.Expression, optional(name, ""))
}

func (e ExprTo) ToExpr() field.Expression {
	return e.Expression
}

type primitive interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64 | ~string | time.Time | *time.Time
}

type DBResult struct {
	Error        error
	RowsAffected int64
}

type ICompactFrom interface {
	field.IToExpr
	TableName() string
}

type ITableName interface {
	TableName() string
}

type IDB interface {
	Model(value any) (tx *gorm.DB)
	Session(config *gorm.Session) *gorm.DB
}

type DefaultGormDB struct {
	getDB func() *gorm.DB
}

func (d *DefaultGormDB) DB() (*sql.DB, error) {
	return d.getDB().DB()
}

func NewDefaultGormDB(db *gorm.DB) *DefaultGormDB {
	return &DefaultGormDB{getDB: func() *gorm.DB {
		return db
	}}
}

func NewDefaultGormDBByFn(f func() *gorm.DB) *DefaultGormDB {
	return &DefaultGormDB{getDB: f}
}

func (d *DefaultGormDB) Group(name string) *gorm.DB {
	return d.getDB().Group(name)
}

func (d *DefaultGormDB) Assign(attrs ...any) (tx *gorm.DB) {
	return d.getDB().Assign(attrs...)
}

func (d *DefaultGormDB) Find(dest any, conds ...any) (tx *gorm.DB) {
	return d.getDB().Find(dest, conds...)
}

func (d *DefaultGormDB) Pluck(column string, dest any) *gorm.DB {
	return d.getDB().Pluck(column, dest)
}

func (d *DefaultGormDB) Count(count *int64) *gorm.DB {
	return d.getDB().Count(count)
}

func (d *DefaultGormDB) Updates(values any) *gorm.DB {
	return d.getDB().Updates(values)
}

func (d *DefaultGormDB) Update(column string, value any) *gorm.DB {
	return d.getDB().Update(column, value)
}

func (d *DefaultGormDB) UpdateColumn(column string, value any) *gorm.DB {
	return d.getDB().UpdateColumn(column, value)
}

func (d *DefaultGormDB) UpdateColumns(values any) *gorm.DB {
	return d.getDB().UpdateColumns(values)
}

func (d *DefaultGormDB) Take(dest any, conds ...any) *gorm.DB {
	return d.getDB().Take(dest, conds...)
}

func (d *DefaultGormDB) Last(dest any, conds ...any) *gorm.DB {
	return d.getDB().Last(dest, conds...)
}

func (d *DefaultGormDB) Distinct(args ...any) *gorm.DB {
	return d.getDB().Distinct(args...)
}

func (d *DefaultGormDB) Offset(offset int) *gorm.DB {
	return d.getDB().Offset(offset)
}

func (d *DefaultGormDB) Limit(limit int) *gorm.DB {
	return d.getDB().Limit(limit)
}

func (d *DefaultGormDB) Order(value any) *gorm.DB {
	return d.getDB().Order(value)
}

func (d *DefaultGormDB) Joins(query string, args ...any) *gorm.DB {
	return d.getDB().Joins(query, args...)
}

func (d *DefaultGormDB) InnerJoins(query string, args ...any) *gorm.DB {
	return d.getDB().InnerJoins(query, args...)
}

func (d *DefaultGormDB) GetStatement() *gorm.Statement {
	return d.getDB().Statement
}

func (d *DefaultGormDB) Model(value any) (tx *gorm.DB) {
	return d.getDB().Model(value)
}

func (d *DefaultGormDB) Scan(dest any) (tx *gorm.DB) {
	return d.getDB().Scan(dest)
}

func (d *DefaultGormDB) Session(config *gorm.Session) *gorm.DB {
	return d.getDB().Session(config)
}

func (d *DefaultGormDB) Create(value any) (tx *gorm.DB) {
	return d.getDB().Create(value)
}

func (d *DefaultGormDB) Select(query any, args ...any) (tx *gorm.DB) {
	return d.getDB().Select(query, args...)
}

func (d *DefaultGormDB) Save(value any) (tx *gorm.DB) {
	return d.getDB().Save(value)
}

func (d *DefaultGormDB) Where(query any, args ...any) (tx *gorm.DB) {
	return d.getDB().Where(query, args...)
}

func (d *DefaultGormDB) Table(name string, args ...any) (tx *gorm.DB) {
	return d.getDB().Table(name, args...)
}

func (d *DefaultGormDB) Delete(value any, conds ...any) (tx *gorm.DB) {
	return d.getDB().Delete(value, conds...)
}

func (d *DefaultGormDB) Transaction(fn func(tx *gorm.DB) error, opts ...*sql.TxOptions) error {
	return d.getDB().Transaction(fn, opts...)
}

func (d *DefaultGormDB) AutoMigrate(table ...any) error {
	return d.getDB().AutoMigrate(table...)
}

func (d *DefaultGormDB) Unscoped() *gorm.DB {
	return d.getDB().Unscoped()
}

func (d *DefaultGormDB) FirstOrCreate(dest any, conds ...any) (tx *gorm.DB) {
	return d.getDB().FirstOrCreate(dest, conds...)
}

func (d *DefaultGormDB) First(dest any, conds ...any) (tx *gorm.DB) {
	return d.getDB().First(dest, conds...)
}

func (d *DefaultGormDB) Scopes(funcs ...func(*gorm.DB) *gorm.DB) (tx *gorm.DB) {
	return d.getDB().Scopes(funcs...)
}

func (d *DefaultGormDB) Clauses(conds ...clause.Expression) (tx *gorm.DB) {
	return d.getDB().Clauses(conds...)
}

func (d *DefaultGormDB) Raw(sql string, values ...any) (tx *gorm.DB) {
	return d.getDB().Raw(sql, values...)
}

func (d *DefaultGormDB) Exec(sql string, values ...any) (tx *gorm.DB) {
	return d.getDB().Exec(sql, values...)
}

func (d *DefaultGormDB) WithContext(ctx context.Context) *gorm.DB {
	return d.getDB().WithContext(ctx)
}
