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
// 用于解析 Go 源代码中的接口定义，提取 Swagger 注释和方法信息
func NewInterfaceParser(importMgr *EnhancedImportManager) *InterfaceParser {
	return &InterfaceParser{
		importMgr: importMgr,
	}
}

// SetConfig 设置解析配置（实现 InterfaceParserInterface 接口）
func (p *InterfaceParser) SetConfig(config *GenerationConfig) {
	// 如果将来需要在解析器中使用配置，可以在这里实现
}

// getContent 从文件的指定位置提取内容
// 根据 token.Position 提供的行列信息，从文件内容中提取指定范围的文本
// 这个函数主要用于提取参数注释，因为 Go AST 不支持解析参数间的注释
func getContent(fileContent []byte, start, end token.Position) string {
	reader := bufio.NewScanner(bytes.NewReader(fileContent))
	var lineNum int
	var sb strings.Builder

	for reader.Scan() {
		lineNum++
		line := reader.Text()

		// 处理起始行
		if lineNum == start.Line {
			var endColumn = len(line)
			if lineNum == end.Line {
				endColumn = end.Column
			}
			sb.WriteString(line[start.Column-1 : endColumn])
		}

		// 处理中间行
		if start.Line != end.Line && lineNum > start.Line && lineNum <= end.Line {
			if lineNum == end.Line {
				sb.WriteString(line[:end.Column])
			} else {
				sb.WriteString(line)
			}
		}

		// 超出范围则退出
		if lineNum > end.Line {
			break
		}
	}

	return sb.String()
}

// ParseFile 解析文件中的接口
// 这是接口解析器的核心方法，解析指定 Go 文件中的所有接口定义
// 提取包含 Swagger 注释的接口和方法，构建完整的接口信息
func (p *InterfaceParser) ParseFile(filename string) (*InterfaceCollection, error) {
	// 创建文件集合用于 token 位置跟踪
	fileSet := token.NewFileSet()

	// 解析 Go 源文件，包含注释信息
	file, err := parser.ParseFile(fileSet, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, NewParseError("Go 文件解析失败", filename, err)
	}

	// 读取文件内容，用于后续的注释提取
	fileBs, err := os.ReadFile(filename)
	if err != nil {
		return nil, NewFileError("读取文件失败", filename, err)
	}

	// 获取包路径和导入信息
	packagePath, err := p.getPackagePath(filename)
	if err != nil {
		return nil, NewParseError("获取包路径失败", filename, err)
	}

	// 从 AST 中提取导入信息
	imports := new(xast.ImportInfoSlice).From(file.Imports)
	p.importMgr.AddOriginalImports(imports)

	// 创建辅助解析器
	annotationParser := NewAnnotationParser(fileSet)
	typeParser := NewReturnTypeParser(p.importMgr, imports)

	// 解析接口定义
	interfaces, err := p.parseInterfaceDeclarations(file, fileBs, fileSet, packagePath, imports, annotationParser, typeParser)
	if err != nil {
		return nil, err
	}

	return &InterfaceCollection{
		Interfaces: interfaces,
		ImportMgr:  p.importMgr,
	}, nil
}

// parseInterfaceDeclarations 解析接口声明
// 遍历文件中的所有声明，找到接口类型并解析其方法
func (p *InterfaceParser) parseInterfaceDeclarations(file *ast.File, fileBs []byte, fileSet *token.FileSet, packagePath string, imports xast.ImportInfoSlice, annotationParser *AnnotationParser, typeParser *ReturnTypeParser) ([]SwaggerInterface, error) {
	var interfaces []SwaggerInterface
	ps, err := newTagParserSafe()
	if err != nil {
		return nil, err
	}

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

			// 解析单个接口
			swaggerInterface, err := p.parseInterface(genDecl, typeSpec, interfaceType, fileBs, fileSet, packagePath, imports, annotationParser, typeParser, ps)
			if err != nil {
				return nil, err
			}

			// 只添加包含 Swagger 方法的接口
			if swaggerInterface != nil && len(swaggerInterface.Methods) > 0 {
				interfaces = append(interfaces, *swaggerInterface)
			}
		}
	}

	return interfaces, nil
}

// parseInterface 解析单个接口
// 解析接口的基本信息、注释和所有方法
func (p *InterfaceParser) parseInterface(genDecl *ast.GenDecl, typeSpec *ast.TypeSpec, interfaceType *ast.InterfaceType, fileBs []byte, fileSet *token.FileSet, packagePath string, imports xast.ImportInfoSlice, annotationParser *AnnotationParser, typeParser *ReturnTypeParser, ps *parsers.Parser) (*SwaggerInterface, error) {
	// 创建接口基本信息
	swaggerInterface := &SwaggerInterface{
		Name:        typeSpec.Name.Name,
		PackagePath: packagePath,
		Imports:     imports,
		Methods:     []SwaggerMethod{},
	}

	// 解析接口级别的注释（作为公共注释）
	if err := p.parseInterfaceComments(genDecl, swaggerInterface, ps); err != nil {
		return nil, err
	}

	// 解析接口中的所有方法
	if err := p.parseInterfaceMethods(interfaceType, swaggerInterface, fileBs, fileSet, annotationParser, typeParser); err != nil {
		return nil, err
	}

	return swaggerInterface, nil
}

// parseInterfaceComments 解析接口级别的注释
func (p *InterfaceParser) parseInterfaceComments(genDecl *ast.GenDecl, swaggerInterface *SwaggerInterface, ps *parsers.Parser) error {
	if genDecl.Doc != nil {
		for _, comment := range genDecl.Doc.List {
			line := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if strings.HasPrefix(line, "@") {
				parse, err := ps.Parse(line)
				if err != nil {
					return NewParseError("接口注释解析失败",
						fmt.Sprintf("在接口 %s 中解析注释 '%s' 失败", swaggerInterface.Name, line), err)
				}
				swaggerInterface.CommonDef = append(swaggerInterface.CommonDef, parse.(parsers.Definition))
			}
		}
	}
	return nil
}

// parseInterfaceMethods 解析接口中的所有方法
func (p *InterfaceParser) parseInterfaceMethods(interfaceType *ast.InterfaceType, swaggerInterface *SwaggerInterface, fileBs []byte, fileSet *token.FileSet, annotationParser *AnnotationParser, typeParser *ReturnTypeParser) error {
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

		// 解析方法注释和定义
		swaggerMethod, err := annotationParser.ParseMethodAnnotations(virtualFunc)
		if err != nil {
			continue // 解析失败的方法跳过
		}

		// 如果没有找到 Swagger 注释，跳过
		if swaggerMethod == nil {
			continue
		}

		// 解析方法参数和返回类型
		p.parseMethodParameters(fileSet, fileBs, swaggerMethod, funcType, typeParser, annotationParser)
		p.parseMethodReturnType(swaggerMethod, funcType, typeParser)

		// 添加方法到接口
		swaggerInterface.Methods = append(swaggerInterface.Methods, *swaggerMethod)
	}

	return nil
}

// parseMethodParameters 解析方法参数
// 将复杂的参数解析逻辑拆分为多个小函数，提高可读性和可维护性
func (p *InterfaceParser) parseMethodParameters(fileSet *token.FileSet, fileBs []byte, swaggerMethod *SwaggerMethod, funcType *ast.FuncType, typeParser *ReturnTypeParser, annotationParser *AnnotationParser) {
	if funcType.Params == nil {
		return
	}

	// 解析参数注释
	paramAnnotations, err := p.parseParameterAnnotations(fileSet, fileBs, funcType)
	if err != nil {
		return // 参数解析失败时跳过，不中断整个流程
	}

	// 提取基础参数信息
	allParams := p.extractBaseParameters(funcType.Params.List, paramAnnotations, typeParser, annotationParser)

	// 处理路径参数映射
	p.mapPathParameters(swaggerMethod, allParams)

	swaggerMethod.Parameters = allParams
}

// parseParameterAnnotations 解析参数注释
func (p *InterfaceParser) parseParameterAnnotations(fileSet *token.FileSet, fileBs []byte, funcType *ast.FuncType) ([]parsers.Parameter, error) {
	// 获取参数的整体文本解析出注释（ast不支持解析参数间的注释)
	start := fileSet.Position(funcType.Params.Opening)
	end := fileSet.Position(funcType.Params.Closing)
	text := getContent(fileBs, start, end)

	return parsers.ParseParameters(text)
}

// extractBaseParameters 提取基础参数信息
func (p *InterfaceParser) extractBaseParameters(fields []*ast.Field, paramAnnotations []parsers.Parameter, typeParser *ReturnTypeParser, annotationParser *AnnotationParser) []Parameter {
	var allParams []Parameter

	for _, field := range fields {
		paramType := typeParser.ParseParameterType(field.Type)
		fullType := xast.GetFieldType(field.Type, nil) // example: *gin.Context

		for _, item := range paramAnnotations {
			if item.Type == fullType {
				parameter := annotationParser.ParseParameterAnnotations(item.Name, item.Tag)
				parameter.Type = paramType
				allParams = append(allParams, parameter)
			}
		}
	}

	return allParams
}

// mapPathParameters 映射路径参数
func (p *InterfaceParser) mapPathParameters(swaggerMethod *SwaggerMethod, allParams []Parameter) {
	// 提取路径中的变量
	for _, routerPath := range swaggerMethod.GetPaths() {
		pathParams := p.extractPathParameters(routerPath)
		p.processPathParams(routerPath, pathParams, allParams)
	}
}

// extractPathParameters 提取路径参数（从AnnotationParser移动到这里以减少依赖）
func (p *InterfaceParser) extractPathParameters(path string) []Parameter {
	var params []Parameter

	// 查找 {param} 格式的参数
	start := 0
	for {
		openIdx := strings.Index(path[start:], "{")
		if openIdx == -1 {
			break
		}
		openIdx += start

		closeIdx := strings.Index(path[openIdx:], "}")
		if closeIdx == -1 {
			break
		}
		closeIdx += openIdx

		paramName := path[openIdx+1 : closeIdx]
		params = append(params, Parameter{
			Name:   paramName,
			Source: ParamSourcePath,
		})

		start = closeIdx + 1
	}

	return params
}

// processPathParams 处理路径参数
func (p *InterfaceParser) processPathParams(routerPath string, pathParams []Parameter, allParams []Parameter) {
	for _, pathParam := range pathParams {
		paramIndex := p.findMatchingParameter(pathParam, allParams)

		if paramIndex != -1 {
			allParams[paramIndex].PathName = pathParam.Name
			allParams[paramIndex].Source = ParamSourcePath
		} else {
			// 记录警告而不是中断程序
			fmt.Printf("警告: 在路径 %s 中未找到参数 `%s`，请使用 @PARAM(%s) 注释来修复\n",
				routerPath, pathParam.Name, pathParam.Name)
		}
	}
}

// findMatchingParameter 查找匹配的参数
func (p *InterfaceParser) findMatchingParameter(pathParam Parameter, allParams []Parameter) int {
	for i, param := range allParams {
		// 直接名称匹配
		if param.Name == pathParam.Name {
			return i
		}

		// 别名匹配
		if param.Alias != "" && param.Alias == pathParam.Name {
			return i
		}

		// 驼峰命名映射（将request_id映射到requestID）
		if parsers.NewCamelString(pathParam.Name).Equal(param.Name) {
			allParams[i].Alias = pathParam.Name
			allParams[i].Source = ParamSourcePath
			return i
		}
	}

	return -1
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
