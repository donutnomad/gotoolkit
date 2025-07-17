package main

import (
	"fmt"
	parsers "github.com/donutnomad/gotoolkit/swagGen/parser"
	"go/ast"
	"go/token"
	"regexp"
	"strings"
)

// NewAnnotationParser 创建注释解析器
func NewAnnotationParser(fileSet *token.FileSet) *AnnotationParser {
	parser := newTagParser()
	if parser == nil {
		// 创建一个简单的 parser 如果注册失败
		parser = parsers.NewParser()
	}
	return &AnnotationParser{
		fileSet:    fileSet,
		tagsParser: parser,
	}
}

// ParseMethodAnnotations 解析方法注释
func (p *AnnotationParser) ParseMethodAnnotations(method *ast.FuncDecl) (*SwaggerMethod, error) {
	if method.Doc == nil {
		return nil, nil
	}

	swaggerMethod := &SwaggerMethod{
		Name: method.Name.Name,
	}

	var commentLines []string
	var summaryLines []string
	var descriptionLines []string

	// // 解析接口注释(作为公共注释)
	//			if genDecl.Doc != nil {
	//				for _, comment := range genDecl.Doc.List {
	//					line := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
	//					if strings.HasPrefix(line, "@") {
	//						parse, err := ps.Parse(line)
	//						if err != nil {
	//							panic(err)
	//						}
	//						swaggerInterface.CommonDef = append(swaggerInterface.CommonDef, parse.(parsers.Definition))
	//					}
	//				}
	//			}

	var isDescription = false

	// 解析所有注释行
	for _, comment := range method.Doc.List {
		line := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		commentLines = append(commentLines, line)

		// 解析特殊注释
		if strings.HasPrefix(line, "@") {
			isDescription = false
			parse, err := p.tagsParser.Parse(line)
			if err != nil {
				// 记录错误并跳过无法解析的注释，而不是崩溃
				return nil, NewParseError("方法注释解析失败",
					fmt.Sprintf("在方法 %s 中解析注释 '%s' 失败", swaggerMethod.Name, line), err)
			}
			swaggerMethod.Def = append(swaggerMethod.Def, parse.(parsers.Definition))
		} else if line != "" {
			// 第一行非空注释作为 Summary
			if len(summaryLines) == 0 {
				summaryLines = append(summaryLines, line)
				isDescription = true
			} else {
				if isDescription {
					// 后续注释作为 Description
					descriptionLines = append(descriptionLines, line)
				}
			}
		}
	}

	// 设置 Summary 和 Description
	if len(summaryLines) > 0 {
		swaggerMethod.Summary = strings.TrimSpace(strings.TrimPrefix(strings.Join(summaryLines, " "), swaggerMethod.Name))
	}
	if len(descriptionLines) > 0 {
		swaggerMethod.Description = strings.Join(descriptionLines, "\n")
	}

	// 如果没有找到任何 Swagger 注释，返回 nil
	if len(swaggerMethod.GetPaths()) == 0 {
		return nil, nil
	}

	return swaggerMethod, nil
}

// ParseParameterAnnotations 解析参数注释
func (p *AnnotationParser) ParseParameterAnnotations(paramName string, tag string) Parameter {
	param := Parameter{
		Name:     paramName,
		Required: true, // 默认必需
	}
	line := tag

	// 解析参数类型注释
	switch {
	case strings.HasPrefix(line, "@PARAM"):
		param.Source = "path"
		// 匹配 @PARAM(alias)
		if aliasRegex := regexp.MustCompile(`@PARAM\s*\(([^)]+)\)`); aliasRegex.MatchString(line) {
			matches := aliasRegex.FindStringSubmatch(line)
			if len(matches) == 2 {
				param.Alias = matches[1]
			}
		}
	case strings.HasPrefix(line, "@HEADER"):
		param.Source = "header"
	}

	return param
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

// ParseCommonAnnotation 解析接口级别的通用注释，支持 exclude 语法
// 例如: @TAG(Company;exclude="StartTransfer,GetTokenHistory")
func (p *AnnotationParser) ParseCommonAnnotation(line string) *CommonAnnotation {
	// 匹配括号内的内容
	re := regexp.MustCompile(`\(([^)]+)\)`)
	matches := re.FindStringSubmatch(line)
	if len(matches) < 2 {
		return nil
	}

	content := matches[1]
	parts := strings.SplitN(content, ";", 2)
	value := strings.TrimSpace(parts[0])

	var excludes []string
	if len(parts) > 1 {
		excludePart := strings.TrimSpace(parts[1])
		if strings.HasPrefix(excludePart, "exclude=") {
			excludeStr := strings.TrimPrefix(excludePart, "exclude=")
			excludeStr = strings.Trim(excludeStr, `"`)
			for _, item := range strings.Split(excludeStr, ",") {
				excludes = append(excludes, strings.TrimSpace(item))
			}
		}
	}

	return &CommonAnnotation{
		Value:   value,
		Exclude: excludes,
	}
}
