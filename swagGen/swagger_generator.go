package main

import (
	"fmt"
	"strings"
)

// NewSwaggerGenerator 创建 Swagger 生成器
func NewSwaggerGenerator(collection *InterfaceCollection) *SwaggerGenerator {
	return &SwaggerGenerator{
		collection: collection,
	}
}

// GenerateSwaggerComments 生成 Swagger 注释
func (g *SwaggerGenerator) GenerateSwaggerComments() string {
	var lines []string

	// 为每个接口的每个方法生成注释
	for _, iface := range g.collection.Interfaces {
		for _, method := range iface.Methods {
			methodComments := g.generateMethodComments(method, iface.Name)
			lines = append(lines, methodComments...)
			lines = append(lines, "") // 空行分隔
		}
	}

	return strings.Join(lines, "\n")
}

// generateMethodComments 生成单个方法的 Swagger 注释
func (g *SwaggerGenerator) generateMethodComments(method SwaggerMethod, interfaceName string) []string {
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
		lines = append(lines, fmt.Sprintf("// @Description %s", method.Description))
	}

	// Tags - 使用接口名作为标签
	tags := method.Tags
	if len(tags) == 0 {
		tags = []string{interfaceName}
	}
	lines = append(lines, fmt.Sprintf("// @Tags %s", strings.Join(tags, ",")))

	// Accept (请求内容类型)
	lines = append(lines, fmt.Sprintf("// @Accept %s", g.getAcceptType(method.ContentType)))

	// Produce (响应内容类型)
	lines = append(lines, fmt.Sprintf("// @Produce %s", g.getProduceType(method.AcceptType)))

	// Security (如果需要认证)
	lines = append(lines, "// @Security ApiKeyAuth")

	// Parameters
	paramLines := g.generateParameterComments(method.Parameters)
	lines = append(lines, paramLines...)

	// Success response
	successLine := g.generateSuccessComment(method.ResponseType)
	lines = append(lines, successLine)

	// Router
	routerLine := fmt.Sprintf("// @Router %s [%s]", method.Path, strings.ToLower(method.HTTPMethod))
	lines = append(lines, routerLine)

	return lines
}

// generateParameterComments 生成参数注释
func (g *SwaggerGenerator) generateParameterComments(parameters []Parameter) []string {
	var lines []string

	for _, param := range parameters {
		// 跳过 gin.Context 参数
		if param.Type.FullName == "*gin.Context" || param.Type.TypeName == "Context" {
			continue
		}

		paramLine := g.generateParameterComment(param)
		lines = append(lines, paramLine)
	}

	return lines
}

// generateParameterComment 生成单个参数注释
func (g *SwaggerGenerator) generateParameterComment(param Parameter) string {
	// 格式: // @Param name in type required description

	// 获取参数类型
	paramType := g.getParameterType(param)

	// 确定是否必需
	required := "true"
	if !param.Required {
		required = "false"
	}

	// 获取描述
	description := param.Comment
	if description == "" {
		description = param.Name
	}

	// 特殊处理 body 参数
	if param.Source == "body" {
		return fmt.Sprintf("// @Param %s body %s %s \"%s\"",
			param.Name, param.Type.FullName, required, description)
	}

	return fmt.Sprintf("// @Param %s %s %s %s \"%s\"",
		param.Name, param.Source, paramType, required, description)
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

// getAcceptType 获取 Accept 类型
func (g *SwaggerGenerator) getAcceptType(contentType string) string {
	switch contentType {
	case "application/json":
		return "json"
	case "application/x-www-form-urlencoded":
		return "x-www-form-urlencoded"
	case "multipart/form-data":
		return "multipart/form-data"
	case "application/xml":
		return "xml"
	case "text/plain":
		return "plain"
	default:
		// 自定义 MIME 类型直接返回
		return contentType
	}
}

// getProduceType 获取 Produce 类型
func (g *SwaggerGenerator) getProduceType(acceptType string) string {
	switch acceptType {
	case "application/json":
		return "json"
	case "application/xml":
		return "xml"
	case "text/plain":
		return "plain"
	case "text/html":
		return "html"
	default:
		return "json" // 默认 JSON
	}
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

// GenerateComplete 生成完整的文件内容
func (g *SwaggerGenerator) GenerateComplete(packageName string) string {
	var parts []string

	// 文件头部
	parts = append(parts, g.GenerateFileHeader(packageName))

	// 导入声明
	imports := g.GenerateImports()
	if imports != "" {
		parts = append(parts, imports)
		parts = append(parts, "")
	}

	// 类型引用
	typeRefs := g.GenerateTypeReferences()
	if typeRefs != "" {
		parts = append(parts, typeRefs)
		parts = append(parts, "")
	}

	// Swagger 注释
	swaggerComments := g.GenerateSwaggerComments()
	if swaggerComments != "" {
		parts = append(parts, swaggerComments)
	}

	return strings.Join(parts, "\n")
}
