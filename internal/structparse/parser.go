package structparse

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/donutnomad/gotoolkit/internal/xast"
)

// FieldInfo 表示结构体字段信息
type FieldInfo struct {
	Name       string // 字段名
	Type       string // 字段类型
	PkgPath    string // 类型所在包路径
	Tag        string // 字段标签
	SourceType string // 字段来源类型，为空表示来自结构体本身，否则表示来自嵌入的结构体
}

// StructInfo 表示结构体信息
type StructInfo struct {
	Name        string      // 结构体名称
	PackageName string      // 包名
	Fields      []FieldInfo // 字段列表
	Imports     []string    // 导入的包
}

// ParseStruct 解析指定文件中的结构体
func ParseStruct(filename, structName string) (*StructInfo, error) {
	// 解析当前文件的导入信息
	imports, err := extractImports(filename)
	if err != nil {
		return nil, err
	}
	return parseStructWithStackAndImports(filename, structName, make(map[string]bool), imports)
}

// extractImports 提取文件中的导入信息
func extractImports(filename string) (map[string]string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	imports := make(map[string]string)
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")

		var packageAlias string
		if imp.Name != nil {
			// 有显式别名
			packageAlias = imp.Name.Name
		} else {
			// 使用路径最后一部分作为包名
			parts := strings.Split(importPath, "/")
			packageAlias = parts[len(parts)-1]
		}

		imports[packageAlias] = importPath
	}

	return imports, nil
}

// shouldExpandEmbeddedField 判断是否应该展开嵌入字段
func shouldExpandEmbeddedField(fieldType string) bool {
	// 内置类型不展开
	builtinTypes := []string{
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"string", "bool",
		"byte", "rune",
		"error",
		"time.Time", "time.Duration",
	}

	for _, builtin := range builtinTypes {
		if fieldType == builtin {
			return false
		}
	}

	// 跳过指针、切片、映射等复合类型
	if strings.HasPrefix(fieldType, "*") ||
		strings.HasPrefix(fieldType, "[]") ||
		strings.HasPrefix(fieldType, "map[") ||
		strings.HasPrefix(fieldType, "chan ") ||
		strings.HasPrefix(fieldType, "func(") {
		return false
	}

	// 其他所有结构体类型都尝试展开
	return true
}

// parseEmbeddedStructWithStack 带栈的递归解析，避免循环引用
func parseEmbeddedStructWithStack(structType string, stack map[string]bool, imports map[string]string) ([]FieldInfo, error) {
	// 检查是否已经在解析栈中（避免循环引用）
	if stack[structType] {
		return nil, nil
	}

	// 将当前类型加入解析栈
	stack[structType] = true
	defer delete(stack, structType) // 解析完成后从栈中移除

	// 解析包名和结构体名
	packageName, structName := parseTypePackageAndName(structType)

	var targetFile string
	var err error

	if packageName == "" {
		// 同包内的结构体，在当前目录查找
		files, err := findGoFiles(".")
		if err != nil {
			return nil, fmt.Errorf("查找当前目录Go文件失败: %v", err)
		}

		for _, file := range files {
			if containsStruct(file, structName) {
				targetFile = file
				break
			}
		}

		if targetFile == "" {
			return nil, fmt.Errorf("未在当前包中找到结构体 %s", structName)
		}
	} else {
		// 跨包结构体，需要根据import路径查找
		targetFile, err = findStructInPackageWithImports(packageName, structName, imports)
		if err != nil {
			// 明确报告找不到第三方包的错误
			return nil, fmt.Errorf("无法解析嵌入的结构体 %s.%s: %v", packageName, structName, err)
		}

		if targetFile == "" {
			return nil, fmt.Errorf("未在包 %s 中找到结构体 %s", packageName, structName)
		}
	}

	// 递归解析该结构体
	structInfo, err := parseStructWithStackAndImports(targetFile, structName, stack, imports)
	if err != nil {
		return nil, fmt.Errorf("解析嵌入结构体 %s 失败: %v", structType, err)
	}

	// 为从嵌入结构体来的字段标记来源
	fields := make([]FieldInfo, len(structInfo.Fields))
	for i, field := range structInfo.Fields {
		fields[i] = field
		// 如果字段已经有来源标记，保持原来的来源；否则标记为当前嵌入类型
		if field.SourceType == "" {
			fields[i].SourceType = structType
		}
	}

	return fields, nil
}

// parseStructWithStackAndImports 带栈和导入信息的结构体解析
func parseStructWithStackAndImports(filename, structName string, stack map[string]bool, imports map[string]string) (*StructInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("解析文件失败: %w", err)
	}

	structInfo := &StructInfo{
		Name:        structName,
		PackageName: node.Name.Name,
	}

	// 收集导入信息
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")
		structInfo.Imports = append(structInfo.Imports, importPath)
	}

	// 查找目标结构体
	var targetStruct *ast.StructType
	ast.Inspect(node, func(n ast.Node) bool {
		if genDecl, ok := n.(*ast.GenDecl); ok {
			if genDecl.Tok == token.TYPE {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						if typeSpec.Name.Name == structName {
							if structType, ok := typeSpec.Type.(*ast.StructType); ok {
								targetStruct = structType
								return false
							}
						}
					}
				}
			}
		}
		return true
	})

	if targetStruct == nil {
		return nil, fmt.Errorf("未找到结构体 %s", structName)
	}

	// 解析字段（传入栈信息和导入信息）
	fields, err := parseStructFieldsWithStackAndImports(targetStruct.Fields.List, stack, imports)
	if err != nil {
		return nil, err
	}
	structInfo.Fields = fields

	return structInfo, nil
}

// parseStructWithStack 带栈的结构体解析（保留向后兼容）
func parseStructWithStack(filename, structName string, stack map[string]bool) (*StructInfo, error) {
	// 提取当前文件的导入信息
	imports, err := extractImports(filename)
	if err != nil {
		return nil, err
	}
	return parseStructWithStackAndImports(filename, structName, stack, imports)
}

// parseStructFieldsWithStackAndImports 带栈和导入信息的字段解析
func parseStructFieldsWithStackAndImports(fieldList []*ast.Field, stack map[string]bool, imports map[string]string) ([]FieldInfo, error) {
	var fields []FieldInfo

	for _, field := range fieldList {
		fieldType := xast.GetFieldType(field.Type, nil)

		// 获取字段标签
		var fieldTag string
		if field.Tag != nil {
			fieldTag = field.Tag.Value
		}

		if len(field.Names) == 0 {
			// 匿名字段 (嵌入字段)
			if shouldExpandEmbeddedField(fieldType) {
				// 需要扩展的嵌入字段，尝试递归解析
				embeddedFields, err := parseEmbeddedStructWithStack(fieldType, stack, imports)
				if err != nil {
					return nil, err // 传递错误给上层
				}
				fields = append(fields, embeddedFields...)
			} else {
				// 不需要扩展的嵌入字段，保持原样
				fields = append(fields, FieldInfo{
					Name: fieldType,
					Type: fieldType,
					Tag:  fieldTag,
				})
			}
		} else {
			// 有名字段
			for _, name := range field.Names {
				fields = append(fields, FieldInfo{
					Name: name.Name,
					Type: fieldType,
					Tag:  fieldTag,
				})
			}
		}
	}

	for i, field := range fields {
		if field.SourceType != "" {
			if idx := strings.Index(field.SourceType, "."); idx >= 0 {
				if !strings.Contains(field.Type, ".") && (field.Type[0] >= 'A' && field.Type[0] <= 'Z') {
					fields[i].Type = field.SourceType[:idx] + "." + field.Type
				}
			}
		}
	}

	return fields, nil
}

// parseStructFieldsWithStack 带栈的字段解析（保留向后兼容）
func parseStructFieldsWithStack(fieldList []*ast.Field, stack map[string]bool) []FieldInfo {
	// 使用空的imports映射来保持向后兼容
	fields, err := parseStructFieldsWithStackAndImports(fieldList, stack, make(map[string]string))
	if err != nil {
		// 向后兼容，忽略错误
		return nil
	}
	return fields
}

// parseTypePackageAndName 解析类型的包名和结构体名
// 输入: "orm.Model" 返回: "orm", "Model"
// 输入: "User" 返回: "", "User"
func parseTypePackageAndName(typeName string) (packageName, structName string) {
	parts := strings.Split(typeName, ".")
	if len(parts) == 1 {
		return "", parts[0]
	}
	return parts[0], parts[1]
}

// findStructInPackageWithImports 在指定包中查找结构体定义，使用导入信息
func findStructInPackageWithImports(packageName, structName string, imports map[string]string) (string, error) {
	// 从imports中获取完整的导入路径
	fullImportPath, exists := imports[packageName]
	if !exists {
		return "", fmt.Errorf("未找到包 %s 的导入信息", packageName)
	}

	// 首先尝试从当前项目的根目录开始查找
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}

	// 根据完整导入路径查找包路径
	packagePath, err := findPackagePathByImport(projectRoot, fullImportPath)
	if err != nil {
		return "", err
	}

	// 在包路径中查找包含结构体的文件
	files, err := findGoFiles(packagePath)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if containsStruct(file, structName) {
			return file, nil
		}
	}

	return "", fmt.Errorf("未在包 %s 中找到结构体 %s", packageName, structName)
}

// findStructInPackage 在指定包中查找结构体定义
func findStructInPackage(packageName, structName string) (string, error) {
	// 首先尝试从当前项目的根目录开始查找
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}

	// 在项目根目录中查找对应的包路径
	packagePath, err := findPackagePath(projectRoot, packageName)
	if err != nil {
		return "", err
	}

	// 在包路径中查找包含结构体的文件
	files, err := findGoFiles(packagePath)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if containsStruct(file, structName) {
			return file, nil
		}
	}

	return "", fmt.Errorf("未在包 %s 中找到结构体 %s", packageName, structName)
}

// findPackagePathByImport 根据完整导入路径查找包路径
func findPackagePathByImport(projectRoot, importPath string) (string, error) {
	// 读取go.mod获取module名称
	moduleName, err := getModuleName(projectRoot)
	if err != nil {
		return "", err
	}

	// 如果导入路径以当前模块名开头，则是项目内部包
	if strings.HasPrefix(importPath, moduleName) {
		relativePath := strings.TrimPrefix(importPath, moduleName)
		relativePath = strings.TrimPrefix(relativePath, "/")
		packagePath := filepath.Join(projectRoot, relativePath)

		if _, err := os.Stat(packagePath); err == nil {
			return packagePath, nil
		}
	}

	// 处理标准库导入：标准库包不包含域名（不含"."）
	// 例如：fmt, os, net/http, encoding/json, crypto/sha256 等
	if !strings.Contains(importPath, ".") {
		return "", fmt.Errorf("标准库包 %s 不支持结构体解析", importPath)
	}

	// 处理第三方包：尝试从Go模块缓存中查找
	return findThirdPartyPackage(importPath)
}

// findThirdPartyPackage 查找第三方包的路径
func findThirdPartyPackage(importPath string) (string, error) {
	// 获取GOPATH和GOMODCACHE
	goPath := os.Getenv("GOPATH")
	goModCache := os.Getenv("GOMODCACHE")

	// 如果GOMODCACHE为空，使用默认路径
	if goModCache == "" {
		if goPath == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("无法获取用户主目录: %v", err)
			}
			goPath = filepath.Join(homeDir, "go")
		}
		goModCache = filepath.Join(goPath, "pkg", "mod")
	}

	// 尝试在GOMODCACHE中查找包
	// Go模块缓存中的路径格式通常是: github.com/user/repo@version
	// 我们需要遍历可能的版本
	packageCachePattern := filepath.Join(goModCache, importPath+"@*")
	matches, err := filepath.Glob(packageCachePattern)
	if err != nil {
		return "", fmt.Errorf("搜索模块缓存失败: %v", err)
	}

	// 如果找到多个版本，选择最新的一个（按字典序排序）
	if len(matches) > 0 {
		// 简单地选择最后一个（通常版本号较高）
		latestMatch := matches[len(matches)-1]
		if _, err := os.Stat(latestMatch); err == nil {
			return latestMatch, nil
		}
	}

	// 如果在模块缓存中没找到，尝试在GOPATH/src中查找（旧版本Go的方式）
	if goPath != "" {
		goPathSrc := filepath.Join(goPath, "src", importPath)
		if _, err := os.Stat(goPathSrc); err == nil {
			return goPathSrc, nil
		}
	}

	return "", fmt.Errorf("未找到第三方包 %s，请确保该包已正确安装", importPath)
}

// getModuleName 从go.mod文件获取模块名称
func getModuleName(projectRoot string) (string, error) {
	goModPath := filepath.Join(projectRoot, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}

	return "", fmt.Errorf("未在 go.mod 中找到模块名称")
}

// findProjectRoot 查找项目根目录（包含go.mod的目录）
func findProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := currentDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// 已经到了根目录
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("未找到项目根目录（go.mod文件）")
}

// findPackagePath 根据包名查找包路径
func findPackagePath(projectRoot, packageName string) (string, error) {
	// 如果是相对导入（如 "taas-backend/pkg/orm"）
	// 则在项目根目录下查找对应路径

	// 读取当前文件的import信息来解析完整的import路径
	currentFile := ""
	files, err := findGoFiles(".")
	if err == nil && len(files) > 0 {
		currentFile = files[0]
	}

	if currentFile != "" {
		fullImportPath, err := findImportPath(currentFile, packageName)
		if err == nil {
			// 解析相对于项目根目录的路径
			if strings.Contains(fullImportPath, "/") {
				parts := strings.Split(fullImportPath, "/")
				if len(parts) > 1 {
					// 从项目根目录构建路径
					relativePath := strings.Join(parts[1:], "/")
					packagePath := filepath.Join(projectRoot, relativePath)
					if _, err := os.Stat(packagePath); err == nil {
						return packagePath, nil
					}
				}
			}
		}
	}

	// 如果无法从import解析，尝试常见的包结构
	commonPaths := []string{
		filepath.Join(projectRoot, "pkg", packageName),
		filepath.Join(projectRoot, "internal", packageName),
		filepath.Join(projectRoot, packageName),
		filepath.Join(projectRoot, "src", packageName),
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("未找到包 %s 的路径", packageName)
}

// findImportPath 从文件中查找指定包的完整import路径
func findImportPath(filename, packageAlias string) (string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return "", err
	}

	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, "\"")

		// 检查是否有别名
		if imp.Name != nil {
			if imp.Name.Name == packageAlias {
				return importPath, nil
			}
		} else {
			// 没有别名，使用路径最后一部分作为包名
			parts := strings.Split(importPath, "/")
			if len(parts) > 0 && parts[len(parts)-1] == packageAlias {
				return importPath, nil
			}
		}
	}

	return "", fmt.Errorf("未找到包 %s 的导入路径", packageAlias)
}

// findGoFiles 查找目录中的所有Go文件
func findGoFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// containsStruct 检查文件是否包含指定的结构体
func containsStruct(filename, structName string) bool {
	content, err := os.ReadFile(filename)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), fmt.Sprintf("type %s struct", structName))
}
