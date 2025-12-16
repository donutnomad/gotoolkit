package automap

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/donutnomad/gotoolkit/internal/gormparse"
	"github.com/donutnomad/gotoolkit/internal/structparse"
)

// Mapper 映射分析器
type Mapper struct {
	filePath string
	fset     *token.FileSet
	file     *ast.File

	// 类型定义缓存
	typeSpecs map[string]*ast.TypeSpec

	// 方法定义缓存：receiverType -> methodName -> funcDecl
	methodDecls map[string]map[string]*ast.FuncDecl

	// 当前分析的上下文
	receiverType string
	funcName     string
	paramName    string // 函数参数名（如 "d"）
	sourceType   string // 源类型名（如 "CustomerDomain"）

	// 局部变量映射：变量名 -> 源路径
	varMap map[string]string

	// 方法调用映射：变量名 -> (methodName, receiverType)
	methodCallMap map[string]methodCallInfo

	// 解析结果
	result *ParseResult2
}

// methodCallInfo 方法调用信息
type methodCallInfo struct {
	methodName   string
	receiverType string
}

// NewMapper 创建新的 Mapper
func NewMapper(filePath string) *Mapper {
	return &Mapper{
		filePath:    filePath,
		typeSpecs:   make(map[string]*ast.TypeSpec),
		methodDecls: make(map[string]map[string]*ast.FuncDecl),
	}
}

// Parse 解析指定的函数
func (m *Mapper) Parse(receiverType, funcName string) (*ParseResult2, error) {
	m.receiverType = receiverType
	m.funcName = funcName
	m.result = &ParseResult2{
		FuncName:     funcName,
		ReceiverType: receiverType,
		TargetType:   receiverType,
	}

	// 解析文件
	if err := m.parseFile(); err != nil {
		return nil, err
	}

	// 收集类型定义
	m.collectTypeSpecs()

	// 查找并分析函数
	if err := m.analyzeFunction(); err != nil {
		return nil, err
	}

	// 扁平化所有映射
	m.flattenMappings()

	// 收集目标类型的所有列名
	m.collectTargetColumns()

	return m.result, nil
}

// parseFile 解析源文件
func (m *Mapper) parseFile() error {
	m.fset = token.NewFileSet()
	var err error
	m.file, err = parser.ParseFile(m.fset, m.filePath, nil, parser.ParseComments)
	return err
}

// collectTypeSpecs 收集所有类型定义和方法定义
// 会扫描同包中的所有Go文件，以便找到跨文件的类型定义和方法
func (m *Mapper) collectTypeSpecs() {
	// 首先从当前文件收集
	m.collectTypeSpecsFromFile(m.file)

	// 然后扫描同目录下的其他Go文件
	dir := filepath.Dir(m.filePath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		fullPath := filepath.Join(dir, name)
		if fullPath == m.filePath {
			continue // 跳过已解析的文件
		}

		// 解析其他文件
		otherFile, err := parser.ParseFile(m.fset, fullPath, nil, parser.ParseComments)
		if err != nil {
			continue
		}
		m.collectTypeSpecsFromFile(otherFile)
	}
}

// collectTypeSpecsFromFile 从单个文件收集类型定义和方法定义
func (m *Mapper) collectTypeSpecsFromFile(file *ast.File) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			// 收集类型定义
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if ok {
					m.typeSpecs[typeSpec.Name.Name] = typeSpec
				}
			}

		case *ast.FuncDecl:
			// 收集方法定义
			if d.Recv == nil || len(d.Recv.List) == 0 {
				continue
			}
			recvType := m.extractTypeName(d.Recv.List[0].Type)
			if recvType == "" {
				continue
			}
			if m.methodDecls[recvType] == nil {
				m.methodDecls[recvType] = make(map[string]*ast.FuncDecl)
			}
			m.methodDecls[recvType][d.Name.Name] = d
		}
	}
}

// analyzeFunction 分析目标函数
func (m *Mapper) analyzeFunction() error {
	// 首先在当前文件中查找
	for _, decl := range m.file.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		// 检查是否是目标函数
		if !m.isTargetFunction(funcDecl) {
			continue
		}

		// 提取参数名和类型
		m.extractFuncParams(funcDecl)

		// 分析函数体
		return m.analyzeFuncBody(funcDecl.Body)
	}

	// 如果当前文件没找到，从收集的方法定义中查找
	if methods, exists := m.methodDecls[m.receiverType]; exists {
		if funcDecl, exists := methods[m.funcName]; exists {
			// 提取参数名和类型
			m.extractFuncParams(funcDecl)

			// 分析函数体
			return m.analyzeFuncBody(funcDecl.Body)
		}
	}

	return fmt.Errorf("function %s.%s not found", m.receiverType, m.funcName)
}

// isTargetFunction 检查是否是目标函数
func (m *Mapper) isTargetFunction(funcDecl *ast.FuncDecl) bool {
	// 检查函数名
	if funcDecl.Name.Name != m.funcName {
		return false
	}

	// 检查接收者
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		return false
	}

	recvType := m.extractTypeName(funcDecl.Recv.List[0].Type)
	return recvType == m.receiverType
}

// extractTypeName 提取类型名（去掉指针）
func (m *Mapper) extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return m.extractTypeName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return ""
}

// extractFuncParams 提取函数参数信息
func (m *Mapper) extractFuncParams(funcDecl *ast.FuncDecl) {
	if funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) == 0 {
		return
	}

	param := funcDecl.Type.Params.List[0]
	if len(param.Names) > 0 {
		m.paramName = param.Names[0].Name
	}

	// 提取源类型和包信息
	typeName, pkgName := m.extractTypeNameWithPackage(param.Type)
	m.sourceType = typeName
	m.result.SourceType = typeName
	m.result.SourceTypePackage = pkgName

	// 如果有包名，查找对应的导入路径
	if pkgName != "" {
		m.result.SourceTypeImportPath = m.resolveImportPath(pkgName)
	}
}

// extractTypeNameWithPackage 提取类型名和包名
// 返回: (typeName, packageName)
// 例如: *domain.ListingDomain -> ("ListingDomain", "domain")
// 例如: *ListingDomain -> ("ListingDomain", "")
func (m *Mapper) extractTypeNameWithPackage(expr ast.Expr) (string, string) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, ""
	case *ast.StarExpr:
		return m.extractTypeNameWithPackage(t.X)
	case *ast.SelectorExpr:
		// pkg.TypeName
		if pkgIdent, ok := t.X.(*ast.Ident); ok {
			return t.Sel.Name, pkgIdent.Name
		}
		return t.Sel.Name, ""
	}
	return "", ""
}

// resolveImportPath 根据包别名查找导入路径
func (m *Mapper) resolveImportPath(pkgName string) string {
	if m.file == nil {
		return ""
	}

	for _, imp := range m.file.Imports {
		// 获取导入路径（去除引号）
		importPath := strings.Trim(imp.Path.Value, "\"")

		// 检查是否有别名
		if imp.Name != nil {
			if imp.Name.Name == pkgName {
				return importPath
			}
		} else {
			// 没有别名时，使用路径的最后一部分作为包名
			parts := strings.Split(importPath, "/")
			if len(parts) > 0 && parts[len(parts)-1] == pkgName {
				return importPath
			}
		}
	}
	return ""
}

// analyzeFuncBody 分析函数体
func (m *Mapper) analyzeFuncBody(body *ast.BlockStmt) error {
	// 初始化变量映射表
	m.varMap = make(map[string]string)
	m.methodCallMap = make(map[string]methodCallInfo)

	// 第一遍：收集所有局部变量的赋值
	m.collectVariableAssignments(body)

	// 第二遍：分析 return 语句
	for _, stmt := range body.List {
		// 查找 return 语句
		retStmt, ok := stmt.(*ast.ReturnStmt)
		if !ok {
			continue
		}

		// 分析返回的结构体字面量
		for _, result := range retStmt.Results {
			if err := m.analyzeReturnExpr(result); err != nil {
				return err
			}
		}
	}
	return nil
}

// collectVariableAssignments 收集局部变量赋值
func (m *Mapper) collectVariableAssignments(body *ast.BlockStmt) {
	for _, stmt := range body.List {
		switch s := stmt.(type) {
		case *ast.AssignStmt:
			// 处理赋值语句: name := d.Name 或 name = d.Name
			m.processAssignStmt(s)

		case *ast.DeclStmt:
			// 处理变量声明: var name = d.Name
			if genDecl, ok := s.Decl.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						m.processValueSpec(valueSpec)
					}
				}
			}

		case *ast.IfStmt:
			// 递归处理 if 语句块（只收集变量，不进入条件分支的覆盖）
			// 注意：我们只收集初始赋值，忽略条件分支中的重新赋值
			if s.Init != nil {
				if assignStmt, ok := s.Init.(*ast.AssignStmt); ok {
					m.processAssignStmt(assignStmt)
				}
			}

		case *ast.ForStmt:
			// 递归处理 for 语句的初始化部分
			if s.Init != nil {
				if assignStmt, ok := s.Init.(*ast.AssignStmt); ok {
					m.processAssignStmt(assignStmt)
				}
			}

		case *ast.RangeStmt:
			// 忽略 range 语句的迭代变量
		}
	}
}

// processAssignStmt 处理赋值语句
func (m *Mapper) processAssignStmt(s *ast.AssignStmt) {
	// 只处理简单赋值（:= 或 =）
	if len(s.Lhs) != len(s.Rhs) {
		return
	}

	for i, lhs := range s.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}

		varName := ident.Name
		rhs := s.Rhs[i]

		// 检查是否是方法调用：d.MethodName()
		if methodInfo := m.extractMethodCallInfo(rhs); methodInfo != nil {
			if _, exists := m.methodCallMap[varName]; !exists {
				m.methodCallMap[varName] = *methodInfo
			}
			continue
		}

		// 提取右侧的源路径
		sourcePath := m.extractSourcePathFromExpr(rhs)
		if sourcePath != "" {
			// 只记录第一次赋值（初始值）
			if _, exists := m.varMap[varName]; !exists {
				m.varMap[varName] = sourcePath
			}
		} else {
			// 检查是否是从另一个局部变量赋值
			if rhsIdent, ok := rhs.(*ast.Ident); ok {
				if existingPath, exists := m.varMap[rhsIdent.Name]; exists {
					if _, exists := m.varMap[varName]; !exists {
						m.varMap[varName] = existingPath
					}
				}
				// 检查是否是从另一个方法调用变量赋值
				if existingMethod, exists := m.methodCallMap[rhsIdent.Name]; exists {
					if _, exists := m.methodCallMap[varName]; !exists {
						m.methodCallMap[varName] = existingMethod
					}
				}
			}
		}
	}
}

// extractMethodCallInfo 提取方法调用信息
func (m *Mapper) extractMethodCallInfo(expr ast.Expr) *methodCallInfo {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}

	// 检查是否是 d.MethodName() 形式
	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	// 检查接收者是否是参数
	recvIdent, ok := sel.X.(*ast.Ident)
	if !ok || recvIdent.Name != m.paramName {
		return nil
	}

	return &methodCallInfo{
		methodName:   sel.Sel.Name,
		receiverType: m.sourceType,
	}
}

// processValueSpec 处理变量声明
func (m *Mapper) processValueSpec(spec *ast.ValueSpec) {
	if len(spec.Names) != len(spec.Values) {
		return
	}

	for i, name := range spec.Names {
		varName := name.Name
		value := spec.Values[i]

		sourcePath := m.extractSourcePathFromExpr(value)
		if sourcePath != "" {
			m.varMap[varName] = sourcePath
		}
	}
}

// extractSourcePathFromExpr 从表达式提取源路径（仅用于变量收集）
func (m *Mapper) extractSourcePathFromExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		return m.buildSelectorPath(e)
	case *ast.CallExpr:
		// 处理方法调用如 d.Name.String()
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if innerSel, ok := sel.X.(*ast.SelectorExpr); ok {
				return m.buildSelectorPath(innerSel)
			}
		}
		// 处理函数调用如 someFunc(d.Name)
		if len(e.Args) > 0 {
			if argSel, ok := e.Args[0].(*ast.SelectorExpr); ok {
				return m.buildSelectorPath(argSel)
			}
		}
	}
	return ""
}

// analyzeReturnExpr 分析返回表达式
func (m *Mapper) analyzeReturnExpr(expr ast.Expr) error {
	// 处理取地址 &Type{}
	if unary, ok := expr.(*ast.UnaryExpr); ok && unary.Op == token.AND {
		expr = unary.X
	}

	// 获取结构体字面量
	compLit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil
	}

	// 分析结构体字面量
	return m.analyzeCompositeLit(compLit, "", "")
}

// analyzeCompositeLit 分析结构体字面量
func (m *Mapper) analyzeCompositeLit(compLit *ast.CompositeLit, targetPrefix, jsonColumn string) error {
	// 获取目标类型
	targetType := m.extractTypeName(compLit.Type)
	if targetType == "" {
		targetType = m.receiverType
	}

	// 获取类型的字段信息
	typeSpec := m.typeSpecs[targetType]
	structType, _ := m.getStructType(typeSpec)

	for _, elt := range compLit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		// 获取字段名
		fieldName := m.getKeyName(kv.Key)
		if fieldName == "" {
			continue
		}

		// 获取字段的 GORM 信息
		fieldInfo := m.getFieldInfo(structType, fieldName)

		// 分析值
		if err := m.analyzeFieldValue(fieldName, kv.Value, targetPrefix, fieldInfo, jsonColumn); err != nil {
			return err
		}
	}

	return nil
}

// getKeyName 获取键名
func (m *Mapper) getKeyName(key ast.Expr) string {
	if ident, ok := key.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

// FieldAnalysisInfo 字段分析信息
type FieldAnalysisInfo struct {
	ColumnName     string
	IsEmbedded     bool
	EmbeddedPrefix string
	IsJSONType     bool
}

// getFieldInfo 获取字段的 GORM 信息
func (m *Mapper) getFieldInfo(structType *ast.StructType, fieldName string) *FieldAnalysisInfo {
	if structType == nil {
		return &FieldAnalysisInfo{ColumnName: toSnakeCase(fieldName)}
	}

	for _, field := range structType.Fields.List {
		// 检查是否是嵌入字段
		if len(field.Names) == 0 {
			embeddedName := m.extractTypeName(field.Type)
			if embeddedName == fieldName {
				info := &FieldAnalysisInfo{IsEmbedded: true}
				// 检查 embedded 标签
				if field.Tag != nil {
					tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
					gormTag := tag.Get("gorm")
					if strings.Contains(gormTag, "embedded") {
						// 提取前缀
						for _, part := range strings.Split(gormTag, ";") {
							if strings.HasPrefix(part, "embeddedPrefix:") {
								info.EmbeddedPrefix = strings.TrimPrefix(part, "embeddedPrefix:")
							}
						}
					}
				}
				return info
			}
			continue
		}

		// 检查命名字段
		for _, name := range field.Names {
			if name.Name != fieldName {
				continue
			}

			info := &FieldAnalysisInfo{
				ColumnName: toSnakeCase(fieldName),
			}

			if field.Tag == nil {
				return info
			}

			tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))

			// 解析 GORM 标签
			gormTag := tag.Get("gorm")
			for _, part := range strings.Split(gormTag, ";") {
				if strings.HasPrefix(part, "column:") {
					info.ColumnName = strings.TrimPrefix(part, "column:")
				}
				if strings.HasPrefix(part, "embedded") {
					info.IsEmbedded = true
				}
				if strings.HasPrefix(part, "embeddedPrefix:") {
					info.EmbeddedPrefix = strings.TrimPrefix(part, "embeddedPrefix:")
				}
			}

			// 检查是否是 JSONType
			fieldType := m.getExprString(field.Type)
			if strings.Contains(fieldType, "JSONType") || strings.Contains(fieldType, "datatypes.JSON") {
				info.IsJSONType = true
			}

			return info
		}
	}

	return &FieldAnalysisInfo{ColumnName: toSnakeCase(fieldName)}
}

// getStructType 获取结构体类型
func (m *Mapper) getStructType(typeSpec *ast.TypeSpec) (*ast.StructType, bool) {
	if typeSpec == nil {
		return nil, false
	}
	structType, ok := typeSpec.Type.(*ast.StructType)
	return structType, ok
}

// analyzeFieldValue 分析字段值
func (m *Mapper) analyzeFieldValue(fieldName string, value ast.Expr, targetPrefix string, fieldInfo *FieldAnalysisInfo, jsonColumn string) error {
	targetPath := fieldName
	if targetPrefix != "" {
		targetPath = targetPrefix + "." + fieldName
	}

	// 处理嵌入字段的结构体字面量
	if fieldInfo != nil && fieldInfo.IsEmbedded {
		if compLit, ok := value.(*ast.CompositeLit); ok {
			return m.analyzeEmbeddedCompositeLit(fieldName, compLit, fieldInfo)
		}
	}

	// 处理 JSONType - datatypes.NewJSONType(...)
	if callExpr, ok := value.(*ast.CallExpr); ok {
		if m.isJSONTypeConstructor(callExpr) && len(callExpr.Args) > 0 {
			if compLit, ok := callExpr.Args[0].(*ast.CompositeLit); ok {
				return m.analyzeJSONCompositeLit(fieldName, compLit, fieldInfo)
			}
		}
	}

	// 检查是否是直接的方法调用 d.MethodName()
	if methodInfo := m.extractMethodCallInfo(value); methodInfo != nil {
		return m.analyzeMethodCallMapping(fieldName, methodInfo, fieldInfo)
	}

	// 检查是否是来自方法调用的局部变量
	if ident, ok := value.(*ast.Ident); ok {
		if methodInfo, exists := m.methodCallMap[ident.Name]; exists {
			return m.analyzeMethodCallMapping(fieldName, &methodInfo, fieldInfo)
		}
	}

	// 提取源路径和转换表达式
	sourcePath, convertExpr := m.extractSourcePath(value)
	if sourcePath == "" {
		return nil
	}

	// 确定映射类型
	columnName := fieldInfo.ColumnName
	if jsonColumn != "" {
		columnName = jsonColumn
	}

	mapping := FieldMapping2{
		SourcePath:  sourcePath,
		TargetPath:  targetPath,
		ColumnName:  columnName,
		ConvertExpr: convertExpr,
	}

	// 添加到对应的组
	m.addMapping(mapping, fieldInfo, jsonColumn)
	return nil
}

// analyzeMethodCallMapping 分析方法调用映射
func (m *Mapper) analyzeMethodCallMapping(fieldName string, methodInfo *methodCallInfo, fieldInfo *FieldAnalysisInfo) error {
	// 查找方法定义
	methods, exists := m.methodDecls[methodInfo.receiverType]
	if !exists {
		return nil
	}
	funcDecl, exists := methods[methodInfo.methodName]
	if !exists {
		return nil
	}

	// 分析方法体，提取使用的字段
	usedFields := m.extractUsedFieldsFromMethod(funcDecl)
	if len(usedFields) == 0 {
		return nil
	}

	// 创建 MethodCall 映射组
	group := MappingGroup{
		Type:        MethodCall,
		TargetField: fieldName,
		MethodName:  methodInfo.methodName,
	}

	for _, usedField := range usedFields {
		mapping := FieldMapping2{
			SourcePath: usedField,
			TargetPath: fieldName,
			ColumnName: fieldInfo.ColumnName,
		}
		group.Mappings = append(group.Mappings, mapping)
	}

	m.result.Groups = append(m.result.Groups, group)
	return nil
}

// extractUsedFieldsFromMethod 从方法体中提取使用的字段
func (m *Mapper) extractUsedFieldsFromMethod(funcDecl *ast.FuncDecl) []string {
	if funcDecl.Body == nil {
		return nil
	}

	// 获取接收者名称
	var recvName string
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		recv := funcDecl.Recv.List[0]
		if len(recv.Names) > 0 {
			recvName = recv.Names[0].Name
		}
	}
	if recvName == "" {
		return nil
	}

	// 收集使用的字段
	usedFields := make(map[string]bool)
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// 检查是否是 recv.Field 形式
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == recvName {
			usedFields[sel.Sel.Name] = true
		}
		return true
	})

	// 转换为有序列表
	var result []string
	for field := range usedFields {
		result = append(result, field)
	}

	// 按字母顺序排序以保证稳定性
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// analyzeEmbeddedCompositeLit 分析嵌入字段的结构体字面量
func (m *Mapper) analyzeEmbeddedCompositeLit(fieldName string, compLit *ast.CompositeLit, fieldInfo *FieldAnalysisInfo) error {
	// 获取嵌入类型的字段信息
	embeddedTypeName := m.extractTypeName(compLit.Type)
	embeddedTypeSpec := m.typeSpecs[embeddedTypeName]
	embeddedStructType, _ := m.getStructType(embeddedTypeSpec)

	group := MappingGroup{
		Type:        Embedded,
		TargetField: fieldName,
	}

	for _, elt := range compLit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		subFieldName := m.getKeyName(kv.Key)
		if subFieldName == "" {
			continue
		}

		sourcePath, convertExpr := m.extractSourcePath(kv.Value)
		if sourcePath == "" {
			continue
		}

		// 获取子字段的列名
		subFieldInfo := m.getFieldInfo(embeddedStructType, subFieldName)
		columnName := subFieldInfo.ColumnName
		if fieldInfo.EmbeddedPrefix != "" {
			columnName = fieldInfo.EmbeddedPrefix + columnName
		}

		mapping := FieldMapping2{
			SourcePath:  sourcePath,
			TargetPath:  fieldName + "." + subFieldName,
			ColumnName:  columnName,
			ConvertExpr: convertExpr,
		}
		group.Mappings = append(group.Mappings, mapping)
	}

	if len(group.Mappings) > 0 {
		m.result.Groups = append(m.result.Groups, group)
	}
	return nil
}

// analyzeJSONCompositeLit 分析 JSON 结构体字面量
func (m *Mapper) analyzeJSONCompositeLit(fieldName string, compLit *ast.CompositeLit, fieldInfo *FieldAnalysisInfo) error {
	group := MappingGroup{
		Type:        ManyToOne,
		TargetField: fieldName,
	}

	m.extractJSONMappings(&group, compLit, "", fieldInfo.ColumnName)

	if len(group.Mappings) > 0 {
		m.result.Groups = append(m.result.Groups, group)
	}
	return nil
}

// extractJSONMappings 递归提取 JSON 映射
func (m *Mapper) extractJSONMappings(group *MappingGroup, compLit *ast.CompositeLit, jsonPrefix string, columnName string) {
	// 获取 JSON 类型的字段信息（用于获取 json tag）
	jsonTypeName := m.extractTypeName(compLit.Type)
	jsonTypeSpec := m.typeSpecs[jsonTypeName]
	jsonStructType, _ := m.getStructType(jsonTypeSpec)

	for _, elt := range compLit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}

		subFieldName := m.getKeyName(kv.Key)
		if subFieldName == "" {
			continue
		}

		// 获取 JSON 字段名（从 json tag）
		jsonFieldName := m.getJSONTagName(jsonStructType, subFieldName)
		jsonPath := jsonFieldName
		if jsonPrefix != "" {
			jsonPath = jsonPrefix + "." + jsonFieldName
		}

		// 检查是否是嵌套结构体
		if nestedCompLit, ok := kv.Value.(*ast.CompositeLit); ok {
			m.extractJSONMappings(group, nestedCompLit, jsonPath, columnName)
			continue
		}

		// 提取源路径
		sourcePath, convertExpr := m.extractSourcePath(kv.Value)
		if sourcePath == "" {
			continue
		}

		mapping := FieldMapping2{
			SourcePath:  sourcePath,
			TargetPath:  group.TargetField,
			ColumnName:  columnName,
			JSONPath:    jsonPath,
			ConvertExpr: convertExpr,
		}
		group.Mappings = append(group.Mappings, mapping)
	}
}

// getJSONTagName 获取字段的 JSON 标签名
func (m *Mapper) getJSONTagName(structType *ast.StructType, fieldName string) string {
	if structType == nil {
		return toSnakeCase(fieldName)
	}

	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			if name.Name != fieldName {
				continue
			}
			if field.Tag == nil {
				return toSnakeCase(fieldName)
			}
			tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
			jsonTag := tag.Get("json")
			if jsonTag == "" || jsonTag == "-" {
				return toSnakeCase(fieldName)
			}
			parts := strings.SplitN(jsonTag, ",", 2)
			return parts[0]
		}
	}
	return toSnakeCase(fieldName)
}

// isJSONTypeConstructor 检查是否是 JSONType 构造函数
func (m *Mapper) isJSONTypeConstructor(callExpr *ast.CallExpr) bool {
	switch fun := callExpr.Fun.(type) {
	case *ast.SelectorExpr:
		// datatypes.NewJSONType
		if fun.Sel.Name == "NewJSONType" {
			return true
		}
	case *ast.Ident:
		// NewJSONType
		if fun.Name == "NewJSONType" {
			return true
		}
	}
	return false
}

// extractSourcePath 提取源路径和转换表达式
func (m *Mapper) extractSourcePath(expr ast.Expr) (sourcePath, convertExpr string) {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		// d.Field 或 d.Struct.Field
		return m.buildSelectorPath(e), ""

	case *ast.CallExpr:
		// d.Field.Unix() 或 someFunc(d.Field)
		return m.extractFromCallExpr(e)

	case *ast.Ident:
		// 局部变量 - 查找变量映射表
		if path, exists := m.varMap[e.Name]; exists {
			return path, ""
		}
		return "", ""
	}
	return "", ""
}

// buildSelectorPath 构建选择器路径
func (m *Mapper) buildSelectorPath(expr *ast.SelectorExpr) string {
	var parts []string
	current := expr

	for {
		parts = append([]string{current.Sel.Name}, parts...)

		switch x := current.X.(type) {
		case *ast.SelectorExpr:
			current = x
		case *ast.Ident:
			// 检查是否是参数名
			if x.Name == m.paramName {
				return strings.Join(parts, ".")
			}
			// 其他标识符，可能是包名
			return ""
		default:
			return ""
		}
	}
}

// extractFromCallExpr 从调用表达式提取源路径
func (m *Mapper) extractFromCallExpr(callExpr *ast.CallExpr) (sourcePath, convertExpr string) {
	switch fun := callExpr.Fun.(type) {
	case *ast.SelectorExpr:
		// 方法调用：d.Field.Unix()
		methodName := fun.Sel.Name

		// 检查是否是在字段上调用方法
		if selector, ok := fun.X.(*ast.SelectorExpr); ok {
			path := m.buildSelectorPath(selector)
			if path != "" {
				return path, "." + methodName + "()"
			}
		}

		// 检查是否是包函数调用：decimal.NewFromBigInt(d.Field, 0)
		if len(callExpr.Args) > 0 {
			if argSelector, ok := callExpr.Args[0].(*ast.SelectorExpr); ok {
				path := m.buildSelectorPath(argSelector)
				if path != "" {
					// 获取包名
					if pkgIdent, ok := fun.X.(*ast.Ident); ok {
						return path, fmt.Sprintf("%s.%s(...)", pkgIdent.Name, methodName)
					}
				}
			}
		}

	case *ast.Ident:
		// 函数调用：someFunc(d.Field)
		if len(callExpr.Args) > 0 {
			if argSelector, ok := callExpr.Args[0].(*ast.SelectorExpr); ok {
				path := m.buildSelectorPath(argSelector)
				if path != "" {
					return path, fun.Name + "(...)"
				}
			}
		}
	}

	return "", ""
}

// addMapping 添加映射到结果
func (m *Mapper) addMapping(mapping FieldMapping2, fieldInfo *FieldAnalysisInfo, jsonColumn string) {
	// 检查是否是一对多映射（源路径包含点）
	if strings.Contains(mapping.SourcePath, ".") && jsonColumn == "" && !fieldInfo.IsEmbedded && !fieldInfo.IsJSONType {
		// 一对多映射
		parts := strings.SplitN(mapping.SourcePath, ".", 2)
		sourceField := parts[0]

		// 查找或创建组
		for i := range m.result.Groups {
			if m.result.Groups[i].Type == OneToMany && m.result.Groups[i].SourceField == sourceField {
				m.result.Groups[i].Mappings = append(m.result.Groups[i].Mappings, mapping)
				return
			}
		}

		// 创建新组
		group := MappingGroup{
			Type:        OneToMany,
			SourceField: sourceField,
			Mappings:    []FieldMapping2{mapping},
		}
		m.result.Groups = append(m.result.Groups, group)
		return
	}

	// 一对一映射
	for i := range m.result.Groups {
		if m.result.Groups[i].Type == OneToOne {
			m.result.Groups[i].Mappings = append(m.result.Groups[i].Mappings, mapping)
			return
		}
	}

	// 创建一对一组
	group := MappingGroup{
		Type:     OneToOne,
		Mappings: []FieldMapping2{mapping},
	}
	m.result.Groups = append(m.result.Groups, group)
}

// flattenMappings 扁平化所有映射
func (m *Mapper) flattenMappings() {
	for _, group := range m.result.Groups {
		m.result.AllMappings = append(m.result.AllMappings, group.Mappings...)
	}
}

// DebugMode 启用调试模式
// 可以通过设置环境变量 AUTOMAP_DEBUG=1 来启用
var DebugMode = os.Getenv("AUTOMAP_DEBUG") == "1"

// collectTargetColumns 收集目标类型（PO）的所有数据库列名
func (m *Mapper) collectTargetColumns() {
	// 使用 structparse 解析目标类型，它能正确处理外部包的嵌入类型
	// 首先尝试在当前文件中查找
	structInfo, err := structparse.ParseStruct(m.filePath, m.receiverType)
	if err != nil {
		if DebugMode {
			fmt.Printf("[DEBUG] structparse.ParseStruct failed for %s in %s: %v\n", m.receiverType, m.filePath, err)
			fmt.Println("[DEBUG] Trying to find struct in same directory...")
		}
		// 在同目录下的其他文件中查找
		structInfo, err = m.findStructInSameDirectory()
		if err != nil {
			if DebugMode {
				fmt.Printf("[DEBUG] Failed to find struct %s in same directory: %v\n", m.receiverType, err)
				fmt.Println("[DEBUG] Falling back to collectTargetColumnsSimple")
			}
			// 解析失败时回退到简单方式
			m.collectTargetColumnsSimple()
			return
		}
	}

	if DebugMode {
		fmt.Printf("[DEBUG] structparse.ParseStruct succeeded for %s, found %d fields:\n", m.receiverType, len(structInfo.Fields))
		for i, field := range structInfo.Fields {
			fmt.Printf("[DEBUG]   %d. Name=%s, Type=%s, Tag=%s, SourceType=%s, EmbeddedPrefix=%s\n",
				i+1, field.Name, field.Type, field.Tag, field.SourceType, field.EmbeddedPrefix)
		}
	}

	// 从解析结果中提取列名
	for _, field := range structInfo.Fields {
		columnName := gormparse.ExtractColumnNameWithPrefix(field.Name, field.Tag, field.EmbeddedPrefix)
		m.result.TargetColumns = append(m.result.TargetColumns, columnName)
		if DebugMode {
			fmt.Printf("[DEBUG] Column: %s (from field %s)\n", columnName, field.Name)
		}
	}

	if DebugMode {
		fmt.Printf("[DEBUG] Total target columns: %d\n", len(m.result.TargetColumns))
	}
}

// findStructInSameDirectory 在同目录下的其他Go文件中查找结构体
func (m *Mapper) findStructInSameDirectory() (*structparse.StructInfo, error) {
	dir := filepath.Dir(m.filePath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		filePath := filepath.Join(dir, entry.Name())
		// 跳过当前文件，因为已经尝试过了
		if filePath == m.filePath {
			continue
		}

		structInfo, err := structparse.ParseStruct(filePath, m.receiverType)
		if err == nil {
			if DebugMode {
				fmt.Printf("[DEBUG] Found struct %s in file: %s\n", m.receiverType, filePath)
			}
			return structInfo, nil
		}
	}

	return nil, fmt.Errorf("在目录 %s 中未找到结构体 %s", dir, m.receiverType)
}

// collectTargetColumnsSimple 简单方式收集列名（仅处理当前文件中的类型）
func (m *Mapper) collectTargetColumnsSimple() {
	typeSpec := m.typeSpecs[m.receiverType]
	if typeSpec == nil {
		return
	}

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return
	}

	m.result.TargetColumns = m.collectStructColumns(structType, "")
}

// collectStructColumns 递归收集结构体的所有列名
func (m *Mapper) collectStructColumns(structType *ast.StructType, prefix string) []string {
	var columns []string

	for _, field := range structType.Fields.List {
		// 处理嵌入字段
		if len(field.Names) == 0 {
			embeddedTypeName := m.extractTypeName(field.Type)
			embeddedTypeSpec := m.typeSpecs[embeddedTypeName]

			// 获取嵌入字段的前缀
			embeddedPrefix := ""
			if field.Tag != nil {
				tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
				gormTag := tag.Get("gorm")
				for _, part := range strings.Split(gormTag, ";") {
					if strings.HasPrefix(part, "embeddedPrefix:") {
						embeddedPrefix = strings.TrimPrefix(part, "embeddedPrefix:")
					}
				}
			}

			if embeddedTypeSpec != nil {
				if embeddedStructType, ok := embeddedTypeSpec.Type.(*ast.StructType); ok {
					subColumns := m.collectStructColumns(embeddedStructType, prefix+embeddedPrefix)
					columns = append(columns, subColumns...)
				}
			}
			continue
		}

		// 处理命名字段
		for _, name := range field.Names {
			fieldName := name.Name
			// 跳过非导出字段
			if len(fieldName) == 0 || fieldName[0] < 'A' || fieldName[0] > 'Z' {
				continue
			}

			// 获取列名
			columnName := toSnakeCase(fieldName)
			if field.Tag != nil {
				tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
				gormTag := tag.Get("gorm")
				for _, part := range strings.Split(gormTag, ";") {
					if strings.HasPrefix(part, "column:") {
						columnName = strings.TrimPrefix(part, "column:")
					}
				}
			}

			columns = append(columns, prefix+columnName)
		}
	}

	return columns
}

// getExprString 获取表达式的字符串表示
func (m *Mapper) getExprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return m.getExprString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + m.getExprString(e.X)
	case *ast.IndexExpr:
		return m.getExprString(e.X) + "[" + m.getExprString(e.Index) + "]"
	case *ast.ArrayType:
		return "[]" + m.getExprString(e.Elt)
	}
	return ""
}

// toSnakeCase 驼峰转蛇形（使用 gormparse 的实现，正确处理 ID 等缩写）
func toSnakeCase(s string) string {
	return gormparse.ToSnakeCase(s)
}
