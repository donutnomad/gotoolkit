package main

import (
	"github.com/donutnomad/gotoolkit/internal/xast"
	"github.com/samber/lo"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// NewInterfaceParser 创建接口解析器
func NewInterfaceParser(importMgr *EnhancedImportManager) *InterfaceParser {
	return &InterfaceParser{
		importMgr: importMgr,
	}
}

// ParseFile 解析文件中的接口
func (p *InterfaceParser) ParseFile(filename string) (*InterfaceCollection, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// 获取包路径
	packagePath, err := p.getPackagePath(filename)
	if err != nil {
		return nil, err
	}

	// 获取导入信息
	imports := new(xast.ImportInfoSlice).From(file.Imports)

	// 将原始导入信息添加到导入管理器
	p.importMgr.AddOriginalImports(imports)

	// 创建注释解析器
	annotationParser := NewAnnotationParser(fileSet)

	// 创建类型解析器
	typeParser := NewReturnTypeParser(p.importMgr, imports)

	var interfaces []SwaggerInterface

	// 遍历文件中的所有声明
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		// 遍历类型声明
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}

			// 解析接口
			swaggerInterface := SwaggerInterface{
				Name:        typeSpec.Name.Name,
				PackagePath: packagePath,
				Imports:     imports,
				Methods:     []SwaggerMethod{},
			}

			// 解析接口注释
			if genDecl.Doc != nil {
				for _, comment := range genDecl.Doc.List {
					line := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
					if line != "" {
						swaggerInterface.Comments = append(swaggerInterface.Comments, line)
					}
				}
			}

			// 解析接口方法
			for _, field := range interfaceType.Methods.List {
				funcType, ok := field.Type.(*ast.FuncType)
				if !ok {
					continue
				}

				// 获取方法名
				if len(field.Names) == 0 {
					continue
				}

				// 创建虚拟的函数声明来解析注释
				virtualFunc := &ast.FuncDecl{
					Name: field.Names[0],
					Type: funcType,
					Doc:  field.Doc,
				}

				// 解析方法注释
				swaggerMethod, err := annotationParser.ParseMethodAnnotations(virtualFunc)
				if err != nil {
					continue
				}

				// 如果没有找到 Swagger 注释，跳过
				if swaggerMethod == nil {
					continue
				}

				// 解析方法参数
				p.parseMethodParameters(swaggerMethod, funcType, typeParser, annotationParser)

				// 解析返回类型
				p.parseMethodReturnType(swaggerMethod, funcType, typeParser)

				// 添加方法到接口
				swaggerInterface.Methods = append(swaggerInterface.Methods, *swaggerMethod)
			}

			// 只添加包含 Swagger 方法的接口
			if len(swaggerInterface.Methods) > 0 {
				interfaces = append(interfaces, swaggerInterface)
			}
		}
	}

	return &InterfaceCollection{
		Interfaces: interfaces,
		ImportMgr:  p.importMgr,
	}, nil
}

// parseMethodParameters 解析方法参数
func (p *InterfaceParser) parseMethodParameters(swaggerMethod *SwaggerMethod, funcType *ast.FuncType, typeParser *ReturnTypeParser, annotationParser *AnnotationParser) {
	if funcType.Params == nil {
		return
	}

	var allParams []Parameter

	// 从路径中提取参数
	pathParams := annotationParser.extractPathParameters(swaggerMethod.Path)

	// 解析方法参数
	for _, field := range funcType.Params.List {
		// 解析参数类型
		paramType := typeParser.ParseParameterType(field.Type)

		// 解析参数注释
		annotationParams := annotationParser.ParseParameterAnnotations(field)

		// 如果没有注释，根据类型推断参数来源
		if len(annotationParams) == 0 {
			for _, name := range field.Names {
				param := Parameter{
					Name:     name.Name,
					Type:     paramType,
					Required: true,
				}

				// 根据参数名和类型推断来源
				param.Source = p.inferParameterSource(param.Name, paramType, swaggerMethod)

				allParams = append(allParams, param)
			}
		} else {
			// 使用注释信息
			for _, param := range annotationParams {
				param.Type = paramType
				allParams = append(allParams, param)
			}
		}
	}

	// 合并路径参数和方法参数
	swaggerMethod.Parameters = annotationParser.mergeParameters(pathParams, allParams)
}

// parseMethodReturnType 解析方法返回类型
func (p *InterfaceParser) parseMethodReturnType(swaggerMethod *SwaggerMethod, funcType *ast.FuncType, typeParser *ReturnTypeParser) {
	if funcType.Results == nil || len(funcType.Results.List) == 0 {
		return
	}

	// 通常取第一个返回值作为响应类型
	firstResult := funcType.Results.List[0]
	swaggerMethod.ResponseType = typeParser.ParseReturnType(firstResult.Type)
}

// isContextType 检查是否是 context.Context 类型
func (p *InterfaceParser) isContextType(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}

	return ident.Name == "context" && selector.Sel.Name == "Context"
}

// inferParameterSource 推断参数来源
func (p *InterfaceParser) inferParameterSource(paramName string, _ TypeInfo, method *SwaggerMethod) string {
	// 检查是否是路径参数
	if strings.Contains(method.Path, "{"+paramName+"}") {
		return "path"
	}

	// 根据参数名推断
	lowerName := strings.ToLower(paramName)
	if strings.Contains(lowerName, "id") && strings.Contains(method.Path, "{") {
		return "path"
	}

	// 根据内容类型推断
	switch method.ContentType {
	case "application/json":
		return "body"
	case "multipart/form-data":
		return "formData"
	default:
		return "formData"
	}
}

// getPackagePath 获取包路径
func (p *InterfaceParser) getPackagePath(filename string) (string, error) {
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}

	// 这里简化处理，实际应该使用 go/packages 来获取完整的包路径
	dir := filepath.Dir(absPath)
	return filepath.Base(dir), nil
}

// ParseDirectory 解析目录中的所有接口
func (p *InterfaceParser) ParseDirectory(dirPath string) (*InterfaceCollection, error) {
	files, err := filepath.Glob(filepath.Join(dirPath, "*.go"))
	if err != nil {
		return nil, err
	}

	var allInterfaces []SwaggerInterface

	for _, file := range files {
		// 跳过测试文件
		if strings.HasSuffix(file, "_test.go") {
			continue
		}

		collection, err := p.ParseFile(file)
		if err != nil {
			continue // 跳过有错误的文件
		}

		allInterfaces = append(allInterfaces, collection.Interfaces...)
	}

	return &InterfaceCollection{
		Interfaces: allInterfaces,
		ImportMgr:  p.importMgr,
	}, nil
}

// FilterInterfacesByName 按名称过滤接口
func (collection *InterfaceCollection) FilterInterfacesByName(names []string) *InterfaceCollection {
	if len(names) == 0 {
		return collection
	}

	filtered := lo.Filter(collection.Interfaces, func(iface SwaggerInterface, _ int) bool {
		return lo.Contains(names, iface.Name)
	})

	return &InterfaceCollection{
		Interfaces: filtered,
		ImportMgr:  collection.ImportMgr,
	}
}

// GetAllMethods 获取所有方法
func (collection *InterfaceCollection) GetAllMethods() []SwaggerMethod {
	var allMethods []SwaggerMethod

	for _, iface := range collection.Interfaces {
		allMethods = append(allMethods, iface.Methods...)
	}

	return allMethods
}
