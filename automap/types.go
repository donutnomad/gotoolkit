package automap

import (
	"go/ast"
	"go/token"
)

// FuncSignature 函数签名信息
type FuncSignature struct {
	PackageName string    // 包名
	Receiver    string    // 接收者类型（如"X"）
	FuncName    string    // 函数名
	InputType   TypeInfo  // 输入类型A
	OutputType  TypeInfo  // 输出类型B
	HasError    bool      // 是否返回error
	Pos         token.Pos // 位置信息
}

// TypeInfo 类型信息
type TypeInfo struct {
	Name      string       // 类型名（如"A"）
	Package   string       // 包名（如""表示当前包）
	FullName  string       // 完整类型名（如"package.A"）
	FilePath  string       // 定义文件路径
	Fields    []FieldInfo  // 字段列表
	IsPointer bool         // 是否为指针类型
	Methods   []MethodInfo // 方法列表
}

// FieldInfo 字段信息
type FieldInfo struct {
	Name       string          // 字段名
	Type       string          // 字段类型
	GormTag    string          // GORM标签
	ColumnName string          // 数据库列名
	IsJSONType bool            // 是否为JSONType
	JSONFields []JSONFieldInfo // JSON字段信息
	SourceType string          // 来源类型（嵌入字段）
	IsEmbedded bool            // 是否为嵌入字段
	ASTField   *ast.Field      // AST字段节点
	StructType *ast.StructType // 字段是所属的结构体
}

// JSONFieldInfo JSON字段信息
type JSONFieldInfo struct {
	Name string // JSON字段名
	Type string // JSON字段类型
	Tag  string // JSON标签
}

// MethodInfo 方法信息
type MethodInfo struct {
	Name       string     // 方法名
	Params     []TypeInfo // 参数列表
	Returns    []TypeInfo // 返回值列表
	IsExported bool       // 是否导出
}

// MappingRelation 映射关系
type MappingRelation struct {
	AField     string   // A类型字段名
	BFields    []string // B类型对应字段名列表
	IsJSONType bool     // 是否为JSONType映射
	JSONField  string   // JSON字段名（如果IsJSONType为true）
	Condition  string   // 映射条件（如果有）
	Order      int      // 赋值顺序（按照在函数中出现的顺序）
}

// FieldMapping 字段映射
type FieldMapping struct {
	OneToOne             map[string]string      // A字段 -> B字段（一对一）
	OneToMany            map[string][]string    // A字段 -> B字段列表（一对多）
	ManyToOne            map[string][]string    // B字段 -> A字段列表（多对一）
	JSONFields           map[string]JSONMapping // JSON字段映射
	OrderedRelations     []MappingRelation      // 按顺序排列的映射关系（非JSON）
	OrderedJSONRelations []MappingRelation      // 按顺序排列的JSON映射关系
}

// JSONMapping JSON字段映射
type JSONMapping struct {
	FieldName string            // JSON字段名
	SubFields map[string]string // A字段 -> JSON子字段
}

// ParseResult 解析结果
type ParseResult struct {
	FuncSignature    FuncSignature
	AType            TypeInfo
	BType            TypeInfo
	FieldMapping     FieldMapping
	HasExportPatch   bool
	GeneratedCode    string
	MappingRelations []MappingRelation
}

// MappingType 映射类型枚举
type MappingType int

const (
	MappingOneToOne MappingType = iota
	MappingOneToMany
	MappingManyToOne
	MappingJSON
)

// Error 自定义错误类型
type Error struct {
	Msg    string
	Pos    token.Pos
	Detail string
}

func (e *Error) Error() string {
	if e.Detail != "" {
		return e.Msg + ": " + e.Detail
	}
	return e.Msg
}

// NewError 创建新错误
func NewError(msg string, pos token.Pos, detail string) *Error {
	return &Error{
		Msg:    msg,
		Pos:    pos,
		Detail: detail,
	}
}
