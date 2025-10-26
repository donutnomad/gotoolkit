package automap

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Parser AST解析器
type Parser struct {
	fset        *token.FileSet
	pkgPath     string
	pkgName     string
	currentFile string
}

// NewParser 创建新的解析器
func NewParser() *Parser {
	return &Parser{
		fset: token.NewFileSet(),
	}
}

// SetCurrentFile 设置当前文件路径
func (p *Parser) SetCurrentFile(filePath string) {
	p.currentFile = filePath
}

// ParseFunction 解析函数签名
func (p *Parser) ParseFunction(funcName string) (*FuncSignature, *TypeInfo, *TypeInfo, error) {
	// 解析函数名格式（支持 "Func" 和 "X.Func" 格式）
	receiver, actualFuncName := p.parseFunctionName(funcName)

	// 使用当前文件所在目录解析包
	var parsePath string
	if p.currentFile != "" {
		parsePath = p.currentFile
	} else {
		parsePath = "."
	}

	file, err := p.parseFile(parsePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("解析文件失败: %w", err)
	}

	p.pkgPath = "" // 使用当前文件所在目录
	p.pkgName = file.Name.Name

	// 查找目标函数
	var targetFunc *ast.FuncDecl
	if receiver != "" {
		// 查找方法
		targetFunc = p.findMethodInFile(file, receiver, actualFuncName)
	} else {
		// 查找函数
		targetFunc = p.findFunctionInFile(file, actualFuncName)

		// 如果找不到函数，尝试查找方法（可能有接收者但没有指定）
		if targetFunc == nil {
			receiver, targetFunc = p.findMethodAnyInFile(file, actualFuncName)
		}
	}

	if targetFunc == nil {
		return nil, nil, nil, fmt.Errorf("未找到函数: %s", funcName)
	}

	// 解析函数签名
	sig, inputType, outputType, err := p.parseFuncSignature(targetFunc, receiver)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("解析函数签名失败: %w", err)
	}

	return sig, inputType, outputType, nil
}

// parseFunctionName 解析函数名格式
func (p *Parser) parseFunctionName(funcName string) (receiver, name string) {
	parts := strings.Split(funcName, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", parts[0]
}

// getCallerInfo 获取调用者信息
func (p *Parser) getCallerInfo() string {
	// 查找调用栈中第一个不在automap包中的文件
	for i := 1; i < 10; i++ {
		pc, file, _, ok := runtime.Caller(i)
		if !ok {
			break
		}

		// 获取函数信息
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		// 跳过automap包中的函数
		if !p.contains(fn.Name(), "automap.") {
			return file
		}
	}
	return ""
}

// contains 检查字符串是否包含子字符串
func (p *Parser) contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// parseFile 解析单个文件
func (p *Parser) parseFile(filePath string) (*ast.File, error) {
	// 解析单个文件
	file, err := parser.ParseFile(p.fset, filePath, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// findFunctionInFile 在单个文件中查找函数（不包括方法）
func (p *Parser) findFunctionInFile(file *ast.File, funcName string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == funcName {
			// 只有当没有接收者时才是真正的函数
			if fn.Recv == nil || len(fn.Recv.List) == 0 {
				return fn
			}
		}
	}
	return nil
}

// findMethodInFile 在单个文件中查找方法
func (p *Parser) findMethodInFile(file *ast.File, receiver, methodName string) *ast.FuncDecl {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == methodName {
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				// 检查接收者类型
				if recvType, ok := fn.Recv.List[0].Type.(*ast.Ident); ok {
					if recvType.Name == receiver {
						return fn
					}
				}
				// 处理指针接收者
				if starExpr, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
					if ident, ok := starExpr.X.(*ast.Ident); ok && ident.Name == receiver {
						return fn
					}
				}
			}
		}
	}
	return nil
}

// findMethodAnyInFile 在单个文件中的任何类型中查找方法
func (p *Parser) findMethodAnyInFile(file *ast.File, methodName string) (string, *ast.FuncDecl) {
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == methodName {
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				// 提取接收者类型名
				if recvType, ok := fn.Recv.List[0].Type.(*ast.Ident); ok {
					return recvType.Name, fn
				}
				// 处理指针接收者
				if starExpr, ok := fn.Recv.List[0].Type.(*ast.StarExpr); ok {
					if ident, ok := starExpr.X.(*ast.Ident); ok {
						return ident.Name, fn
					}
				}
			}
		}
	}
	return "", nil
}

// parseFuncSignature 解析函数签名
func (p *Parser) parseFuncSignature(fn *ast.FuncDecl, receiver string) (*FuncSignature, *TypeInfo, *TypeInfo, error) {
	sig := &FuncSignature{
		PackageName: p.pkgName,
		Receiver:    receiver,
		FuncName:    fn.Name.Name,
		Pos:         fn.Pos(),
	}

	// 解析参数
	if len(fn.Type.Params.List) != 1 {
		return nil, nil, nil, fmt.Errorf("函数必须有且只有一个参数")
	}

	param := fn.Type.Params.List[0]
	if len(param.Names) != 1 {
		return nil, nil, nil, fmt.Errorf("参数必须有且只有一个名称")
	}

	inputType, err := p.parseType(param.Type)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("解析输入类型失败: %w", err)
	}

	sig.InputType = *inputType

	// 解析返回值
	if len(fn.Type.Results.List) == 0 {
		return nil, nil, nil, fmt.Errorf("函数必须有返回值")
	}

	if len(fn.Type.Results.List) == 1 {
		// 单个返回值
		result := fn.Type.Results.List[0]
		outputType, err := p.parseType(result.Type)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("解析输出类型失败: %w", err)
		}
		sig.OutputType = *outputType
	} else if len(fn.Type.Results.List) == 2 {
		// 两个返回值，检查第二个是否为error
		first := fn.Type.Results.List[0]
		second := fn.Type.Results.List[1]

		// 检查第二个返回值是否为error
		if !p.isErrorType(second.Type) {
			return nil, nil, nil, fmt.Errorf("函数的第二个返回值必须是error类型")
		}

		outputType, err := p.parseType(first.Type)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("解析输出类型失败: %w", err)
		}
		sig.OutputType = *outputType
		sig.HasError = true
	} else {
		return nil, nil, nil, fmt.Errorf("函数返回值数量不正确")
	}

	// 验证输入输出类型都是指针类型
	if !sig.InputType.IsPointer {
		return nil, nil, nil, fmt.Errorf("输入类型必须是指针类型")
	}

	if !sig.OutputType.IsPointer {
		return nil, nil, nil, fmt.Errorf("输出类型必须是指针类型")
	}

	return sig, inputType, &sig.OutputType, nil
}

// parseType 解析类型
func (p *Parser) parseType(expr ast.Expr) (*TypeInfo, error) {
	switch t := expr.(type) {
	case *ast.StarExpr:
		// 指针类型
		baseType, err := p.parseType(t.X)
		if err != nil {
			return nil, err
		}
		baseType.IsPointer = true
		return baseType, nil

	case *ast.Ident:
		// 简单标识符
		return &TypeInfo{
			Name:      t.Name,
			Package:   p.pkgName,
			FullName:  t.Name,
			IsPointer: false,
		}, nil

	case *ast.SelectorExpr:
		// 包名.类型名
		packageName, ok := t.X.(*ast.Ident)
		if !ok {
			return nil, fmt.Errorf("无效的包选择器")
		}

		return &TypeInfo{
			Name:      t.Sel.Name,
			Package:   packageName.Name,
			FullName:  packageName.Name + "." + t.Sel.Name,
			IsPointer: false,
		}, nil

	default:
		return nil, fmt.Errorf("不支持的类型表达式: %T", expr)
	}
}

// isErrorType 检查是否为error类型
func (p *Parser) isErrorType(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "error"
	}
	return false
}

// findTypeDefinition 查找类型定义
func (p *Parser) findTypeDefinition(typeInfo *TypeInfo) (*ast.TypeSpec, error) {
	// 如果是当前包的类型
	if typeInfo.Package == p.pkgName || typeInfo.Package == "" {
		return p.findTypeInPackage(p.pkgPath, typeInfo.Name)
	}

	// TODO: 支持跨包类型查找
	return nil, fmt.Errorf("暂不支持跨包类型查找: %s", typeInfo.FullName)
}

// findTypeInPackage 在包中查找类型定义
func (p *Parser) findTypeInPackage(pkgPath, typeName string) (*ast.TypeSpec, error) {
	pkgs, err := parser.ParseDir(p.fset, pkgPath, nil, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				if genDecl, ok := decl.(*ast.GenDecl); ok {
					for _, spec := range genDecl.Specs {
						if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == typeName {
							return typeSpec, nil
						}
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("未找到类型定义: %s", typeName)
}

// parseDirectory 解析目录中的所有 .go 文件
func (p *Parser) parseDirectory(dirPath string) ([]*ast.File, error) {
	var files []*ast.File

	// 读取目录
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	// 解析每个 .go 文件
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			filePath := filepath.Join(dirPath, entry.Name())
			file, err := parser.ParseFile(p.fset, filePath, nil, parser.AllErrors)
			if err != nil {
				// 如果单个文件解析失败，继续解析其他文件
				continue
			}
			files = append(files, file)
		}
	}

	return files, nil
}

// getFieldType 获取字段类型字符串
func (p *Parser) getFieldType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + p.getFieldType(t.X)
	case *ast.SelectorExpr:
		return p.getFieldType(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + p.getFieldType(t.Elt)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// extractGormTag 提取GORM标签
func (p *Parser) extractGormTag(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}

	tagStr := strings.Trim(tag.Value, "`")

	if !strings.HasPrefix(tagStr, "gorm:") {
		return ""
	}

	// 去除gorm:前缀和引号
	gormContent := strings.TrimPrefix(tagStr, "gorm:")
	result := strings.Trim(gormContent, `"`)

	return result
}

// extractColumnName 提取列名
func (p *Parser) extractColumnName(gormTag string) string {
	if gormTag == "" {
		return ""
	}

	// 简单解析column:"name"
	parts := strings.Split(gormTag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "column:") {
			columnValue := strings.TrimPrefix(part, "column:")
			// 去除可能的引号
			return strings.Trim(columnValue, `"`)
		}
	}

	return ""
}

// isJSONType 检查是否为JSONType
func (p *Parser) isJSONType(expr ast.Expr) bool {
	if selectorExpr, ok := expr.(*ast.SelectorExpr); ok {
		if x, ok := selectorExpr.X.(*ast.Ident); ok {
			return x.Name == "datatypes" && selectorExpr.Sel.Name == "JSONType"
		}
	}

	// 检查泛型形式：datatypes.JSONType[B_Token]
	if indexExpr, ok := expr.(*ast.IndexExpr); ok {
		if selectorExpr, ok := indexExpr.X.(*ast.SelectorExpr); ok {
			if x, ok := selectorExpr.X.(*ast.Ident); ok {
				return x.Name == "datatypes" && selectorExpr.Sel.Name == "JSONType"
			}
		}
	}

	return false
}

// parseJSONFields 解析JSON字段
func (p *Parser) parseJSONFields(expr ast.Expr) []JSONFieldInfo {
	// TODO: 解析JSONType的泛型参数获取JSON字段信息
	return []JSONFieldInfo{}
}

// parseTypeMethods 解析类型方法
func (p *Parser) parseTypeMethods(typeSpec *ast.TypeSpec) ([]MethodInfo, error) {
	// TODO: 解析类型的方法列表，特别是检查ExportPatch方法
	return []MethodInfo{}, nil
}
