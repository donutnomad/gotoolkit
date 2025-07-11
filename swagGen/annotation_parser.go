package main

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

// NewAnnotationParser 创建注释解析器
func NewAnnotationParser(fileSet *token.FileSet) *AnnotationParser {
	return &AnnotationParser{
		fileSet: fileSet,
	}
}

// ParseMethodAnnotations 解析方法注释
func (p *AnnotationParser) ParseMethodAnnotations(method *ast.FuncDecl) (*SwaggerMethod, error) {
	if method.Doc == nil {
		return nil, nil
	}

	swaggerMethod := &SwaggerMethod{
		Name:        method.Name.Name,
		HTTPMethod:  "POST",                              // 默认值
		ContentType: "application/x-www-form-urlencoded", // 默认值
		AcceptType:  "application/json",                  // 默认值
		Tags:        []string{},                          // 初始化为空切片
	}

	var commentLines []string
	var summaryLines []string
	var descriptionLines []string

	// 解析所有注释行
	for _, comment := range method.Doc.List {
		line := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		commentLines = append(commentLines, line)

		// 解析特殊注释
		if strings.HasPrefix(line, "@") {
			p.parseSwaggerAnnotation(line, swaggerMethod)
		} else if line != "" {
			// 第一行非空注释作为 Summary
			if len(summaryLines) == 0 {
				summaryLines = append(summaryLines, line)
			} else {
				// 后续注释作为 Description
				descriptionLines = append(descriptionLines, line)
			}
		}
	}

	// 设置 Summary 和 Description
	if len(summaryLines) > 0 {
		swaggerMethod.Summary = strings.Join(summaryLines, " ")
	}
	if len(descriptionLines) > 0 {
		swaggerMethod.Description = strings.Join(descriptionLines, " ")
	}

	swaggerMethod.Comments = commentLines

	// 如果没有找到任何 Swagger 注释，返回 nil
	if swaggerMethod.Path == "" {
		return nil, nil
	}

	return swaggerMethod, nil
}

// parseSwaggerAnnotation 解析 Swagger 注释
func (p *AnnotationParser) parseSwaggerAnnotation(line string, method *SwaggerMethod) {
	line = strings.TrimSpace(line)

	// 解析 HTTP 方法和路径: @POST(/api/v1/swap/v1/{id})
	if httpMethodRegex := regexp.MustCompile(`@(GET|POST|PUT|DELETE|PATCH)\s*\(([^)]+)\)`); httpMethodRegex.MatchString(line) {
		matches := httpMethodRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			method.HTTPMethod = matches[1]
			method.Path = matches[2]
		}
	}

	// 解析内容类型
	switch {
	case strings.HasPrefix(line, "@FORM"):
		method.ContentType = "application/x-www-form-urlencoded"
	case strings.HasPrefix(line, "@JSON"):
		method.ContentType = "application/json"
		method.AcceptType = "application/json"
	case strings.HasPrefix(line, "@MULTIPART"):
		method.ContentType = "multipart/form-data"
	case strings.HasPrefix(line, "@MIME"):
		// 解析自定义 MIME 类型: @MIME(application/x-json-stream)
		if mimeRegex := regexp.MustCompile(`@MIME\s*\(([^)]+)\)`); mimeRegex.MatchString(line) {
			matches := mimeRegex.FindStringSubmatch(line)
			if len(matches) == 2 {
				method.ContentType = matches[1]
			}
		}
	}
}

// ParseParameterAnnotations 解析参数注释
func (p *AnnotationParser) ParseParameterAnnotations(field *ast.Field) []Parameter {
	if field.Doc == nil {
		return nil
	}

	var parameters []Parameter

	// 获取参数名称
	var paramNames []string
	for _, name := range field.Names {
		paramNames = append(paramNames, name.Name)
	}

	// 解析注释
	for _, comment := range field.Doc.List {
		line := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))

		for _, paramName := range paramNames {
			param := Parameter{
				Name:     paramName,
				Required: true, // 默认必需
			}

			// 解析参数类型注释
			switch {
			case strings.HasPrefix(line, "@PARM"):
				param.Source = "path"
			case strings.HasPrefix(line, "@FORM"):
				param.Source = "formData"
			case strings.HasPrefix(line, "@JSON"), strings.HasPrefix(line, "@BODY"):
				param.Source = "body"
			case strings.HasPrefix(line, "@QUERY"):
				param.Source = "query"
			case strings.HasPrefix(line, "@HEADER"):
				param.Source = "header"
			default:
				continue
			}

			parameters = append(parameters, param)
		}
	}

	return parameters
}

// extractPathParameters 从路径中提取参数
func (p *AnnotationParser) extractPathParameters(path string) []Parameter {
	var parameters []Parameter

	// 匹配路径参数 {id}, {name} 等
	pathParamRegex := regexp.MustCompile(`\{([^}]+)\}`)
	matches := pathParamRegex.FindAllStringSubmatch(path, -1)

	for _, match := range matches {
		if len(match) == 2 {
			paramName := match[1]
			param := Parameter{
				Name:     paramName,
				Source:   "path",
				Required: true,
			}
			parameters = append(parameters, param)
		}
	}

	return parameters
}

// mergeParameters 合并参数列表，严格保持接口定义的顺序
func (p *AnnotationParser) mergeParameters(pathParams, interfaceParams []Parameter) []Parameter {
	// 创建路径参数映射，用于更新参数信息
	pathParamMap := make(map[string]Parameter)
	for _, param := range pathParams {
		pathParamMap[param.Name] = param
	}

	// 以接口参数的顺序为准，并用路径参数信息更新
	var merged []Parameter
	for _, param := range interfaceParams {
		// 如果路径参数中有同名参数，使用路径参数的信息（特别是 Source）
		if pathParam, exists := pathParamMap[param.Name]; exists {
			// 保留接口参数的类型信息，但使用路径参数的 Source
			finalParam := param
			finalParam.Source = pathParam.Source
			merged = append(merged, finalParam)
		} else {
			// 没有对应的路径参数，直接使用接口参数
			merged = append(merged, param)
		}
	}

	return merged
}
