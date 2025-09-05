package main

import (
	"fmt"
	"slices"
	"strings"

	parsers "github.com/donutnomad/gotoolkit/swagGen/parser"
	"github.com/samber/lo"
)

// NewSwaggerGenerator 创建 Swagger 生成器
func NewSwaggerGenerator(collection *InterfaceCollection) *SwaggerGenerator {
	parser, err := newTagParserSafe()
	if err != nil {
		panic(err)
	}
	return &SwaggerGenerator{
		collection: collection,
		tagsParser: parser,
	}
}

// GenerateSwaggerComments 生成 Swagger 注释
func (g *SwaggerGenerator) GenerateSwaggerComments() map[string]string {
	var out = make(map[string]string)
	// 为每个接口的每个方法生成注释
	for _, iface := range g.collection.Interfaces {
		for _, method := range iface.Methods {
			if method.Def.IsRemoved() {
				continue
			}
			methodComments := g.generateMethodComments(method, iface)
			// 使用接口名+方法名作为唯一键，避免不同接口中同名方法的冲突
			methodKey := fmt.Sprintf("%s.%s", iface.Name, method.Name)
			out[methodKey] = strings.Join(methodComments, "\n")
		}
	}
	return out
}

func mergeDefs[T any](ifaceDefs, methodDefs []parsers.Definition, f func(item parsers.Definition) (T, bool), post func([]T)) {
	var methodTags []T
	for _, item := range methodDefs {
		if v, ok := f(item); ok {
			methodTags = append(methodTags, v)
		}
	}
	if len(methodTags) == 0 {
		for _, item := range ifaceDefs {
			if v, ok := f(item); ok {
				methodTags = append(methodTags, v)
			}
		}
	}
	post(methodTags)
}

// generateMethodComments 生成单个方法的 Swagger 注释
func (g *SwaggerGenerator) generateMethodComments(method SwaggerMethod, iface SwaggerInterface) []string {
	var lines []string

	// 方法名注释
	lines = append(lines, fmt.Sprintf("// %s", method.Name))

	// Summary
	if method.Summary != "" {
		lines = append(lines, fmt.Sprintf("// @Summary %s", method.Summary))
	} else {
		lines = append(lines, fmt.Sprintf("// @Summary %s", method.Name))
	}

	// Description
	if method.Description != "" {
		for _, desc := range strings.Split(method.Description, "\n") {
			lines = append(lines, fmt.Sprintf("// @Description %s", desc))
		}
	}

	// MID
	for _, md := range CollectDef[*parsers.MiddleWare](method.Def) {
		for _, name := range md.Value {
			if idx := strings.Index(name, "_"); idx > 0 {
				prefix := name[:idx]
				if idx != len(name)-1 {
					lines = append(lines, fmt.Sprintf("// @Description %s: %s", prefix, name[idx+1:]))
				}
			}
		}
	}

	// Tags - 应用覆盖和排除逻辑
	mergeDefs[string](iface.CommonDef, method.Def, func(item parsers.Definition) (string, bool) {
		v, ok := item.(*parsers.Tag)
		if !ok {
			return "", false
		}
		return v.Value, true
	}, func(i []string) {
		if len(i) > 0 {
			lines = append(lines, fmt.Sprintf("// @Tags %s", strings.Join(i, ",")))
		}
	})

	// Accept (请求内容类型)
	if method.GetHTTPMethod() != "GET" {
		mergeDefs[string](iface.CommonDef, method.Def, func(item parsers.Definition) (string, bool) {
			return DefSlice{item}.GetAcceptType()
		}, func(i []string) {
			var ret = "json"
			if len(i) > 0 {
				ret = i[0]
			}
			lines = append(lines, fmt.Sprintf("// @Accept %s", ret))
		})
	}
	// Produce (响应内容类型)
	mergeDefs[string](iface.CommonDef, method.Def, func(item parsers.Definition) (string, bool) {
		return DefSlice{item}.GetContentType()
	}, func(i []string) {
		var ret = "json"
		if len(i) > 0 {
			ret = i[0]
		}
		lines = append(lines, fmt.Sprintf("// @Produce %s", ret))
	})

	// Security - 应用覆盖和排除逻辑
	mergeDefs[string](iface.CommonDef, method.Def, func(item parsers.Definition) (string, bool) {
		v, ok := item.(*parsers.Security)
		if !ok {
			return "", false
		}
		ok = false
		if len(v.Include) > 0 {
			if lo.Contains(v.Include, method.Name) {
				ok = true
			}
		} else if len(v.Exclude) > 0 {
			if !lo.Contains(v.Include, method.Name) {
				ok = true
			}
		} else {
			ok = true
		}
		return v.Value, ok
	}, func(i []string) {
		if len(i) > 0 {
			lines = append(lines, fmt.Sprintf("// @Security %s", strings.Join(i, ",")))
		}
	})

	// Parameters
	paramLines := g.generateParameterComments(method, method.Parameters, iface.CommonDef, method.Def)
	lines = append(lines, paramLines...)

	for _, md := range CollectDef[*parsers.Raw](method.Def) {
		lines = append(lines, fmt.Sprintf("// %s", md.Value))
	}

	// Success response
	successLine := g.generateSuccessComment(method.ResponseType)
	lines = append(lines, successLine)

	prefix := iface.CommonDef.GetPrefix()

	// Router
	for _, pathRouter := range method.GetPaths() {
		lines = append(lines, fmt.Sprintf("// @Router %s [%s]", prefix+pathRouter, strings.ToLower(method.GetHTTPMethod())))
	}

	return lines
}

// generateParameterComments 生成参数注释
func (g *SwaggerGenerator) generateParameterComments(method SwaggerMethod, parameters []Parameter, ifaceDef, def DefSlice) []string {
	var lines []string

	for i, param := range parameters {
		// 跳过 gin.Context 参数
		if param.Type.FullName == GinContextType || param.Type.TypeName == "Context" {
			continue
		}
		if param.Source == "path" {
		} else if param.Source == "header" {
		} else if i == len(parameters)-1 {
			// 默认的
			if method.GetHTTPMethod() == "GET" {
				param.Source = "query"
			} else if v, _ := slices.Concat(def, ifaceDef).GetAcceptType(); v == "json" {
				param.Source = "body"
			} else {
				param.Source = "formData"
			}
		}

		paramLine := g.generateParameterComment(param)
		lines = append(lines, paramLine)
	}

	// 需要生成header的注释
	// @Param        X-MyHeader	  header    string    true   	"MyHeader must be set for valid response"
	// @Param        X-API-VERSION    header    string    true   	"API version eg.: 1.0"
	var allSlice = slices.Concat(ifaceDef, def)
	var headerMap = make(map[string]*parsers.Header)
	var headerNames []string
	for _, param := range allSlice {
		if v, ok := param.(*parsers.Header); ok {
			_, exists := headerMap[v.Value]
			headerMap[v.Value] = v
			if !exists {
				headerNames = append(headerNames, v.Value)
			}
		}
	}
	for _, key := range headerNames {
		value := headerMap[key]
		headerLine := fmt.Sprintf("// @Param %s header string %s \"%s\"", key, lo.Ternary(value.Required, "true", "false"), lo.Ternary(len(value.Description) > 0, value.Description, key))
		lines = append(lines, headerLine)
	}

	return lines
}

// generateParameterComment 生成单个参数注释
func (g *SwaggerGenerator) generateParameterComment(param Parameter) string {
	// 格式: // @Param name in type required description

	// 获取参数类型
	paramType := g.getParameterType(param)
	// 确定是否必需
	required := lo.Ternary(param.Required, "true", "false")
	// 获取描述
	description := lo.Ternary(param.Comment == "", param.Name, param.Comment)

	// 特殊处理 body 参数
	if param.Source == "body" {
		return fmt.Sprintf("// @Param %s body %s %s \"%s\"", param.Name, param.Type.FullName, required, description)
	}

	n := lo.Ternary(len(param.PathName) > 0, param.PathName, param.Name)
	return fmt.Sprintf("// @Param %s %s %s %s \"%s\"", n, param.Source, paramType, required, description)
}

// generateSuccessComment 生成成功响应注释
func (g *SwaggerGenerator) generateSuccessComment(responseType TypeInfo) string {
	// 默认 200 响应
	if responseType.FullName == "" {
		return "// @Success 200 {string} string \"success\""
	}

	// 获取响应类型
	responseTypeStr := responseType.FullName
	if responseTypeStr == "" {
		responseTypeStr = "string"
	}

	return fmt.Sprintf("// @Success 200 {object} %s", responseTypeStr)
}

// getParameterType 获取参数类型字符串
func (g *SwaggerGenerator) getParameterType(param Parameter) string {
	typeInfo := param.Type

	// 处理基本类型
	switch typeInfo.GetSwaggerType() {
	case "string":
		return "string"
	case "integer":
		format := typeInfo.GetSwaggerFormat()
		if format != "" {
			return format
		}
		return "integer"
	case "number":
		format := typeInfo.GetSwaggerFormat()
		if format != "" {
			return format
		}
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		return "array"
	default:
		// 对象类型，返回完整类型名
		if typeInfo.FullName != "" {
			return typeInfo.FullName
		}
		return "string"
	}
}

// GenerateFileHeader 生成文件头部
func (g *SwaggerGenerator) GenerateFileHeader(packageName string) string {
	var lines []string

	lines = append(lines, "// Code generated by swagGen. DO NOT EDIT.")
	lines = append(lines, "//")
	lines = append(lines, "// This file contains Swagger documentation and Gin binding code.")
	lines = append(lines, "// Generated from interface definitions with Swagger annotations.")
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("package %s", packageName))
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

// GenerateImports 生成导入声明
func (g *SwaggerGenerator) GenerateImports() string {
	// 添加必要的导入
	g.collection.ImportMgr.AddImport("github.com/gin-gonic/gin")
	g.collection.ImportMgr.AddImport("strings")

	// 检查是否需要 cast 导入
	if g.needsCastImport() {
		g.collection.ImportMgr.AddImport("github.com/spf13/cast")
	}

	return g.collection.ImportMgr.GetImportDeclarations()
}

// needsCastImport 检查是否需要 cast 导入
func (g *SwaggerGenerator) needsCastImport() bool {
	for _, iface := range g.collection.Interfaces {
		for _, method := range iface.Methods {
			for _, param := range method.Parameters {
				if param.Source == "path" || param.Source == "query" {
					typeName := param.Type.TypeName
					switch typeName {
					case "int", "int8", "int16", "int32", "int64",
						"uint", "uint8", "uint16", "uint32", "uint64",
						"float32", "float64", "bool":
						return true
					}
				}
			}
		}
	}
	return false
}

// GenerateTypeReferences 生成类型引用
func (g *SwaggerGenerator) GenerateTypeReferences() string {
	return g.collection.ImportMgr.GetTypeReferences()
}
