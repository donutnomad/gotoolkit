package field

import (
	"github.com/samber/mo"
	"gorm.io/gorm/clause"
)

type Expression = clause.Expression

type IField interface {
	// ToColumn 转换为clause.Column对象，只有非expr模式才支持导出
	ToColumn() clause.Column
	// ToExpr 转换为表达式
	ToExpr() Expression
	// Name 返回字段名称
	// 对于expr，返回别名
	// 对于普通字段，有别名的返回别名，否则返回真实名字
	Name() string
	// IsExpr 是否是一个表达式字段
	IsExpr() bool
	// As 创建一个别名字段
	As(alias string) IField
}

type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

type INumber[T any] interface {
	IField
	Gt(value T) Expression
	GtOpt(value mo.Option[T]) Expression
	GtField(other INumber[T]) Expression
	Gte(value T) Expression
	GteOpt(value mo.Option[T]) Expression
	GteField(other INumber[T]) Expression
	Lt(value T) Expression
	LtOpt(value mo.Option[T]) Expression
	LtField(other INumber[T]) Expression
	Lte(value T) Expression
	LteOpt(value mo.Option[T]) Expression
	LteField(other INumber[T]) Expression
}

type IPointer interface {
	IField
	NotNil() Expression
	IsNil() Expression
}

type IString[T any] interface {
	IField
	NotLike(value T) Expression
	NotLikeOpt(value mo.Option[T]) Expression
	Like(value T) Expression
	LikeOpt(value mo.Option[T]) Expression
	Contains(value T) Expression
	ContainsOpt(value mo.Option[T]) Expression
	HasPrefix(value T) Expression
	HasPrefixOpt(value mo.Option[T]) Expression
	HasSuffix(value T) Expression
	HasSuffixOpt(value mo.Option[T]) Expression
}

type IComparable[T any] interface {
	IField
	Eq(value T) Expression
	EqOpt(value mo.Option[T]) Expression
	EqF(other IComparable[T]) Expression
	Not(value T) Expression
	NotOpt(value mo.Option[T]) Expression
	NotField(other IComparable[T]) Expression
	In(values ...T) Expression
	NotIn(values ...T) Expression
}

type IFieldType[T any] interface {
	FieldType() T
}

type Pattern[T any] struct {
	Base
	comparableImpl[T]
	patternImpl[T]
	pointerImpl
}

func NewPattern[T any](tableName, name string) Pattern[T] {
	b := NewBase(tableName, name)
	return Pattern[T]{
		Base:           *b,
		comparableImpl: comparableImpl[T]{IField: b},
		patternImpl:    patternImpl[T]{IField: b},
		pointerImpl:    pointerImpl{IField: b},
	}
}

func NewPatternWith[T any](b Base) Pattern[T] {
	return Pattern[T]{
		Base:           b,
		comparableImpl: comparableImpl[T]{IField: b},
		patternImpl:    patternImpl[T]{IField: b},
		pointerImpl:    pointerImpl{IField: b},
	}
}

func (f Pattern[T]) WithTable(tableName interface{ TableName() string }, fieldName ...string) Pattern[T] {
	var name = f.Base.columnName
	if len(fieldName) > 0 {
		name = fieldName[0]
	}
	return NewPattern[T](tableName.TableName(), name)
}

func (f Pattern[T]) WithName(name string) Pattern[T] {
	return NewPattern[T](f.Base.tableName, name)
}

func (f Pattern[T]) FieldType() T {
	var def T
	return def
}

type Comparable[T any] struct {
	Base
	comparableImpl[T]
	pointerImpl
}

func NewComparable[T any](tableName, name string) Comparable[T] {
	b := NewBase(tableName, name)
	return Comparable[T]{
		Base:           *b,
		comparableImpl: comparableImpl[T]{IField: b},
		pointerImpl:    pointerImpl{IField: b},
	}
}

func NewComparableWith[T any](b Base) Comparable[T] {
	return Comparable[T]{
		Base:           b,
		comparableImpl: comparableImpl[T]{IField: b},
		pointerImpl:    pointerImpl{IField: b},
	}
}

func (f Comparable[T]) FieldType() T {
	var def T
	return def
}

func (f Comparable[T]) WithTable(tableName interface{ TableName() string }, fieldNames ...string) Comparable[T] {
	return NewComparable[T](tableName.TableName(), optional(fieldNames, f.Base.Name()))
}

func (f Comparable[T]) WithName(fieldName string) Comparable[T] {
	return NewComparable[T](f.Base.tableName, fieldName)
}

// TODO: 缺少一个BETWEEN操作符
// TODO: 增加一个Blob类型(支持比较 + LIKE(字符串操作))
// TODO: 增加一个JSON类型（不支持比较)
// TODO: 增加一个空间类型（不支持比较)

// | 数据类型类别       | 具体数据类型 (示例)                                | `=`    | `!=` (`<>`) | `>` `<` `>=` `<=` | `BETWEEN` (`AND`) | `LIKE` ( `%` `_` ) | `IN` (`()`) | `IS NULL` (`IS NOT NULL`) | 备注                                                              |
//| :----------------- | :------------------------------------------------- | :----- | :---------- | :---------------- | :---------------- | :----------------- | :---------- | :------------------------ | :---------------------------------------------------------------- |
//| **数值类型**       | `TINYINT`, `INT`, `BIGINT`, `DECIMAL`, `FLOAT`   | ✅      | ✅          | ✅                | ✅                | ❌ (通常不适用)    | ✅          | ✅                        | 对数字进行数学比较。对 `LIKE` 会隐式转换为字符串。             |
//| **字符串类型**     | `CHAR`, `VARCHAR`, `TEXT`, `LONGTEXT`, `ENUM`, `SET` | ✅      | ✅          | ✅                | ✅                | ✅                 | ✅          | ✅                        | 字典序比较，受字符集和排序规则影响。`SET` 和 `ENUM` 值视为字符串。 |
//| **日期时间类型**   | `DATE`, `TIME`, `DATETIME`, `TIMESTAMP`, `YEAR` | ✅      | ✅          | ✅                | ✅                | ❌ (通常不适用)    | ✅          | ✅                        | 按时间顺序比较。对 `LIKE` 会隐式转换为字符串。                   |
//| **二进制字符串类型** | `BINARY`, `VARBINARY`, `BLOB`, `LONGBLOB`          | ✅      | ✅          | ✅                | (较少用)          | ✅                 | ✅          | ✅                        | 按字节二进制值比较（最严格）。`BETWEEN` 虽可用但意义不大。         |
//| **布尔类型**       | `BOOLEAN` (实际是 `TINYINT(1)`)                   | ✅      | ✅          | ✅                | ✅                | ❌                 | ✅          | ✅                        | `TRUE`=1, `FALSE`=0。`BETWEEN FALSE AND TRUE`等同于`IS NOT NULL`。 |
//| **JSON 类型**      | `JSON`                                             | ❌ (特定场景) | ❌ (特定场景) | ❌                | ❌                | ❌                 | ❌          | ✅                        | 通常需`JSON_EXTRACT()`提取值再比较。JSON文档本身不能简单使用这些操作符进行大小/范围比较。 |
//| **空间数据类型**   | `GEOMETRY`, `POINT`, `LINESTRING`, `POLYGON`     | ❌      | ❌          | ❌                | ❌                | ❌                 | ❌          | ✅                        | 使用专门的空间函数进行操作，无常规比较。                       |

// 不适合或不常见使用 BETWEEN 的类型（即使可能通过隐式转换工作）：
//
//LOB 类型 (Large Object Types) 如 BLOB, MEDIUMBLOB, LONGBLOB：
//这些是存储二进制大对象的类型，虽然它们的二进制内容可以进行字节级别的比较 (> 和 <)，但使用 BETWEEN 来定义一个有意义的二进制大对象范围是极不常见且几乎没有实际意义的。你很少会去查询一个 BLOB 是否“在”另一个 BLOB 的二进制内容“之间”。如果你需要，通常是对其元数据（如大小、创建日期）或其他属性进行比较。
//
//空间数据类型 (Spatial Data Types)：
//如 GEOMETY, POINT, LINESTRING, POLYGON 等。这些类型表示地理或几何信息，它们没有一个简单的“大于”或“小于”的概念，也没有一个自然的线性范围可以由 BETWEEN 来定义。你通常会使用专门的空间函数（如 ST_Contains(), ST_Intersects(), ST_Within()）进行空间关系判断。
