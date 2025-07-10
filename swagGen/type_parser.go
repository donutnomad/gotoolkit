package main

import (
	"fmt"
	"github.com/donutnomad/gotoolkit/internal/xast"
	"go/ast"
	"strings"
)

// NewReturnTypeParser 创建返回类型解析器
func NewReturnTypeParser(importMgr *EnhancedImportManager, imports xast.ImportInfoSlice) *ReturnTypeParser {
	return &ReturnTypeParser{
		importMgr: importMgr,
		imports:   imports,
	}
}

// ParseReturnType 解析返回类型
func (p *ReturnTypeParser) ParseReturnType(expr ast.Expr) TypeInfo {
	return p.parseType(expr)
}

// ParseParameterType 解析参数类型
func (p *ReturnTypeParser) ParseParameterType(expr ast.Expr) TypeInfo {
	return p.parseType(expr)
}

// parseType 解析类型表达式
func (p *ReturnTypeParser) parseType(expr ast.Expr) TypeInfo {
	switch t := expr.(type) {
	case *ast.Ident:
		// 基本类型或本地定义的类型
		return TypeInfo{
			FullName:  t.Name,
			TypeName:  t.Name,
			Package:   "",
			Alias:     "",
			IsGeneric: false,
		}

	case *ast.SelectorExpr:
		// 包.类型格式
		return p.parseSelectorType(t)

	case *ast.StarExpr:
		// 指针类型
		baseType := p.parseType(t.X)
		baseType.IsPointer = true
		if baseType.FullName != "" {
			baseType.FullName = "*" + baseType.FullName
		}
		return baseType

	case *ast.ArrayType:
		// 数组/切片类型
		return p.parseArrayType(t)

	case *ast.IndexExpr:
		// 泛型类型 T[U]
		return p.parseGenericType(t)

	case *ast.MapType:
		// map 类型
		return p.parseMapType(t)

	case *ast.InterfaceType:
		// interface{} 类型
		return TypeInfo{
			FullName:  "interface{}",
			TypeName:  "interface{}",
			Package:   "",
			Alias:     "",
			IsGeneric: false,
		}

	case *ast.StructType:
		// 匿名结构体
		return TypeInfo{
			FullName:  "struct{}",
			TypeName:  "struct{}",
			Package:   "",
			Alias:     "",
			IsGeneric: false,
		}

	default:
		// 未知类型，返回空类型信息
		return TypeInfo{
			FullName:  "interface{}",
			TypeName:  "interface{}",
			Package:   "",
			Alias:     "",
			IsGeneric: false,
		}
	}
}

// parseSelectorType 解析选择器类型 (包.类型)
func (p *ReturnTypeParser) parseSelectorType(expr *ast.SelectorExpr) TypeInfo {
	// 获取包名
	ident, ok := expr.X.(*ast.Ident)
	if !ok {
		return TypeInfo{
			FullName:  "interface{}",
			TypeName:  "interface{}",
			Package:   "",
			Alias:     "",
			IsGeneric: false,
		}
	}

	pkgName := ident.Name
	typeName := expr.Sel.Name

	// 查找包路径
	pkgPath := p.resolvePackagePath(pkgName)
	if pkgPath == "" {
		// 如果找不到包路径，可能是内置类型
		return TypeInfo{
			FullName:  fmt.Sprintf("%s.%s", pkgName, typeName),
			TypeName:  typeName,
			Package:   pkgName,
			Alias:     pkgName,
			IsGeneric: false,
		}
	}

	// 添加类型引用并获取别名
	alias := p.importMgr.AddTypeReference(pkgPath, typeName)

	fullName := typeName
	if alias != "" {
		fullName = fmt.Sprintf("%s.%s", alias, typeName)
	}

	return TypeInfo{
		FullName:  fullName,
		TypeName:  typeName,
		Package:   pkgPath,
		Alias:     alias,
		IsGeneric: false,
	}
}

// parseArrayType 解析数组类型
func (p *ReturnTypeParser) parseArrayType(expr *ast.ArrayType) TypeInfo {
	// 解析元素类型
	elemType := p.parseType(expr.Elt)

	// 构建数组类型信息
	fullName := "[]" + elemType.FullName
	typeName := "[]" + elemType.TypeName

	return TypeInfo{
		FullName:  fullName,
		TypeName:  typeName,
		Package:   elemType.Package,
		Alias:     elemType.Alias,
		IsGeneric: elemType.IsGeneric,
		IsSlice:   true,
	}
}

// parseGenericType 解析泛型类型
func (p *ReturnTypeParser) parseGenericType(expr *ast.IndexExpr) TypeInfo {
	// 解析基础类型
	baseType := p.parseType(expr.X)

	// 解析泛型参数
	var genericArgs []TypeInfo
	switch argExpr := expr.Index.(type) {
	case *ast.Ident:
		// 单个泛型参数
		genericArgs = append(genericArgs, p.parseType(argExpr))
	default:
		// 其他情况，尝试解析
		genericArgs = append(genericArgs, p.parseType(argExpr))
	}

	// 构建泛型类型名称
	var argNames []string
	for _, arg := range genericArgs {
		argNames = append(argNames, arg.FullName)
	}

	fullName := fmt.Sprintf("%s[%s]", baseType.FullName, strings.Join(argNames, ", "))
	typeName := fmt.Sprintf("%s[%s]", baseType.TypeName, strings.Join(argNames, ", "))

	return TypeInfo{
		FullName:    fullName,
		TypeName:    typeName,
		Package:     baseType.Package,
		Alias:       baseType.Alias,
		IsGeneric:   true,
		GenericArgs: genericArgs,
	}
}

// parseMapType 解析 map 类型
func (p *ReturnTypeParser) parseMapType(expr *ast.MapType) TypeInfo {
	keyType := p.parseType(expr.Key)
	valueType := p.parseType(expr.Value)

	fullName := fmt.Sprintf("map[%s]%s", keyType.FullName, valueType.FullName)
	typeName := fmt.Sprintf("map[%s]%s", keyType.TypeName, valueType.TypeName)

	return TypeInfo{
		FullName:  fullName,
		TypeName:  typeName,
		Package:   "", // map 是内置类型
		Alias:     "",
		IsGeneric: false,
	}
}

// resolvePackagePath 解析包路径
func (p *ReturnTypeParser) resolvePackagePath(pkgName string) string {
	// 从 imports 中查找包路径
	for _, imp := range p.imports {
		importPath := imp.GetPath()
		alias := imp.GetAlias()

		// 如果有别名，使用别名匹配
		if alias != "" && alias == pkgName {
			return importPath
		}

		// 如果没有别名，使用包名匹配
		if alias == "" {
			// 从路径中提取包名
			parts := strings.Split(importPath, "/")
			if len(parts) > 0 {
				lastPart := parts[len(parts)-1]
				if lastPart == pkgName {
					return importPath
				}
			}
		}
	}

	return ""
}

// GetSwaggerType 获取 Swagger 类型字符串
func (info *TypeInfo) GetSwaggerType() string {
	if info.IsSlice {
		return "array"
	}

	// 移除指针和别名前缀
	typeName := info.TypeName
	if info.IsPointer {
		typeName = strings.TrimPrefix(typeName, "*")
	}

	// 基本类型映射
	switch typeName {
	case "string":
		return "string"
	case "int", "int8", "int16", "int32", "int64":
		return "integer"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "object"
	}
}

// GetSwaggerFormat 获取 Swagger 格式字符串
func (info *TypeInfo) GetSwaggerFormat() string {
	typeName := info.TypeName
	if info.IsPointer {
		typeName = strings.TrimPrefix(typeName, "*")
	}

	switch typeName {
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	case "float32":
		return "float"
	case "float64":
		return "double"
	default:
		return ""
	}
}
