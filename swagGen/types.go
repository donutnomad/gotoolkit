package main

import (
	"fmt"
	"github.com/donutnomad/gotoolkit/internal/xast"
	parsers "github.com/donutnomad/gotoolkit/swagGen/parser"
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
	Source   string   // path,header,query
	Required bool     // 是否必需
	Comment  string   // 参数注释
}

// SwaggerMethod 表示 Swagger 方法
type SwaggerMethod struct {
	Name         string      // 方法名
	Parameters   []Parameter // 参数列表
	ResponseType TypeInfo    // 返回类型

	Summary     string // 摘要
	Description string // 描述

	Def DefSlice
}

func CollectDef[T any](input DefSlice) []T {
	var ret []T
	for _, item := range input {
		if v, ok := item.(T); ok {
			ret = append(ret, v)
		}
	}
	return ret
}

type DefSlice []parsers.Definition

func (s DefSlice) IsRemoved() bool {
	for _, item := range s {
		switch item.(type) {
		case *parsers.Removed:
			return true
		default:
		}
	}
	return false
}

func (s DefSlice) GetAcceptType() (string, bool) {
	for _, item := range s {
		switch item.(type) {
		case *parsers.FormReq:
			return "x-www-form-urlencoded", true
		case *parsers.JsonReq:
			return "json", true
		default:
		}
	}
	return "json", false
}

func (s DefSlice) GetContentType() (string, bool) {
	for _, item := range s {
		switch v := item.(type) {
		case *parsers.JSON:
			return "json", true
		case *parsers.MIME:
			return v.Value, true
		default:
		}
	}
	return "json", false
}

func (s SwaggerMethod) GetPaths() []string {
	var ret []string
	for _, item := range s.Def {
		switch v := item.(type) {
		case *parsers.GET:
			ret = append(ret, v.Value)
		case *parsers.POST:
			ret = append(ret, v.Value)
		case *parsers.PUT:
			ret = append(ret, v.Value)
		case *parsers.DELETE:
			ret = append(ret, v.Value)
		case *parsers.PATCH:
			ret = append(ret, v.Value)
		default:
		}
	}
	return ret
}

func (s SwaggerMethod) GetHTTPMethod() string {
	for _, item := range s.Def {
		switch v := item.(type) {
		case *parsers.GET:
			return v.Name()
		case *parsers.POST:
			return v.Name()
		case *parsers.PUT:
			return v.Name()
		case *parsers.DELETE:
			return v.Name()
		case *parsers.PATCH:
			return v.Name()
		default:
		}
	}
	return "GET"
}

// CommonAnnotation 表示可应用于接口中所有方法的通用注释，支持排除特定方法。
type CommonAnnotation struct {
	Value   string   // 注释的值，例如 "Company" 或 "ApiKeyAuth"
	Exclude []string // 要从此注释中排除的方法名列表
}

// SwaggerInterface 表示 Swagger 接口
type SwaggerInterface struct {
	Name        string               // 接口名
	Methods     []SwaggerMethod      // 方法列表
	PackagePath string               // 包路径
	Comments    []string             // 接口注释
	Imports     xast.ImportInfoSlice // 导入信息
	CommonDef   DefSlice
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
	fileSet    *token.FileSet
	tagsParser *parsers.Parser
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
	tagsParser *parsers.Parser
}

// GinGenerator Gin 绑定代码生成器
type GinGenerator struct {
	collection *InterfaceCollection
}
