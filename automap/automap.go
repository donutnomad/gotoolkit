package automap

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"runtime"
	"strings"
)

// AutoMap 主解析器
type AutoMap struct {
	parser          *Parser
	typeResolver    *TypeResolver
	mappingAnalyzer *MappingAnalyzer
	validator       *Validator
	codeGenerator   *CodeGenerator
	fset            *token.FileSet
	currentFile     string
}

// New 创建新的AutoMap实例
func New() *AutoMap {
	fset := token.NewFileSet()
	typeResolver := NewTypeResolver()
	return &AutoMap{
		parser:          NewParser(),
		typeResolver:    typeResolver,
		mappingAnalyzer: NewMappingAnalyzer(fset),
		validator:       NewValidator(nil), // 稍后设置typeResolver
		codeGenerator:   NewCodeGenerator(typeResolver),
		fset:            fset,
	}
}

// Parse 解析映射函数并生成代码
func (am *AutoMap) Parse(funcName, callerFile string) (*ParseResult, error) {
	return am.ParseWithContext(funcName, callerFile)
}

// ParseWithContext 使用指定文件上下文解析映射函数
func (am *AutoMap) ParseWithContext(funcName, callerFile string) (*ParseResult, error) {
	am.currentFile = callerFile
	am.codeGenerator.typeResolver.currentFile = callerFile
	am.parser.SetCurrentFile(callerFile)
	// 解析函数签名
	funcSignature, aType, bType, err := am.parser.ParseFunction(funcName)
	if err != nil {
		return nil, fmt.Errorf("解析函数签名失败: %w", err)
	}

	// 解析类型详细信息
	if err := am.resolveTypes(aType, bType); err != nil {
		return nil, fmt.Errorf("解析类型详细信息失败: %w", err)
	}

	// 设置typeResolver到validator
	am.validator.typeResolver = am.typeResolver

	// 验证函数签名和类型
	if err := am.validate(funcSignature, aType, bType); err != nil {
		return nil, fmt.Errorf("验证失败: %w", err)
	}

	// 查找函数定义并分析映射关系
	mappingRelations, fieldMapping, err := am.analyzeMapping(funcName, aType, bType)
	if err != nil {
		return nil, fmt.Errorf("分析映射关系失败: %w", err)
	}

	// 解析A和B的嵌套类型
	for idx, item := range aType.Fields {
		if item.IsEmbedded {
			item.EmbeddedFields = am.codeGenerator.getEmbeddedDatabaseFields(item.Type)
			aType.Fields[idx] = item
		}
	}
	for idx, item := range bType.Fields {
		if item.IsEmbedded {
			item.EmbeddedFields = am.codeGenerator.getEmbeddedDatabaseFields(item.Type)
			bType.Fields[idx] = item
		}
	}

	// 检查ExportPatch方法
	hasExportPatch := am.typeResolver.HasExportPatchMethod(aType)

	// 生成代码
	code := am.generateCode(funcSignature, aType, bType, fieldMapping)

	// 构建结果
	result := &ParseResult{
		FuncSignature:    *funcSignature,
		AType:            *aType,
		BType:            *bType,
		FieldMapping:     fieldMapping,
		HasExportPatch:   hasExportPatch,
		GeneratedCode:    code,
		MappingRelations: mappingRelations,
	}

	//// 最终验证
	//if err := am.validator.Validate(result); err != nil {
	//	return nil, fmt.Errorf("最终验证失败: %w", err)
	//}

	return result, nil
}

// resolveTypes 解析类型详细信息
func (am *AutoMap) resolveTypes(aType, bType *TypeInfo) error {
	// 解析A类型
	if err := am.typeResolver.ResolveTypeCurrent(aType); err != nil {
		return fmt.Errorf("解析A类型失败: %w", err)
	}

	// 解析B类型
	if err := am.typeResolver.ResolveTypeCurrent(bType); err != nil {
		return fmt.Errorf("解析B类型失败: %w", err)
	}

	return nil
}

// validate 验证函数签名和类型
func (am *AutoMap) validate(funcSignature *FuncSignature, aType, bType *TypeInfo) error {
	// 验证函数签名
	if err := am.validator.ValidateFunctionSignature(funcSignature); err != nil {
		return err
	}

	// 验证类型
	if err := am.validator.ValidateTypes(aType, bType); err != nil {
		return err
	}

	return nil
}

// analyzeMapping 分析映射关系
func (am *AutoMap) analyzeMapping(funcName string, aType, bType *TypeInfo) ([]MappingRelation, FieldMapping, error) {
	// 查找函数定义的AST节点
	funcDecl, err := am.findFunctionDeclaration(funcName)
	if err != nil {
		return nil, FieldMapping{}, fmt.Errorf("查找函数定义失败: %w", err)
	}

	// 分析映射关系
	mappingRelations, fieldMapping, err := am.mappingAnalyzer.AnalyzeMapping(funcDecl, aType, bType)
	if err != nil {
		return nil, FieldMapping{}, fmt.Errorf("分析映射关系失败: %w", err)
	}

	return mappingRelations, fieldMapping, nil
}

// findFunctionDeclaration 查找函数声明
func (am *AutoMap) findFunctionDeclaration(funcName string) (*ast.FuncDecl, error) {
	// 解析函数名格式
	receiver, actualFuncName := am.parseFunctionName(funcName)

	// 使用当前文件路径
	if am.currentFile == "" {
		return nil, fmt.Errorf("未设置当前文件路径")
	}

	// 解析单个文件
	file, err := parser.ParseFile(am.fset, am.currentFile, nil, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("解析文件失败: %w", err)
	}

	// 查找函数
	if receiver != "" {
		// 查找方法
		if fn := am.findMethodInFile(file, receiver, actualFuncName); fn != nil {
			return fn, nil
		}
	} else {
		// 查找函数
		if fn := am.findFunctionInFile(file, actualFuncName); fn != nil {
			return fn, nil
		}
	}

	return nil, fmt.Errorf("未找到函数: %s", funcName)
}

// findFunctionInFile 在单个文件中查找函数（不包括方法）
func (am *AutoMap) findFunctionInFile(file *ast.File, funcName string) *ast.FuncDecl {
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
func (am *AutoMap) findMethodInFile(file *ast.File, receiver, methodName string) *ast.FuncDecl {
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

// parseFunctionName 解析函数名格式
func (am *AutoMap) parseFunctionName(funcName string) (receiver, name string) {
	parts := strings.Split(funcName, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", parts[0]
}

// findFunctionInPackage 在包中查找函数
func (am *AutoMap) findFunctionInPackage(pkg *ast.Package, funcName string) *ast.FuncDecl {
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == funcName {
				return fn
			}
		}
	}
	return nil
}

// findMethodInPackage 在包中查找方法
func (am *AutoMap) findMethodInPackage(pkg *ast.Package, receiver, methodName string) *ast.FuncDecl {
	for _, file := range pkg.Files {
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
	}
	return nil
}

// generateCode 生成代码
func (am *AutoMap) generateCode(funcSignature *FuncSignature, aType, bType *TypeInfo, fieldMapping FieldMapping) string {
	result := &ParseResult{
		FuncSignature: *funcSignature,
		AType:         *aType,
		BType:         *bType,
		FieldMapping:  fieldMapping,
	}

	return am.codeGenerator.Generate(result)
}

// getCallerFile 获取调用者文件路径
func (am *AutoMap) getCallerFile() string {
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

		fmt.Println(fn.Name())

		// 跳过automap包中的函数
		if !am.contains(fn.Name(), "automap.") {
			return file
		}
	}
	return ""
}

// contains 检查字符串是否包含子字符串
func (am *AutoMap) contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// ParseWithOptions 使用选项解析映射函数
func (am *AutoMap) ParseWithOptions(funcName string, options ...Option) (*ParseResult, error) {
	var callerFile string

	// 应用选项
	for _, option := range options {
		callerFile = option(am, callerFile)
	}

	// 如果没有指定文件上下文，使用默认方式获取
	if callerFile == "" {
		callerFile = am.getCallerFile()
	}

	return am.ParseWithContext(funcName, callerFile)
}

// Option 解析选项
type Option func(*AutoMap, string) string

// WithFileContext 设置文件上下文
func WithFileContext(filePath string) Option {
	return func(am *AutoMap, current string) string {
		return filePath
	}
}

// ParseAndGenerate 解析并生成完整代码（包含导入）
func (am *AutoMap) ParseAndGenerate(funcName string, options ...Option) (string, error) {
	var callerFile string

	// 应用选项
	for _, option := range options {
		callerFile = option(am, callerFile)
	}

	// 如果没有指定文件上下文，使用默认方式获取
	if callerFile == "" {
		callerFile = am.getCallerFile()
	}

	result, err := am.Parse(funcName, callerFile)
	if err != nil {
		return "", err
	}

	return am.codeGenerator.GenerateFullCode(result), nil
}

// ValidateFunction 验证函数是否符合要求
func (am *AutoMap) ValidateFunction(funcName, callerFile string) error {
	result, err := am.Parse(funcName, callerFile)
	if err != nil {
		return err
	}

	// 检查ExportPatch方法
	if !result.HasExportPatch {
		return fmt.Errorf("类型 %s 缺少ExportPatch方法", result.AType.Name)
	}

	return nil
}

// 全局实例
var defaultAutoMap = New()

// ParseWithOptions 使用选项解析映射函数（使用默认实例）
func ParseWithOptions(funcName string, options ...Option) (*ParseResult, error) {
	return defaultAutoMap.ParseWithOptions(funcName, options...)
}

// ParseAndGenerate 解析并生成完整代码（使用默认实例）
func ParseAndGenerate(funcName string, options ...Option) (string, error) {
	return defaultAutoMap.ParseAndGenerate(funcName, options...)
}
