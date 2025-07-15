package main

import (
	"fmt"
	"github.com/donutnomad/gotoolkit/internal/xast"
	"go/token"
)

// TypeInfo 表示类型信息
type TypeInfo struct {
	FullName    string     // types2.BaseResponse[string]
	Package     string     // go.com/pkg/v2/types
	Alias       string     // types2
	TypeName    string     // BaseResponse
	GenericArgs []TypeInfo // [string]
	IsGeneric   bool       // 是否是泛型
	IsSlice     bool       // 是否是切片
	IsPointer   bool       // 是否是指针
}

// Parameter 表示方法参数
type Parameter struct {
	Name     string   // 参数名
	PathName string   // 路径中的参数名
	Alias    string   // 别名
	Type     TypeInfo // 参数类型
	Source   string   // @PARAM, @FORM, @JSON, @BODY
	Required bool     // 是否必需
	Comment  string   // 参数注释
}

// SwaggerMethod 表示 Swagger 方法
type SwaggerMethod struct {
	Name         string      // 方法名
	HTTPMethod   string      // HTTP 方法 (GET, POST, PUT, DELETE)
	Paths        []string    // API 路径
	Parameters   []Parameter // 参数列表
	ResponseType TypeInfo    // 返回类型
	ContentType  string      // 请求内容类型
	AcceptType   string      // 接受的内容类型
	Comments     []string    // 方法注释
	Summary      string      // 摘要
	Description  string      // 描述
	Tags         []string    // 标签
	Security     string      // 安全
}

// SwaggerInterface 表示 Swagger 接口
type SwaggerInterface struct {
	Name        string               // 接口名
	Methods     []SwaggerMethod      // 方法列表
	PackagePath string               // 包路径
	Comments    []string             // 接口注释
	Imports     xast.ImportInfoSlice // 导入信息
}

func (w SwaggerInterface) GetWrapperName() string {
	n := fmt.Sprintf("%sWrap", w.Name)
	if n[0] == 'I' {
		n = n[1:]
	}
	return n
}

// InterfaceCollection 表示接口集合
type InterfaceCollection struct {
	Interfaces []SwaggerInterface     // 接口列表
	ImportMgr  *EnhancedImportManager // 导入管理器
}

// EnhancedImportManager 增强的导入管理器
type EnhancedImportManager struct {
	imports        map[string]*ImportInfo // 包路径 -> 导入信息
	aliasCounter   map[string]int         // 基础名称 -> 计数器
	aliasMapping   map[string]string      // 包路径 -> 别名
	typeReferences map[string][]string    // 包路径 -> 类型列表
	packagePath    string                 // 当前包路径
}

// ImportInfo 导入信息
type ImportInfo struct {
	Path          string // 包路径
	Alias         string // 别名
	OriginalAlias string // 原始别名（来自源码）
	Used          bool   // 是否被使用
	DirectlyUsed  bool   // 是否直接使用（而不仅仅是类型引用）
}

// AnnotationParser 注释解析器
type AnnotationParser struct {
	fileSet *token.FileSet
}

// InterfaceParser 接口解析器
type InterfaceParser struct {
	annotationParser *AnnotationParser
	importMgr        *EnhancedImportManager
}

// ReturnTypeParser 返回类型解析器
type ReturnTypeParser struct {
	importMgr *EnhancedImportManager
	imports   xast.ImportInfoSlice
}

// SwaggerGenerator Swagger 生成器
type SwaggerGenerator struct {
	collection *InterfaceCollection
}

// GinGenerator Gin 绑定代码生成器
type GinGenerator struct {
	collection *InterfaceCollection
}
