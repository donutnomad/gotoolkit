package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/donutnomad/gotoolkit/internal/xast"
	parsers "github.com/donutnomad/gotoolkit/swagGen/parser"
	"github.com/samber/lo"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// NewInterfaceParser 创建接口解析器
func NewInterfaceParser(importMgr *EnhancedImportManager) *InterfaceParser {
	return &InterfaceParser{
		importMgr: importMgr,
	}
}

// 从文件的line:column获取文件内容
func getContent(fileContent []byte, start, end token.Position) string {
	reader := bufio.NewScanner(bytes.NewReader(fileContent))
	var lineNum int
	var sb strings.Builder
	for reader.Scan() {
		lineNum++
		line := reader.Text()
		if lineNum == start.Line {
			var endColumn = len(line)
			if lineNum == end.Line {
				endColumn = end.Column
			}
			sb.WriteString(line[start.Column-1 : endColumn])
		}
		if start.Line != end.Line && lineNum > start.Line && lineNum <= end.Line {
			if lineNum == end.Line {
				sb.WriteString(line[:end.Column])
			} else {
				sb.WriteString(line)
			}
		}
		if lineNum > end.Line {
			break
		}
	}
	return sb.String()
}

// ParseFile 解析文件中的接口
func (p *InterfaceParser) ParseFile(filename string) (*InterfaceCollection, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	fileBs, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	_ = fileBs

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

	ps := newTagParser()

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

			// 解析接口注释(作为公共注释)
			if genDecl.Doc != nil {
				for _, comment := range genDecl.Doc.List {
					line := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
					if strings.HasPrefix(line, "@") {
						parse, err := ps.Parse(line)
						if err != nil {
							panic(err)
						}
						swaggerInterface.CommonDef = append(swaggerInterface.CommonDef, parse.(parsers.Definition))
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
				p.parseMethodParameters(fileSet, fileBs, swaggerMethod, funcType, typeParser, annotationParser)

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
func (p *InterfaceParser) parseMethodParameters(fileSet *token.FileSet, fileBs []byte, swaggerMethod *SwaggerMethod, funcType *ast.FuncType, typeParser *ReturnTypeParser, annotationParser *AnnotationParser) {
	if funcType.Params == nil {
		return
	}

	// 获取参数的整体文本解析出注释（ast不支持解析参数间的注释)
	a := fileSet.Position(funcType.Params.Opening)
	b := fileSet.Position(funcType.Params.Closing)
	text := getContent(fileBs, a, b)
	ps, err := parsers.ParseParameters(text)
	if err != nil {
		panic(err)
	}

	var allParams []Parameter

	for _, field := range funcType.Params.List {
		paramType := typeParser.ParseParameterType(field.Type)
		fullType := xast.GetFieldType(field.Type, nil) // example: *gin.Context
		for _, item := range ps {
			if item.Type == fullType {
				parameter := annotationParser.ParseParameterAnnotations(item.Name, item.Tag)
				parameter.Type = paramType
				allParams = append(allParams, parameter)
			}
		}
	}

	// 提取路径中的变量
	for _, routerPath := range swaggerMethod.GetPaths() {
		pathParams := annotationParser.extractPathParameters(routerPath)
		for _, pathParam := range pathParams {
			var idx = -1
			for i := range allParams {
				item := allParams[i]
				if item.Name == pathParam.Name || (item.Alias != "" && item.Alias == pathParam.Name) {
					idx = i
					break
				} else if parsers.NewCamelString(pathParam.Name).Equal(item.Name) { // 将request_id这种路径变量名映射到requestID或者requestId这种没有注释的变量中，节省开发注意力消耗
					allParams[i].Alias = pathParam.Name
					allParams[i].Source = "path"
					idx = i
					break
				}
			}
			if idx != -1 {
				allParams[idx].PathName = pathParam.Name
				allParams[idx].Source = "path"
			} else {
				panic(fmt.Sprintf("path %s param `%s` was not found in %s \n Use @PARAM(%s) to fix it.", routerPath, pathParam.Name, clean.Replace(text), pathParam.Name))
			}
		}
	}

	swaggerMethod.Parameters = allParams
}

var clean = strings.NewReplacer("\t", " ", "\n", " ")

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
