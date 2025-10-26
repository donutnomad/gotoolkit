package automap

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// TypeResolver 类型解析器，支持跨包类型查找
type TypeResolver struct {
	fset        *token.FileSet
	cache       map[string]*TypeInfo // 类型缓存
	importMap   map[string]string    // import映射
	goMod       string               // go.mod文件路径
	currentFile string
}

// NewTypeResolver 创建类型解析器
func NewTypeResolver() *TypeResolver {
	return &TypeResolver{
		fset:      token.NewFileSet(),
		cache:     make(map[string]*TypeInfo),
		importMap: make(map[string]string),
	}
}

func (tr *TypeResolver) ResolveTypeCurrent(typeInfo *TypeInfo) error {
	return tr.ResolveType(typeInfo, tr.currentFile)
}

// ResolveType 解析类型详细信息
func (tr *TypeResolver) ResolveType(typeInfo *TypeInfo, currentFile string) error {
	// 检查缓存
	cacheKey := typeInfo.FullName
	if cached, exists := tr.cache[cacheKey]; exists {
		*typeInfo = *cached
		return nil
	}

	// 对于当前包的类型，直接使用当前文件目录
	var pkgPath string
	var err error

	if typeInfo.Package == "" || typeInfo.Package == "automap" {
		pkgPath = filepath.Dir(currentFile)
	} else {
		// 对于外部包，首先确保currentFile是有效的文件路径
		var parseFile string
		if currentFile == "" || tr.isDirectory(currentFile) {
			// 如果currentFile是空或目录，尝试从其他地方获取文件路径
			// 对于这种情况，我们跳过import解析，直接使用包路径推断
			parseFile = ""
		} else {
			parseFile = currentFile
		}

		// 只有在有有效文件时才解析imports
		if parseFile != "" {
			if err := tr.parseImports(parseFile); err != nil {
				return fmt.Errorf("解析imports失败: %w", err)
			}
		}

		// 根据包名找到实际路径
		pkgPath, err = tr.resolvePackagePath(typeInfo.Package, parseFile)
		if err != nil {
			return fmt.Errorf("解析包路径失败: %w", err)
		}
	}

	// 查找类型定义
	typeSpec, filePath, err := tr.findTypeDefinition(pkgPath, typeInfo.Name)
	if err != nil {
		return fmt.Errorf("查找类型定义失败: %w", err)
	}

	// 更新类型信息
	typeInfo.FilePath = filePath

	// 解析结构体字段
	if structType, ok := typeSpec.Type.(*ast.StructType); ok {
		fields, err := tr.parseStructFields(structType)
		if err != nil {
			return fmt.Errorf("解析字段失败: %w", err)
		}
		typeInfo.Fields = fields
	}

	// 解析方法
	methods, err := tr.parseTypeMethods(typeSpec.Name.Name, pkgPath)
	if err != nil {
		return fmt.Errorf("解析方法失败: %w", err)
	}
	typeInfo.Methods = methods

	// 缓存结果
	cachedCopy := *typeInfo
	tr.cache[cacheKey] = &cachedCopy

	return nil
}

// isDirectory 检查路径是否为目录
func (tr *TypeResolver) isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// parseImports 解析文件的import信息
func (tr *TypeResolver) parseImports(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("文件路径为空")
	}

	// 如果filePath是目录，尝试在该目录中查找Go文件
	if tr.isDirectory(filePath) {
		// 查找目录中的第一个.go文件
		files, err := os.ReadDir(filePath)
		if err != nil {
			return fmt.Errorf("读取目录失败: %w", err)
		}

		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".go") {
				filePath = filepath.Join(filePath, file.Name())
				break
			}
		}

		// 如果没有找到.go文件，返回错误
		if tr.isDirectory(filePath) {
			return fmt.Errorf("目录中未找到Go文件: %s", filePath)
		}
	}

	file, err := parser.ParseFile(tr.fset, filePath, nil, parser.ImportsOnly)
	if err != nil {
		return fmt.Errorf("解析文件失败: %w", err)
	}

	// 清空之前的import映射
	tr.importMap = make(map[string]string)

	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")

		if imp.Name != nil {
			// 有别名的情况
			tr.importMap[imp.Name.Name] = importPath
		} else {
			// 使用包名的最后一部分作为默认名称
			parts := strings.Split(importPath, "/")
			defaultName := parts[len(parts)-1]
			tr.importMap[defaultName] = importPath
		}
	}

	return nil
}

// findImportPathByAlias 通过别名反向查找import路径
func (tr *TypeResolver) findImportPathByAlias(alias string) string {
	if importPath, exists := tr.importMap[alias]; exists {
		return importPath
	}
	return ""
}

// resolvePackagePath 解析包路径
func (tr *TypeResolver) resolvePackagePath(packageName, currentFile string) (string, error) {
	if packageName == "" || packageName == "." {
		// 当前包
		return filepath.Dir(currentFile), nil
	}

	// 对于相同包名的情况，先尝试当前包
	dir := filepath.Dir(currentFile)
	if pkgName := tr.findPackageNameInDir(dir, packageName); pkgName != "" {
		return dir, nil
	}

	// 检查是否在import映射中
	if importPath, exists := tr.importMap[packageName]; exists {
		// 将import路径转换为文件系统路径
		return tr.importPathToFilePath(importPath, currentFile)
	}

	// 对于别名的import，需要反向查找
	if reversePath := tr.findImportPathByAlias(packageName); reversePath != "" {
		return tr.importPathToFilePath(reversePath, currentFile)
	}

	// 处理特殊包名映射（如 domain -> 实际路径）
	if mappedPath, exists := tr.getSpecialPackageMapping(packageName, currentFile); exists {
		return mappedPath, nil
	}

	// 尝试从go.mod解析模块路径
	goModPath, err := tr.findGoModFile(currentFile)
	if err == nil {
		moduleRoot := filepath.Dir(goModPath)
		moduleName, err := tr.getModuleName(goModPath)
		if err == nil {
			// 尝试在当前模块中查找包
			pkgPath := filepath.Join(moduleRoot, strings.ReplaceAll(packageName, ".", "/"))
			if _, err := os.Stat(pkgPath); err == nil {
				return pkgPath, nil
			}

			// 如果import路径以模块名开头，尝试解析相对路径
			if strings.HasPrefix(packageName, moduleName) {
				relativePath := strings.TrimPrefix(packageName, moduleName)
				relativePath = strings.TrimPrefix(relativePath, "/")
				relativePath = strings.ReplaceAll(relativePath, ".", "/")
				return filepath.Join(moduleRoot, relativePath), nil
			}
		}
	}

	// 标准库包检查
	if tr.isStandardLibrary(packageName) {
		return "", fmt.Errorf("暂不支持标准库类型: %s", packageName)
	}

	return "", fmt.Errorf("未找到包: %s", packageName)
}

// importPathToFilePath 将import路径转换为文件系统路径
func (tr *TypeResolver) importPathToFilePath(importPath, currentFile string) (string, error) {
	// 检查是否为相对路径
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		baseDir := filepath.Dir(currentFile)
		path, _ := filepath.Abs(filepath.Join(baseDir, importPath))
		return path, nil
	}

	// 查找go.mod文件
	goModPath, err := tr.findGoModFile(currentFile)
	if err != nil {
		return "", fmt.Errorf("查找go.mod失败: %w", err)
	}

	// 获取模块根目录
	moduleRoot := filepath.Dir(goModPath)

	// 如果import路径以模块名开头，去掉模块名部分
	moduleName, err := tr.getModuleName(goModPath)
	if err != nil {
		return "", fmt.Errorf("获取模块名失败: %w", err)
	}

	if strings.HasPrefix(importPath, moduleName) {
		relativePath := strings.TrimPrefix(importPath, moduleName)
		relativePath = strings.TrimPrefix(relativePath, "/")
		return filepath.Join(moduleRoot, relativePath), nil
	}

	// 尝试直接在vendor目录中查找
	vendorPath := filepath.Join(moduleRoot, "vendor", importPath)
	if _, err := os.Stat(vendorPath); err == nil {
		return vendorPath, nil
	}

	return "", fmt.Errorf("无法解析import路径: %s", importPath)
}

// findGoModFile 查找go.mod文件
func (tr *TypeResolver) findGoModFile(startPath string) (string, error) {
	dir := filepath.Dir(startPath)

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return goModPath, nil
		}

		// 到达根目录
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("未找到go.mod文件")
}

// getModuleName 获取模块名
func (tr *TypeResolver) getModuleName(goModPath string) (string, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 简单解析go.mod文件获取模块名
	// TODO: 使用更robust的解析方式
	buf := make([]byte, 1024)
	n, err := file.Read(buf)
	if err != nil {
		return "", err
	}

	content := string(buf[:n])
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	return "", fmt.Errorf("未找到模块名")
}

// getSpecialPackageMapping 处理特殊包名映射
func (tr *TypeResolver) getSpecialPackageMapping(packageName, currentFile string) (string, bool) {
	// 从当前文件向上查找go.mod，确定项目根目录
	goModPath, err := tr.findGoModFile(currentFile)
	if err != nil {
		return "", false
	}

	projectRoot := filepath.Dir(goModPath)

	// 常见的包名映射规则
	switch packageName {
	case "domain":
		// 尝试在常见的目录中查找domain相关的包
		possiblePaths := []string{
			filepath.Join(projectRoot, "internal", "app", "launchpad", "biz", "listing"),
			filepath.Join(projectRoot, "internal", "domain"),
			filepath.Join(projectRoot, "domain"),
			filepath.Join(projectRoot, "pkg", "domain"),
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				return path, true
			}
		}
	}

	return "", false
}

// isStandardLibrary 检查是否为标准库
func (tr *TypeResolver) isStandardLibrary(packageName string) bool {
	// 标准库包列表（简化版）
	standardLibs := map[string]bool{
		"fmt": true, "strings": true, "time": true, "context": true,
		"io": true, "os": true, "net": true, "http": true,
		"encoding/json": true, "database/sql": true,
	}

	return standardLibs[packageName]
}

// findPackageNameInDir 在目录中查找指定包名
func (tr *TypeResolver) findPackageNameInDir(dir, targetPkgName string) string {
	// 读取目录中的所有 .go 文件
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	// 检查每个 .go 文件的包声明
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			filePath := filepath.Join(dir, entry.Name())
			file, err := parser.ParseFile(tr.fset, filePath, nil, parser.PackageClauseOnly)
			if err != nil {
				continue
			}
			if file.Name.Name == targetPkgName {
				return targetPkgName
			}
		}
	}

	return ""
}

// parseDirectory 解析目录中的所有 .go 文件
func (tr *TypeResolver) parseDirectory(dirPath string) ([]*ast.File, error) {
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
			file, err := parser.ParseFile(tr.fset, filePath, nil, parser.AllErrors)
			if err != nil {
				// 如果单个文件解析失败，继续解析其他文件
				continue
			}
			files = append(files, file)
		}
	}

	return files, nil
}

// findTypeDefinition 查找类型定义
func (tr *TypeResolver) findTypeDefinition(pkgPath, typeName string) (*ast.TypeSpec, string, error) {
	// 解析目录中的所有文件
	files, err := tr.parseDirectory(pkgPath)
	if err != nil {
		return nil, "", err
	}

	for _, file := range files {
		for _, decl := range file.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == typeName {
						filePath := tr.fset.Position(file.Pos()).Filename
						return typeSpec, filePath, nil
					}
				}
			}
		}
	}

	return nil, "", fmt.Errorf("未找到类型定义: %s", typeName)
}

// parseStructFields 解析结构体字段
func (tr *TypeResolver) parseStructFields(structType *ast.StructType) ([]FieldInfo, error) {
	var fields []FieldInfo

	for _, field := range structType.Fields.List {
		fieldInfos, err := tr.parseField(field)
		if err != nil {
			return nil, err
		}
		fields = append(fields, fieldInfos...)
	}

	return fields, nil
}

// parseField 解析字段
func (tr *TypeResolver) parseField(field *ast.Field) ([]FieldInfo, error) {
	var fieldNames []string
	if len(field.Names) > 0 {
		for _, name := range field.Names {
			fieldNames = append(fieldNames, name.Name)
		}
	} else {
		// 嵌入字段
		fieldNames = append(fieldNames, "")
	}

	var result []FieldInfo

	for _, fieldName := range fieldNames {
		fieldType := tr.getFieldType(field.Type)
		gormTag := tr.extractGormTag(field.Tag)
		jsonTag := tr.extractJsonTag(field.Tag)
		columnName := tr.extractColumnName(gormTag)

		// 检查是否为JSONType
		isJSONType := tr.isJSONType(field.Type)
		var jsonFields []JSONFieldInfo
		if isJSONType {
			jsonFields = tr.parseJSONFields(field.Type)
		}

		fieldInfo := FieldInfo{
			Name:       fieldName,
			Type:       fieldType,
			GormTag:    gormTag,
			JsonTag:    jsonTag,
			ColumnName: columnName,
			IsJSONType: isJSONType,
			JSONFields: jsonFields,
			IsEmbedded: len(field.Names) == 0,
			ASTField:   field,
		}

		result = append(result, fieldInfo)
	}

	return result, nil
}

// getFieldType 获取字段类型字符串
func (tr *TypeResolver) getFieldType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + tr.getFieldType(t.X)
	case *ast.SelectorExpr:
		return tr.getFieldType(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + tr.getFieldType(t.Elt)
	case *ast.MapType:
		return "map[" + tr.getFieldType(t.Key) + "]" + tr.getFieldType(t.Value)
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// extractGormTag 提取GORM标签
func (tr *TypeResolver) extractGormTag(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}

	tagStr := strings.Trim(tag.Value, "`")
	if !strings.HasPrefix(tagStr, "gorm:") {
		return ""
	}

	// 去除gorm:前缀和引号
	gormContent := strings.TrimPrefix(tagStr, "gorm:")
	return strings.Trim(gormContent, `"`)
}

// extractGormTag 提取json标签
func (tr *TypeResolver) extractJsonTag(tag *ast.BasicLit) string {
	if tag == nil {
		return ""
	}

	tagStr := strings.Trim(tag.Value, "`")
	if !strings.HasPrefix(tagStr, "json:") {
		return ""
	}

	// 去除json:前缀和引号
	gormContent := strings.TrimPrefix(tagStr, "json:")
	return strings.Trim(gormContent, `"`)
}

// extractColumnName 提取列名
func (tr *TypeResolver) extractColumnName(gormTag string) string {
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

// isJSONType 检查是否为JSONType或JSONSlice
func (tr *TypeResolver) isJSONType(expr ast.Expr) bool {
	if selectorExpr, ok := expr.(*ast.SelectorExpr); ok {
		if x, ok := selectorExpr.X.(*ast.Ident); ok {
			return x.Name == "datatypes" && (selectorExpr.Sel.Name == "JSONType" || selectorExpr.Sel.Name == "JSONSlice")
		}
	}

	// 检查泛型形式：datatypes.JSONType[B_Token] 或 datatypes.JSONSlice[ExchangeRule]
	if indexExpr, ok := expr.(*ast.IndexExpr); ok {
		if selectorExpr, ok := indexExpr.X.(*ast.SelectorExpr); ok {
			if x, ok := selectorExpr.X.(*ast.Ident); ok {
				return x.Name == "datatypes" && (selectorExpr.Sel.Name == "JSONType" || selectorExpr.Sel.Name == "JSONSlice")
			}
		}
	}

	return false
}

// parseJSONFields 解析JSON字段
func (tr *TypeResolver) parseJSONFields(expr ast.Expr) []JSONFieldInfo {
	// 检查泛型形式：datatypes.JSONType[T] 或 datatypes.JSONSlice[T]
	if indexExpr, ok := expr.(*ast.IndexExpr); ok {
		// 获取泛型参数
		typeArg := indexExpr.Index
		typeStr := tr.getFieldType(typeArg)

		// 检查是否是JSONSlice类型
		if selectorExpr, ok := indexExpr.X.(*ast.SelectorExpr); ok {
			if x, ok := selectorExpr.X.(*ast.Ident); ok && x.Name == "datatypes" && selectorExpr.Sel.Name == "JSONSlice" {
				// JSONSlice类型应该作为整体处理，不解析内部字段
				return []JSONFieldInfo{}
			}
		}

		// 对于JSONType类型，继续解析字段
		// 检查是否是简单类型（不是结构体）
		if tr.isSimpleType(typeStr) {
			// 对于简单类型如 []string, int, string 等，不解析为JSON字段
			return []JSONFieldInfo{}
		}

		// 对于复杂类型，尝试解析结构体字段
		// 尝试解析类型定义
		structFields := tr.parseStructFieldsFromType(typeStr)
		if len(structFields) > 0 {
			return structFields
		}
	}

	// 对于非泛型形式，返回空
	return []JSONFieldInfo{}
}

// parseStructFieldsFromType 从类型字符串解析结构体字段
func (tr *TypeResolver) parseStructFieldsFromType(typeStr string) []JSONFieldInfo {
	// 构建缓存键，包含可能的包前缀
	cacheKeys := []string{typeStr, "domain." + typeStr, "automap." + typeStr}

	// 首先尝试从当前已解析的类型中查找
	for _, cacheKey := range cacheKeys {
		if cachedType, exists := tr.cache[cacheKey]; exists {
			return tr.convertFieldsToJSONFields(cachedType.Fields)
		}
	}

	// 如果缓存中没有，尝试从常见的包路径中解析
	typeInfo := &TypeInfo{
		Name:     typeStr,
		FullName: typeStr,
	}

	// 尝试从domain包解析
	if err := tr.resolveTypeFromDomain(typeInfo); err == nil {
		// 缓存解析结果
		cacheKey := "domain." + typeStr
		cachedCopy := *typeInfo
		tr.cache[cacheKey] = &cachedCopy
		return tr.convertFieldsToJSONFields(typeInfo.Fields)
	}

	// 尝试从当前包解析
	if err := tr.resolveTypeFromCurrentPackage(typeInfo); err == nil {
		// 缓存解析结果
		cacheKey := "automap." + typeStr
		cachedCopy := *typeInfo
		tr.cache[cacheKey] = &cachedCopy
		return tr.convertFieldsToJSONFields(typeInfo.Fields)
	}
	return []JSONFieldInfo{}
}

// isSimpleType 检查是否为简单类型（非结构体）
func (tr *TypeResolver) isSimpleType(typeStr string) bool {
	// 基本类型
	simpleTypes := map[string]bool{
		"string": true, "int": true, "int64": true, "float64": true, "bool": true,
		"[]string": true, "[]int": true, "[]int64": true, "[]float64": true, "[]bool": true,
		"interface{}": true, "any": true,
	}

	// 检查是否为已知的简单类型
	if simpleTypes[typeStr] {
		return true
	}

	// 检查是否以[]开头（切片类型）
	if strings.HasPrefix(typeStr, "[]") {
		return true
	}

	// 检查是否以map开头（map类型）
	if strings.HasPrefix(typeStr, "map[") {
		return true
	}

	return false
}

// convertFieldsToJSONFields 将结构体字段转换为JSON字段信息
func (tr *TypeResolver) convertFieldsToJSONFields(fields []FieldInfo) []JSONFieldInfo {
	var jsonFields []JSONFieldInfo
	for _, field := range fields {
		jsonName := tr.getJSONTagName(field)
		if jsonName == "" {
			jsonName = tr.toSnakeCase(field.Name)
		}
		jsonFields = append(jsonFields, JSONFieldInfo{
			Name: jsonName, // JSON字段名使用转换后的名称
			Type: field.Type,
			Tag:  field.GormTag,
		})
	}
	return jsonFields
}

// getJSONTagName 获取字段的JSON标签名
func (tr *TypeResolver) getJSONTagName(field FieldInfo) string {
	// 如果有AST字段信息，尝试解析JSON标签
	if field.ASTField != nil && field.ASTField.Tag != nil {
		tagStr := strings.Trim(field.ASTField.Tag.Value, "`")
		jsonTag := tr.extractJSONTag(tagStr)
		if jsonTag != "" && jsonTag != "-" {
			return jsonTag
		}
	}
	return ""
}

// extractJSONTag 提取JSON标签
func (tr *TypeResolver) extractJSONTag(tagStr string) string {
	if !strings.HasPrefix(tagStr, "json:") {
		return ""
	}
	jsonContent := strings.TrimPrefix(tagStr, "json:")
	jsonContent = strings.Trim(jsonContent, `"`)

	// 处理JSON标签中的选项，如 "name,omitempty"
	if commaIndex := strings.Index(jsonContent, ","); commaIndex != -1 {
		jsonContent = jsonContent[:commaIndex]
	}
	return jsonContent
}

// resolveTypeFromDomain 从domain包解析类型
func (tr *TypeResolver) resolveTypeFromDomain(typeInfo *TypeInfo) error {
	// 尝试常见的domain包路径
	goModPath, err := tr.findGoModFile(".")
	if err != nil {
		return fmt.Errorf("查找go.mod失败: %w", err)
	}

	projectRoot := filepath.Dir(goModPath)
	possiblePaths := []string{
		filepath.Join(projectRoot, "internal", "app", "launchpad", "biz", "listing"),
		filepath.Join(projectRoot, "internal", "domain"),
		filepath.Join(projectRoot, "domain"),
	}

	for _, pkgPath := range possiblePaths {
		if _, err := os.Stat(pkgPath); err == nil {
			typeSpec, filePath, err := tr.findTypeDefinition(pkgPath, typeInfo.Name)
			if err == nil {
				typeInfo.FilePath = filePath
				// 解析结构体字段
				if structType, ok := typeSpec.Type.(*ast.StructType); ok {
					fields, err := tr.parseStructFields(structType)
					if err != nil {
						return err
					}
					typeInfo.Fields = fields
				}
				return nil
			}
		}
	}

	return fmt.Errorf("在domain包中未找到类型: %s", typeInfo.Name)
}

// resolveTypeFromCurrentPackage 从当前包解析类型
func (tr *TypeResolver) resolveTypeFromCurrentPackage(typeInfo *TypeInfo) error {
	// 尝试当前目录
	typeSpec, filePath, err := tr.findTypeDefinition(".", typeInfo.Name)
	if err == nil {
		typeInfo.FilePath = filePath
		// 解析结构体字段
		if structType, ok := typeSpec.Type.(*ast.StructType); ok {
			fields, err := tr.parseStructFields(structType)
			if err != nil {
				return err
			}
			typeInfo.Fields = fields
		}
		return nil
	}

	return fmt.Errorf("在当前包中未找到类型: %s", typeInfo.Name)
}

// toSnakeCase 转换为snake_case
func (tr *TypeResolver) toSnakeCase(s string) string {
	if s == "" {
		return s
	}
	if s == "ID" {
		return "id"
	}
	if strings.HasSuffix(s, "ID") && len(s) > 2 {
		prefix := s[:len(s)-2]
		return tr.toSnakeCase(prefix) + "_id"
	}

	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// parseTypeMethods 解析类型方法
func (tr *TypeResolver) parseTypeMethods(typeName, pkgPath string) ([]MethodInfo, error) {
	var methods []MethodInfo

	// 解析目录中的所有文件
	files, err := tr.parseDirectory(pkgPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok {
				// 检查是否为该类型的方法
				if fn.Recv != nil && len(fn.Recv.List) > 0 {
					recv := fn.Recv.List[0]
					if tr.isReceiverType(recv.Type, typeName) {
						method := MethodInfo{
							Name:       fn.Name.Name,
							IsExported: fn.Name.IsExported(),
						}

						// 解析参数
						if fn.Type.Params != nil {
							for _, param := range fn.Type.Params.List {
								paramType := &TypeInfo{}
								paramType.Name = tr.getFieldType(param.Type)
								method.Params = append(method.Params, *paramType)
							}
						}

						// 解析返回值
						if fn.Type.Results != nil {
							for _, result := range fn.Type.Results.List {
								resultType := &TypeInfo{}
								resultType.Name = tr.getFieldType(result.Type)
								method.Returns = append(method.Returns, *resultType)
							}
						}

						methods = append(methods, method)
					}
				}
			}
		}
	}

	return methods, nil
}

// isReceiverType 检查接收者类型是否匹配
func (tr *TypeResolver) isReceiverType(expr ast.Expr, typeName string) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name == typeName
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name == typeName
		}
	}
	return false
}

// HasExportPatchMethod 检查类型是否有ExportPatch方法
func (tr *TypeResolver) HasExportPatchMethod(typeInfo *TypeInfo) bool {
	for _, method := range typeInfo.Methods {
		if method.Name == "ExportPatch" && method.IsExported {
			// 检查方法签名：ExportPatch() *Patch
			if len(method.Params) == 0 && len(method.Returns) == 1 {
				returnType := method.Returns[0]
				return strings.HasPrefix(returnType.Name, "*") && strings.HasSuffix(returnType.Name, "Patch")
			}
		}
	}
	return false
}
