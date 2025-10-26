package gsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

var logLevel = logger.Warn

func Debug() {
	logLevel = logger.Info
}

var gsqlSourceDir string

func init() {
	_, file, _, _ := runtime.Caller(0)
	// compatible solution to get gorm source directory with various operating systems
	gsqlSourceDir = sourceDir(file)
}

func sourceDir(file string) string {
	dir := filepath.Dir(file)
	dir = filepath.Dir(dir)

	s := filepath.Dir(dir)
	if filepath.Base(s) != "github.com" {
		s = dir
	}
	return filepath.ToSlash(s) + "/"
}

func optional[T any](args []T, def T) T {
	if len(args) == 0 {
		return def
	}
	return args[0]
}

func getQuoteFunc() func(field string) string {
	return func(field string) string {
		var writer strings.Builder
		dialector.QuoteTo(&writer, field)
		return writer.String()
	}
}

type wrapperLogger struct {
	logger.Interface
	logger.Writer
	logger.Config
	traceErrStr, traceWarnStr, traceStr string
}

func NewWrapperLogger(l logger.Interface, level logger.LogLevel) logger.Interface {
	ret := &wrapperLogger{
		Interface: l,
		Config: logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  level,
			IgnoreRecordNotFoundError: false,
			Colorful:                  true,
		},
	}
	var (
		traceStr     = "%s\n[%.3fms] [rows:%v] %s"
		traceWarnStr = "%s %s\n[%.3fms] [rows:%v] %s"
		traceErrStr  = "%s %s\n[%.3fms] [rows:%v] %s"
	)

	if ret.Colorful {
		traceStr = logger.Green + "%s\n" + logger.Reset + logger.Yellow + "[%.3fms] " + logger.BlueBold + "[rows:%v]" + logger.Reset + " %s"
		traceWarnStr = logger.Green + "%s " + logger.Yellow + "%s\n" + logger.Reset + logger.RedBold + "[%.3fms] " + logger.Yellow + "[rows:%v]" + logger.Magenta + " %s" + logger.Reset
		traceErrStr = logger.RedBold + "%s " + logger.MagentaBold + "%s\n" + logger.Reset + logger.Yellow + "[%.3fms] " + logger.BlueBold + "[rows:%v]" + logger.Reset + " %s"
	}
	ret.traceErrStr = traceErrStr
	ret.traceWarnStr = traceWarnStr
	ret.traceStr = traceStr
	if v, ok := l.(logger.Writer); ok {
		ret.Writer = v
	}
	return ret
}

func (l *wrapperLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= logger.Error && (!errors.Is(err, logger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			l.Printf(l.traceErrStr, fileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Printf(l.traceErrStr, fileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= logger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			l.Printf(l.traceWarnStr, fileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Printf(l.traceWarnStr, fileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case l.LogLevel == logger.Info:
		sql, rows := fc()
		if rows == -1 {
			l.Printf(l.traceStr, fileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Printf(l.traceStr, fileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}

// fileWithLineNum return the file name and line number of the current file
func fileWithLineNum() string {
	pcs := [13]uintptr{}
	// the third caller usually from gorm internal
	len := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:len])
	for i := 0; i < len; i++ {
		// second return value is "more", not "ok"
		frame, _ := frames.Next()
		if (!strings.HasPrefix(frame.File, gsqlSourceDir) ||
			strings.HasSuffix(frame.File, "_test.go")) && !strings.HasSuffix(frame.File, ".gen.go") {
			return string(strconv.AppendInt(append([]byte(frame.File), ':'), int64(frame.Line), 10))
		}
	}
	return ""
}

func Scan(
	logLevel int,
	db *gorm.DB, dest any) *gorm.DB {
	session := &gorm.Session{
		Initialized: true,
	}
	if logLevel > 0 {
		session.Logger = db.Logger.LogMode(logger.LogLevel(logLevel))
	}
	tx := db.Session(session)
	config := *tx.Config
	currentLogger, newLogger := config.Logger, logger.Recorder.New()
	config.Logger = newLogger
	tx.Config = &config

	_ = tx.Statement.Parse(dest)

	if rows, err := tx.Rows(); err == nil {
		if rows.Next() {
			_ = ScanRows(tx, rows, dest)
		} else {
			tx.RowsAffected = 0
			dbErr := rows.Err()
			if dbErr == nil {
				dbErr = gorm.ErrRecordNotFound
			}
			_ = tx.AddError(dbErr)
		}
		_ = tx.AddError(rows.Close())
	}

	currentLogger.Trace(tx.Statement.Context, newLogger.BeginAt, func() (string, int64) {
		return newLogger.SQL, tx.RowsAffected
	}, tx.Error)
	tx.Logger = currentLogger
	return tx
}

func ScanRows(tx *gorm.DB, rows *sql.Rows, dest any) error {
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
func prepareValues(values []any, db *gorm.DB, columnTypes []*sql.ColumnType, columns []string) {
	if db.Statement.Schema != nil {
		for idx, name := range columns {
			if field := db.Statement.Schema.LookUpField(name); field != nil {
				values[idx] = reflect.New(reflect.PointerTo(field.FieldType)).Interface()
				continue
			}
			values[idx] = new(any)
		}
	} else if len(columnTypes) > 0 {
		for idx, columnType := range columnTypes {
			if columnType.ScanType() != nil {
				values[idx] = reflect.New(reflect.PointerTo(columnType.ScanType())).Interface()
			} else {
				values[idx] = new(any)
			}
		}
	} else {
		for idx := range columns {
			values[idx] = new(any)
		}
	}
}

func scanIntoMap(mapValue map[string]any, values []any, columns []string) {
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
		values      = make([]any, len(columns))
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
	case map[string]any, *map[string]any:
		if initialized || rows.Next() {
			columnTypes, _ := rows.ColumnTypes()
			prepareValues(values, db, columnTypes, columns)

			db.RowsAffected++
			db.AddError(rows.Scan(values...))

			mapValue, ok := dest.(map[string]any)
			if !ok {
				if v, ok := dest.(*map[string]any); ok {
					if *v == nil {
						*v = map[string]any{}
					}
					mapValue = *v
				}
			}
			scanIntoMap(mapValue, values, columns)
		}
	case *[]map[string]any:
		columnTypes, _ := rows.ColumnTypes()
		for initialized || rows.Next() {
			prepareValues(values, db, columnTypes, columns)

			initialized = false
			db.RowsAffected++
			db.AddError(rows.Scan(values...))

			mapValue := map[string]any{}
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
								var val any
								values[idx] = &val
							} else {
								var val any
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

func scanIntoStruct2(db *gorm.DB, rows gorm.Rows, reflectValue reflect.Value, values []any, fields []*schema.Field, joinFields [][]*schema.Field, embeddedFields [][]*schema.Field) {
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
	joinedNestedSchemaMap := make(map[string]any)
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
