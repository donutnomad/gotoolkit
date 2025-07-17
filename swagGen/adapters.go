package main

import (
	"fmt"

	parsers "github.com/donutnomad/gotoolkit/swagGen/parser"
)

// SwaggerGeneratorAdapter 适配器，让现有的 SwaggerGenerator 实现新接口
type SwaggerGeneratorAdapter struct {
	generator *SwaggerGenerator
}

// NewSwaggerGeneratorAdapter 创建 Swagger 生成器适配器
func NewSwaggerGeneratorAdapter(collection *InterfaceCollection) *SwaggerGeneratorAdapter {
	return &SwaggerGeneratorAdapter{
		generator: NewSwaggerGenerator(collection),
	}
}

// GenerateSwaggerComments 生成 Swagger 注释（适配器实现）
func (a *SwaggerGeneratorAdapter) GenerateSwaggerComments() (map[string]string, error) {
	if a.generator == nil || a.generator.collection == nil {
		return nil, NewGenerateError("Swagger 生成器未初始化", "生成器或接口集合为空", nil)
	}

	comments := a.generator.GenerateSwaggerComments()
	return comments, nil
}

// GenerateFileHeader 生成文件头部（适配器实现）
func (a *SwaggerGeneratorAdapter) GenerateFileHeader(packageName string) string {
	return a.generator.GenerateFileHeader(packageName)
}

// GenerateImports 生成导入声明（适配器实现）
func (a *SwaggerGeneratorAdapter) GenerateImports() string {
	return a.generator.GenerateImports()
}

// SetInterfaces 设置要生成的接口（适配器实现）
func (a *SwaggerGeneratorAdapter) SetInterfaces(collection *InterfaceCollection) {
	if a.generator == nil {
		a.generator = NewSwaggerGenerator(collection)
	} else {
		a.generator.collection = collection
	}
}

// GinGeneratorAdapter 适配器，让现有的 GinGenerator 实现新接口
type GinGeneratorAdapter struct {
	generator *GinGenerator
}

// NewGinGeneratorAdapter 创建 Gin 生成器适配器
func NewGinGeneratorAdapter(collection *InterfaceCollection) *GinGeneratorAdapter {
	return &GinGeneratorAdapter{
		generator: NewGinGenerator(collection),
	}
}

// GenerateGinCode 生成 Gin 绑定代码（适配器实现）
func (a *GinGeneratorAdapter) GenerateGinCode(comments map[string]string) (string, string, error) {
	if a.generator == nil || a.generator.collection == nil {
		return "", "", NewGenerateError("Gin 生成器未初始化", "生成器或接口集合为空", nil)
	}

	// 现有的 GinGenerator 没有分别返回两个部分，这里简化处理
	code := a.generator.GenerateComplete(comments)
	return code, "", nil
}

// GenerateComplete 生成完整代码（适配器实现）
func (a *GinGeneratorAdapter) GenerateComplete(comments map[string]string) (string, error) {
	if a.generator == nil || a.generator.collection == nil {
		return "", NewGenerateError("Gin 生成器未初始化", "生成器或接口集合为空", nil)
	}

	code := a.generator.GenerateComplete(comments)
	return code, nil
}

// SetInterfaces 设置要生成的接口（适配器实现）
func (a *GinGeneratorAdapter) SetInterfaces(collection *InterfaceCollection) {
	if a.generator == nil {
		a.generator = NewGinGenerator(collection)
	} else {
		a.generator.collection = collection
	}
}

// InterfaceParserAdapter 适配器，让现有的 InterfaceParser 实现新接口
type InterfaceParserAdapter struct {
	parser *InterfaceParser
	config *GenerationConfig
}

// NewInterfaceParserAdapter 创建接口解析器适配器
func NewInterfaceParserAdapter(importMgr *EnhancedImportManager) *InterfaceParserAdapter {
	return &InterfaceParserAdapter{
		parser: NewInterfaceParser(importMgr),
	}
}

// ParseFile 解析单个文件（适配器实现）
func (a *InterfaceParserAdapter) ParseFile(filename string) (*InterfaceCollection, error) {
	collection, err := a.parser.ParseFile(filename)
	if err != nil {
		return nil, fmt.Errorf("解析文件 %s 失败: %w", filename, err)
	}
	return collection, nil
}

// ParseDirectory 解析目录下的所有 Go 文件（适配器实现）
func (a *InterfaceParserAdapter) ParseDirectory(dirPath string) (*InterfaceCollection, error) {
	collection, err := a.parser.ParseDirectory(dirPath)
	if err != nil {
		return nil, fmt.Errorf("解析目录 %s 失败: %w", dirPath, err)
	}
	return collection, nil
}

// SetConfig 设置解析配置（适配器实现）
func (a *InterfaceParserAdapter) SetConfig(config *GenerationConfig) {
	a.config = config
	// 如果将来需要在解析器中使用配置，可以在这里实现
}

// 修复 newTagParser 函数中的 panic，提供安全版本
func newTagParserSafe() (*parsers.Parser, error) {
	parser := parsers.NewParser()
	err := parser.Register(
		parsers.Tag{},
		parsers.GET{},
		parsers.POST{},
		parsers.PUT{},
		parsers.PATCH{},
		parsers.DELETE{},

		parsers.Security{},
		parsers.Header{},
		parsers.MiddleWare{},

		parsers.JsonReq{},
		parsers.FormReq{},
		parsers.MimeReq{},

		parsers.JSON{},
		parsers.MIME{},

		// 参数注释标签
		parsers.FORM{},
		parsers.BODY{},
		parsers.PARAM{},
		parsers.QUERY{},

		parsers.Removed{},
		parsers.ExcludeFromBindAll{},
	)

	return parser, err
}