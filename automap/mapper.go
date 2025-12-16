package automap

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
)

// DebugMode 启用调试模式
// 可以通过设置环境变量 AUTOMAP_DEBUG=1 来启用
var DebugMode = os.Getenv("AUTOMAP_DEBUG") == "1"

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
		FuncName:             funcName,
		ReceiverType:         receiverType,
		TargetType:           receiverType,
		TargetFieldPositions: make(map[string]int),
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

	// 收集目标类型的所有列名（同时填充字段位置映射）
	m.collectTargetColumns()

	// 根据字段位置设置映射的 FieldPosition
	m.setFieldPositions()

	return m.result, nil
}

// parseFile 解析源文件
func (m *Mapper) parseFile() error {
	m.fset = token.NewFileSet()
	var err error
	m.file, err = parser.ParseFile(m.fset, m.filePath, nil, parser.ParseComments)
	return err
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

	recvType := extractTypeName(funcDecl.Recv.List[0].Type)
	return recvType == m.receiverType
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
	typeName, pkgName := extractTypeNameWithPackage(param.Type)
	m.sourceType = typeName
	m.result.SourceType = typeName
	m.result.SourceTypePackage = pkgName

	// 如果有包名，查找对应的导入路径
	if pkgName != "" {
		m.result.SourceTypeImportPath = m.resolveImportPath(pkgName)
	}
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
	targetType := extractTypeName(compLit.Type)
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
	ColumnName       string
	IsEmbedded       bool
	EmbeddedPrefix   string
	EmbeddedTypeName string // 嵌入字段的类型名
	IsJSONType       bool
}

// getFieldInfo 获取字段的 GORM 信息
func (m *Mapper) getFieldInfo(structType *ast.StructType, fieldName string) *FieldAnalysisInfo {
	if structType == nil {
		return &FieldAnalysisInfo{ColumnName: toSnakeCase(fieldName)}
	}

	for _, field := range structType.Fields.List {
		// 检查是否是嵌入字段
		if len(field.Names) == 0 {
			embeddedName := extractTypeName(field.Type)
			if embeddedName == fieldName {
				info := &FieldAnalysisInfo{IsEmbedded: true, EmbeddedTypeName: embeddedName}
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
				if strings.HasPrefix(part, "embedded") && !strings.HasPrefix(part, "embeddedPrefix:") {
					info.IsEmbedded = true
					// 获取嵌入字段的类型名
					info.EmbeddedTypeName = extractTypeName(field.Type)
				}
				if strings.HasPrefix(part, "embeddedPrefix:") {
					info.EmbeddedPrefix = strings.TrimPrefix(part, "embeddedPrefix:")
				}
			}

			// 检查是否是 JSONType
			fieldType := getExprString(field.Type)
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
		// 处理嵌入字段的直接赋值（非结构体字面量）
		// 例如: Account: d.Account 或 Account: d.Account.ToColumns()
		// 这是 EmbeddedOneToMany 映射
		return m.analyzeEmbeddedOneToManyMapping(fieldName, value, fieldInfo)
	}

	// 处理 JSONType - datatypes.NewJSONType(...)
	if callExpr, ok := value.(*ast.CallExpr); ok {
		if m.isJSONTypeConstructor(callExpr) && len(callExpr.Args) > 0 {
			if compLit, ok := callExpr.Args[0].(*ast.CompositeLit); ok {
				return m.analyzeJSONCompositeLit(fieldName, compLit, fieldInfo)
			}
		}

		// 处理 JSONSlice - datatypes.NewJSONSlice(...)
		// 支持三种模式：
		// 1. datatypes.NewJSONSlice(lo.Map(entity.Field, func...)) - lo.Map 转换（字段访问）
		// 2. datatypes.NewJSONSlice(lo.Map(entity.GetMethod(), func...)) - lo.Map 转换（方法调用）
		// 3. datatypes.NewJSONSlice(entity.Field) - 直接传入字段
		if m.isJSONSliceConstructor(callExpr) && len(callExpr.Args) > 0 {
			arg := callExpr.Args[0]

			// 情况1和2: lo.Map 模式
			if innerCall, ok := arg.(*ast.CallExpr); ok {
				// 情况1: lo.Map(entity.Field, func...) - 直接字段访问
				if sourcePath, ok := m.extractLoMapSource(innerCall); ok && sourcePath != "" {
					mapping := FieldMapping2{
						SourcePath: sourcePath,
						TargetPath: targetPath,
						ColumnName: fieldInfo.ColumnName,
					}
					m.addMapping(mapping, fieldInfo, jsonColumn)
					return nil
				}

				// 情况2: lo.Map(entity.GetMethod(), func...) - 方法调用
				if methodInfo := m.extractLoMapMethodCall(innerCall); methodInfo != nil {
					return m.analyzeMethodCallMapping(fieldName, methodInfo, fieldInfo)
				}
			}

			// 情况3: 直接传入字段 datatypes.NewJSONSlice(entity.Field)
			if argSelector, ok := arg.(*ast.SelectorExpr); ok {
				sourcePath := m.buildSelectorPath(argSelector)
				if sourcePath != "" {
					mapping := FieldMapping2{
						SourcePath: sourcePath,
						TargetPath: targetPath,
						ColumnName: fieldInfo.ColumnName,
					}
					m.addMapping(mapping, fieldInfo, jsonColumn)
					return nil
				}
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
// 支持两种情况：
// 1. 方法在本包中定义：分析方法体，提取使用的字段
// 2. 方法在外部包中定义：从方法名推断字段名（如 GetExchangeRules -> ExchangeRules）
func (m *Mapper) analyzeMethodCallMapping(fieldName string, methodInfo *methodCallInfo, fieldInfo *FieldAnalysisInfo) error {
	// 查找方法定义
	methods, exists := m.methodDecls[methodInfo.receiverType]
	if exists {
		if funcDecl, methodExists := methods[methodInfo.methodName]; methodExists {
			// 方法在本包中定义，分析方法体
			usedFields := m.extractUsedFieldsFromMethod(funcDecl)
			if len(usedFields) > 0 {
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
		}
	}

	// 方法在外部包中定义或无法分析，尝试从方法名推断字段名
	// GetExchangeRules -> ExchangeRules
	sourcePath := inferFieldNameFromMethod(methodInfo.methodName)
	if sourcePath != "" {
		mapping := FieldMapping2{
			SourcePath: sourcePath,
			TargetPath: fieldName,
			ColumnName: fieldInfo.ColumnName,
		}
		m.addMapping(mapping, fieldInfo, "")
	}

	return nil
}

// inferFieldNameFromMethod 从方法名推断字段名
// GetExchangeRules -> ExchangeRules
// GetFoo -> Foo
func inferFieldNameFromMethod(methodName string) string {
	if strings.HasPrefix(methodName, "Get") && len(methodName) > 3 {
		return methodName[3:]
	}
	return methodName
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

	case *ast.StarExpr:
		// 指针解引用: *d.Field 或 *entity.Field
		return m.extractSourcePath(e.X)

	case *ast.UnaryExpr:
		// 一元表达式: &d.Field（取地址）或其他一元操作
		return m.extractSourcePath(e.X)
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

// setFieldPositions 根据字段位置设置映射的 FieldPosition
func (m *Mapper) setFieldPositions() {
	// 设置每个映射的 FieldPosition
	for i := range m.result.Groups {
		group := &m.result.Groups[i]
		minPos := -1

		for j := range group.Mappings {
			mapping := &group.Mappings[j]
			if pos, ok := m.result.TargetFieldPositions[mapping.ColumnName]; ok {
				mapping.FieldPosition = pos
				if minPos == -1 || pos < minPos {
					minPos = pos
				}
			}
		}

		// 组的位置取其所有映射的最小位置
		if minPos >= 0 {
			group.FieldPosition = minPos
		}
	}

	// 同时更新 AllMappings 中的 FieldPosition
	for i := range m.result.AllMappings {
		mapping := &m.result.AllMappings[i]
		if pos, ok := m.result.TargetFieldPositions[mapping.ColumnName]; ok {
			mapping.FieldPosition = pos
		}
	}
}
