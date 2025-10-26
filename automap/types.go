package automap

import (
	"fmt"
	"go/ast"
	"go/token"
	"iter"
	"strings"

	"github.com/donutnomad/gotoolkit/internal/gormparse"
	"github.com/donutnomad/gotoolkit/internal/xast"
	"github.com/samber/lo"
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

func (t *TypeInfo) FieldIter() iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, item := range t.Fields {
			if item.IsEmbedded {
				for _, ef := range item.EmbeddedFields {
					if !yield(ef.Name) {
						return
					}
				}
			} else {
				if !yield(item.Name) {
					return
				}
			}
		}
	}
}
func (t *TypeInfo) FieldIter2() iter.Seq2[string, *FieldInfo] {
	return func(yield func(string, *FieldInfo) bool) {
		for _, item := range t.Fields {
			if item.IsEmbedded {
				for _, ef := range item.EmbeddedFields {
					// orm.Model  DefaultID
					// => Model.DefaultID
					if !yield(fmt.Sprintf("%s.%s", lo.LastOrEmpty(strings.Split(item.Type, ".")), ef.Name), &ef) {
						return
					}
				}
			} else {
				if !yield(item.Name, &item) {
					return
				}
			}
		}
	}
}

// NewTypeInfoFromName
// name: xxx.XXX 或者XXX
func NewTypeInfoFromName(fullName string) *TypeInfo {
	typeInfo := &TypeInfo{
		FullName: fullName,
	}
	parts := strings.Split(fullName, ".")
	if len(parts) == 2 {
		typeInfo.Name = parts[1]
		typeInfo.Package = parts[0]
	} else if len(parts) == 1 {
		typeInfo.Name = parts[0]
	}
	return typeInfo
}

// FieldInfo 字段信息
type FieldInfo struct {
	Name           string          // 字段名
	Type           string          // 字段类型
	GormTag        string          // GORM标签
	JsonTag        string          // GORM标签
	ColumnName     string          // 数据库列名
	IsJSONType     bool            // 是否为JSONType
	JSONFields     []JSONFieldInfo // JSON字段信息
	SourceType     string          // 来源类型（嵌入字段）
	IsEmbedded     bool            // 是否为嵌入字段
	ASTField       *ast.Field      // AST字段节点
	StructType     *ast.StructType // 字段是所属的结构体
	EmbeddedFields []FieldInfo
}

func (f *FieldInfo) GetFullType() string {
	return xast.GetFieldType(f.ASTField.Type, nil)
}

func (f *FieldInfo) GetJsonName() string {
	jsonTag := f.JsonTag
	// 如果没有json tag，或者tag值为空字符串，则返回空
	if jsonTag == "" {
		return ""
	}
	// 如果tag是 "-"，表示忽略此字段，返回空
	if jsonTag == "-" {
		return ""
	}
	parts := strings.SplitN(jsonTag, ",", 2)
	return parts[0]
}

func (f *FieldInfo) GetColumnName() string {
	return gormparse.ExtractColumnName(f.Name, "gorm:\""+f.GormTag+"\"")
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
	OneToOne   map[string]string      // A字段 -> B字段（一对一）
	OneToMany  map[string][]string    // A字段 -> B字段列表（一对多）
	JSONFields map[string]JSONMapping // JSON字段映射
}

// JSONMapping JSON字段映射
type JSONMapping struct {
	FieldName string            // B: 数据库的字段名
	SubFields map[string]string // A字段 -> B内部字段，Go的字段名,比如 (string) (len=19) "PlacementAgreements": (string) (len=35) "SupportResource.PlacementAgreements",
}

func NewJSONMapping(bColumnName string) *JSONMapping {
	return &JSONMapping{FieldName: bColumnName,
		SubFields: make(map[string]string),
	}
}

func (j *JSONMapping) SetAToB(aGoFieldName string, bPath string) {
	j.SubFields[aGoFieldName] = bPath
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
