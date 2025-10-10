package gsql

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
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
		if v, ok := f.(field.BaseFields); ok {
			b.selects = append(b.selects, v...)
		} else {
			b.selects = append(b.selects, f)
		}
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
	tx := b.build(db)
	ret := Scan(tx, dest)
	//ret := b.build(db).Find(dest)
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

func Scan(db IDB, dest any) *gorm.DB {
	tx := db.Session(&gorm.Session{
		Initialized: true,
	})
	config := *tx.Config
	currentLogger, newLogger := config.Logger, logger.Recorder.New()
	config.Logger = newLogger
	tx.Config = &config

	if rows, err := tx.Rows(); err == nil {
		if rows.Next() {
			_ = ScanRows(tx, rows, dest)
		} else {
			tx.RowsAffected = 0
			_ = tx.AddError(rows.Err())
		}
		_ = tx.AddError(rows.Close())
	}

	NewWrapperLogger(currentLogger, logLevel).Trace(tx.Statement.Context, newLogger.BeginAt, func() (string, int64) {
		return newLogger.SQL, tx.RowsAffected
	}, tx.Error)
	tx.Logger = currentLogger
	return tx
}

func ScanRows(tx *gorm.DB, rows *sql.Rows, dest interface{}) error {
	if err := tx.Statement.Parse(dest); !errors.Is(err, schema.ErrUnsupportedDataType) {
		tx.AddError(err)
	}
	tx.Statement.Dest = dest
	tx.Statement.ReflectValue = reflect.ValueOf(dest)
	for tx.Statement.ReflectValue.Kind() == reflect.Ptr {
		elem := tx.Statement.ReflectValue.Elem()
		if !elem.IsValid() {
			elem = reflect.New(tx.Statement.ReflectValue.Type().Elem())
			tx.Statement.ReflectValue.Set(elem)
		}
		tx.Statement.ReflectValue = elem
	}
	Scan2(rows, tx, gorm.ScanInitialized)
	return tx.Error
}

// prepareValues prepare values slice
func prepareValues(values []interface{}, db *gorm.DB, columnTypes []*sql.ColumnType, columns []string) {
	if db.Statement.Schema != nil {
		for idx, name := range columns {
			if field := db.Statement.Schema.LookUpField(name); field != nil {
				values[idx] = reflect.New(reflect.PointerTo(field.FieldType)).Interface()
				continue
			}
			values[idx] = new(interface{})
		}
	} else if len(columnTypes) > 0 {
		for idx, columnType := range columnTypes {
			if columnType.ScanType() != nil {
				values[idx] = reflect.New(reflect.PointerTo(columnType.ScanType())).Interface()
			} else {
				values[idx] = new(interface{})
			}
		}
	} else {
		for idx := range columns {
			values[idx] = new(interface{})
		}
	}
}

func scanIntoMap(mapValue map[string]interface{}, values []interface{}, columns []string) {
	for idx, column := range columns {
		if reflectValue := reflect.Indirect(reflect.Indirect(reflect.ValueOf(values[idx]))); reflectValue.IsValid() {
			mapValue[column] = reflectValue.Interface()
			if valuer, ok := mapValue[column].(driver.Valuer); ok {
				mapValue[column], _ = valuer.Value()
			} else if b, ok := mapValue[column].(sql.RawBytes); ok {
				mapValue[column] = string(b)
			}
		} else {
			mapValue[column] = nil
		}
	}
}

var cacheStore = &sync.Map{}

// Scan2 scan rows into db statement
func Scan2(rows gorm.Rows, db *gorm.DB, mode gorm.ScanMode) {
	var (
		columns, _  = rows.Columns()
		values      = make([]interface{}, len(columns))
		initialized = mode&gorm.ScanInitialized != 0
		//update      = mode&gorm.ScanUpdate != 0
		//onConflictDonothing = mode&gorm.ScanOnConflictDoNothing != 0
	)

	if len(db.Statement.ColumnMapping) > 0 {
		for i, column := range columns {
			v, ok := db.Statement.ColumnMapping[column]
			if ok {
				columns[i] = v
			}
		}
	}

	db.RowsAffected = 0

	switch dest := db.Statement.Dest.(type) {
	case map[string]interface{}, *map[string]interface{}:
		if initialized || rows.Next() {
			columnTypes, _ := rows.ColumnTypes()
			prepareValues(values, db, columnTypes, columns)

			db.RowsAffected++
			db.AddError(rows.Scan(values...))

			mapValue, ok := dest.(map[string]interface{})
			if !ok {
				if v, ok := dest.(*map[string]interface{}); ok {
					if *v == nil {
						*v = map[string]interface{}{}
					}
					mapValue = *v
				}
			}
			scanIntoMap(mapValue, values, columns)
		}
	case *[]map[string]interface{}:
		columnTypes, _ := rows.ColumnTypes()
		for initialized || rows.Next() {
			prepareValues(values, db, columnTypes, columns)

			initialized = false
			db.RowsAffected++
			db.AddError(rows.Scan(values...))

			mapValue := map[string]interface{}{}
			scanIntoMap(mapValue, values, columns)
			*dest = append(*dest, mapValue)
		}
	case *int, *int8, *int16, *int32, *int64,
		*uint, *uint8, *uint16, *uint32, *uint64, *uintptr,
		*float32, *float64,
		*bool, *string, *time.Time,
		*sql.NullInt32, *sql.NullInt64, *sql.NullFloat64,
		*sql.NullBool, *sql.NullString, *sql.NullTime:
		for initialized || rows.Next() {
			initialized = false
			db.RowsAffected++
			db.AddError(rows.Scan(dest))
		}
	default:
		var (
			fields         = make([]*schema.Field, len(columns))
			joinFields     [][]*schema.Field
			embeddedFields [][]*schema.Field // 用于存储嵌入字段信息
			sch            = db.Statement.Schema
			reflectValue   = db.Statement.ReflectValue
		)

		if reflectValue.Kind() == reflect.Interface {
			reflectValue = reflectValue.Elem()
		}

		reflectValueType := reflectValue.Type()
		switch reflectValueType.Kind() {
		case reflect.Array, reflect.Slice:
			reflectValueType = reflectValueType.Elem()
		}
		isPtr := reflectValueType.Kind() == reflect.Ptr
		if isPtr {
			reflectValueType = reflectValueType.Elem()
		}

		if sch != nil {
			if reflectValueType != sch.ModelType && reflectValueType.Kind() == reflect.Struct {
				sch, _ = schema.Parse(db.Statement.Dest, cacheStore, db.NamingStrategy)
			}

			if len(columns) == 1 {
				// Is Pluck
				if _, ok := reflect.New(reflectValueType).Interface().(sql.Scanner); (reflectValueType != sch.ModelType && ok) || // is scanner
					reflectValueType.Kind() != reflect.Struct || // is not struct
					sch.ModelType.ConvertibleTo(schema.TimeReflectType) { // is time
					sch = nil
				}
			}

			// Not Pluck
			if sch != nil {
				// 首先解析 gsql:"embedded:prefix" 标签的字段
				embeddedFieldsMap := make(map[string]*schema.Field) // prefix -> field
				embeddedSchemas := make(map[string]*schema.Schema)  // prefix -> schema

				for _, field := range sch.Fields {
					if field.StructField.Tag.Get("gsql") != "" {
						tagValue := field.StructField.Tag.Get("gsql")
						if strings.HasPrefix(tagValue, "embedded:") {
							prefix := strings.TrimPrefix(tagValue, "embedded:")
							embeddedFieldsMap[prefix] = field

							// 解析嵌入字段的 schema
							embeddedType := field.StructField.Type
							if embeddedType.Kind() == reflect.Ptr {
								embeddedType = embeddedType.Elem()
							}
							if embeddedType.Kind() == reflect.Struct {
								embeddedSchema, _ := schema.Parse(reflect.New(embeddedType).Interface(), cacheStore, db.NamingStrategy)
								embeddedSchemas[prefix] = embeddedSchema
							}
						}
					}
				}

				matchedFieldCount := make(map[string]int, len(columns))
				for idx, column := range columns {
					if field := sch.LookUpField(column); field != nil && field.Readable {
						fields[idx] = field
						if count, ok := matchedFieldCount[column]; ok {
							// handle duplicate fields
							for _, selectField := range sch.Fields {
								if selectField.DBName == column && selectField.Readable {
									if count == 0 {
										matchedFieldCount[column]++
										fields[idx] = selectField
										break
									}
									count--
								}
							}
						} else {
							matchedFieldCount[column] = 1
						}
					} else {
						// 检查是否匹配嵌入字段的前缀
						matched := false
						for prefix, embeddedField := range embeddedFieldsMap {
							if strings.HasPrefix(column, prefix) {
								// 找到匹配的前缀，去掉前缀后在嵌入 schema 中查找
								columnWithoutPrefix := strings.TrimPrefix(column, prefix)
								if embeddedSchema, ok := embeddedSchemas[prefix]; ok {
									if field := embeddedSchema.LookUpField(columnWithoutPrefix); field != nil && field.Readable {
										fields[idx] = field

										if len(embeddedFields) == 0 {
											embeddedFields = make([][]*schema.Field, len(columns))
										}
										// embeddedFields[idx][0] 是嵌入字段本身
										// embeddedFields[idx][1] 是嵌入字段内的具体字段
										embeddedFields[idx] = []*schema.Field{embeddedField, field}
										matched = true
										break
									}
								}
							}
						}

						if !matched {
							if names := utils.SplitNestedRelationName(column); len(names) > 1 { // has nested relation
								aliasName := utils.JoinNestedRelationNames(names[0 : len(names)-1])
								for _, join := range db.Statement.Joins {
									if join.Alias == aliasName {
										names = append(strings.Split(join.Name, "."), names[len(names)-1])
										break
									}
								}

								if rel, ok := sch.Relationships.Relations[names[0]]; ok {
									subNameCount := len(names)
									// nested relation fields
									relFields := make([]*schema.Field, 0, subNameCount-1)
									relFields = append(relFields, rel.Field)
									for _, name := range names[1 : subNameCount-1] {
										rel = rel.FieldSchema.Relationships.Relations[name]
										relFields = append(relFields, rel.Field)
									}
									// latest name is raw dbname
									dbName := names[subNameCount-1]
									if field := rel.FieldSchema.LookUpField(dbName); field != nil && field.Readable {
										fields[idx] = field

										if len(joinFields) == 0 {
											joinFields = make([][]*schema.Field, len(columns))
										}
										relFields = append(relFields, field)
										joinFields[idx] = relFields
										continue
									}
								}
								var val interface{}
								values[idx] = &val
							} else {
								var val interface{}
								values[idx] = &val
							}
						}
					}
				}
			}
		}

		switch reflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			var (
				elem        reflect.Value
				isArrayKind = reflectValue.Kind() == reflect.Array
			)

			//if !update || reflectValue.Len() == 0 {
			if reflectValue.Len() == 0 {
				//update = false
				if isArrayKind {
					db.Statement.ReflectValue.Set(reflect.Zero(reflectValue.Type()))
				} else {
					// if the slice cap is externally initialized, the externally initialized slice is directly used here
					if reflectValue.Cap() == 0 {
						db.Statement.ReflectValue.Set(reflect.MakeSlice(reflectValue.Type(), 0, 20))
					} else {
						reflectValue.SetLen(0)
						db.Statement.ReflectValue.Set(reflectValue)
					}
				}
			}

			for initialized || rows.Next() {
				//BEGIN:
				initialized = false

				//if update {
				//	if int(db.RowsAffected) >= reflectValue.Len() {
				//		return
				//	}
				//	elem = reflectValue.Index(int(db.RowsAffected))
				//if onConflictDonothing {
				//	for _, field := range fields {
				//		if _, ok := field.ValueOf(db.Statement.Context, elem); !ok {
				//			db.RowsAffected++
				//			goto BEGIN
				//		}
				//	}
				//}
				//} else {
				elem = reflect.New(reflectValueType)
				//}

				scanIntoStruct2(db, rows, elem, values, fields, joinFields, embeddedFields)

				//if !update {
				if !isPtr {
					elem = elem.Elem()
				}
				if isArrayKind {
					if reflectValue.Len() >= int(db.RowsAffected) {
						reflectValue.Index(int(db.RowsAffected - 1)).Set(elem)
					}
				} else {
					reflectValue = reflect.Append(reflectValue, elem)
				}
				//}
			}

			//if !update {
			db.Statement.ReflectValue.Set(reflectValue)
			//}
		case reflect.Struct, reflect.Ptr:
			if initialized || rows.Next() {
				if mode == gorm.ScanInitialized && reflectValue.Kind() == reflect.Struct {
					db.Statement.ReflectValue.Set(reflect.Zero(reflectValue.Type()))
				}
				scanIntoStruct2(db, rows, reflectValue, values, fields, joinFields, embeddedFields)
			}
		default:
			db.AddError(rows.Scan(dest))
		}
	}

	if err := rows.Err(); err != nil && err != db.Error {
		db.AddError(err)
	}

	if db.RowsAffected == 0 && db.Statement.RaiseErrorOnNotFound && db.Error == nil {
		db.AddError(gorm.ErrRecordNotFound)
	}
}

func scanIntoStruct2(db *gorm.DB, rows gorm.Rows, reflectValue reflect.Value, values []interface{}, fields []*schema.Field, joinFields [][]*schema.Field, embeddedFields [][]*schema.Field) {
	for idx, field := range fields {
		if field != nil {
			values[idx] = field.NewValuePool.Get()
		} else if len(fields) == 1 {
			if reflectValue.CanAddr() {
				values[idx] = reflectValue.Addr().Interface()
			} else {
				values[idx] = reflectValue.Interface()
			}
		}
	}

	db.RowsAffected++
	db.AddError(rows.Scan(values...))
	joinedNestedSchemaMap := make(map[string]interface{})
	embeddedInitialized := make(map[string]bool) // 记录哪些嵌入字段已经初始化

	for idx, field := range fields {
		if field == nil {
			continue
		}

		// 检查是否是嵌入字段
		if len(embeddedFields) > 0 && len(embeddedFields[idx]) > 0 {
			embeddedField := embeddedFields[idx][0] // 嵌入字段本身
			targetField := embeddedFields[idx][1]   // 嵌入字段内的具体字段

			// 获取或初始化嵌入字段
			embeddedValue := embeddedField.ReflectValueOf(db.Statement.Context, reflectValue)
			if !embeddedInitialized[embeddedField.Name] {
				// 如果是指针类型且为 nil，需要初始化
				if embeddedValue.Kind() == reflect.Ptr && embeddedValue.IsNil() {
					embeddedValue.Set(reflect.New(embeddedValue.Type().Elem()))
				}
				// 如果是零值结构体，也需要确保字段可访问
				if embeddedValue.Kind() == reflect.Struct {
					// 结构体类型不需要特殊初始化
				}
				embeddedInitialized[embeddedField.Name] = true
			}

			// 设置值到嵌入字段的具体字段
			if embeddedValue.Kind() == reflect.Ptr {
				db.AddError(targetField.Set(db.Statement.Context, embeddedValue.Elem(), values[idx]))
			} else {
				db.AddError(targetField.Set(db.Statement.Context, embeddedValue, values[idx]))
			}
		} else if len(joinFields) == 0 || len(joinFields[idx]) == 0 {
			db.AddError(field.Set(db.Statement.Context, reflectValue, values[idx]))
		} else { // joinFields count is larger than 2 when using join
			var isNilPtrValue bool
			var relValue reflect.Value
			// does not contain raw dbname
			nestedJoinSchemas := joinFields[idx][:len(joinFields[idx])-1]
			// current reflect value
			currentReflectValue := reflectValue
			fullRels := make([]string, 0, len(nestedJoinSchemas))
			for _, joinSchema := range nestedJoinSchemas {
				fullRels = append(fullRels, joinSchema.Name)
				relValue = joinSchema.ReflectValueOf(db.Statement.Context, currentReflectValue)
				if relValue.Kind() == reflect.Ptr {
					fullRelsName := utils.JoinNestedRelationNames(fullRels)
					// same nested structure
					if _, ok := joinedNestedSchemaMap[fullRelsName]; !ok {
						if value := reflect.ValueOf(values[idx]).Elem(); value.Kind() == reflect.Ptr && value.IsNil() {
							isNilPtrValue = true
							break
						}

						relValue.Set(reflect.New(relValue.Type().Elem()))
						joinedNestedSchemaMap[fullRelsName] = nil
					}
				}
				currentReflectValue = relValue
			}

			if !isNilPtrValue { // ignore if value is nil
				f := joinFields[idx][len(joinFields[idx])-1]
				db.AddError(f.Set(db.Statement.Context, relValue, values[idx]))
			}
		}

		// release data to pool
		field.NewValuePool.Put(values[idx])
	}
}
