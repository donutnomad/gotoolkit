package automap

import (
	"go/ast"
	"reflect"
	"strings"
)

// analyzeJSONCompositeLit 分析 JSON 结构体字面量
func (m *Mapper) analyzeJSONCompositeLit(fieldName string, compLit *ast.CompositeLit, fieldInfo *FieldAnalysisInfo) error {
	group := MappingGroup{
		Type:        ManyToOne,
		TargetField: fieldName,
	}

	m.extractJSONMappings(&group, compLit, "", "", fieldInfo.ColumnName)

	if len(group.Mappings) > 0 {
		m.result.Groups = append(m.result.Groups, group)
	}
	return nil
}

// extractJSONMappings 递归提取 JSON 映射
// goFieldPrefix: Go 字段路径前缀（真实的 Go 字段名）
func (m *Mapper) extractJSONMappings(group *MappingGroup, compLit *ast.CompositeLit, jsonPrefix string, goFieldPrefix string, columnName string) {
	// 获取 JSON 类型的字段信息（用于获取 json tag）
	jsonTypeName := extractTypeName(compLit.Type)
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

		// 构建 Go 字段路径（真实的 Go 字段名）
		goFieldPath := subFieldName
		if goFieldPrefix != "" {
			goFieldPath = goFieldPrefix + "." + subFieldName
		}

		// 检查是否是嵌套结构体
		if nestedCompLit, ok := kv.Value.(*ast.CompositeLit); ok {
			m.extractJSONMappings(group, nestedCompLit, jsonPath, goFieldPath, columnName)
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
			GoFieldPath: goFieldPath,
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
		return fun.Sel.Name == "NewJSONType"
	case *ast.Ident:
		// NewJSONType
		return fun.Name == "NewJSONType"
	}
	return false
}

// isJSONSliceConstructor 检查是否是 JSONSlice 构造函数
func (m *Mapper) isJSONSliceConstructor(callExpr *ast.CallExpr) bool {
	switch fun := callExpr.Fun.(type) {
	case *ast.SelectorExpr:
		// datatypes.NewJSONSlice
		return fun.Sel.Name == "NewJSONSlice"
	case *ast.Ident:
		// NewJSONSlice
		return fun.Name == "NewJSONSlice"
	}
	return false
}

// extractLoMapSource 从 lo.Map 调用中提取源路径
// lo.Map(entity.Field, func...) -> entity.Field
// 注意：此函数只处理直接字段访问，方法调用由 extractLoMapMethodCall 处理
func (m *Mapper) extractLoMapSource(callExpr *ast.CallExpr) (string, bool) {
	// 检查是否是 lo.Map(source, func...)
	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	// 检查是否是 lo.Map
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok || pkgIdent.Name != "lo" {
		return "", false
	}
	if sel.Sel.Name != "Map" {
		return "", false
	}

	// 需要至少一个参数
	if len(callExpr.Args) < 1 {
		return "", false
	}

	// 提取第一个参数作为源
	if argSelector, ok := callExpr.Args[0].(*ast.SelectorExpr); ok {
		path := m.buildSelectorPath(argSelector)
		if path != "" {
			return path, true
		}
	}

	return "", false
}

// extractLoMapMethodCall 从 lo.Map 调用中提取方法调用信息
// lo.Map(entity.GetMethod(), func...) -> methodCallInfo
func (m *Mapper) extractLoMapMethodCall(callExpr *ast.CallExpr) *methodCallInfo {
	// 检查是否是 lo.Map(source, func...)
	sel, ok := callExpr.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	// 检查是否是 lo.Map
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok || pkgIdent.Name != "lo" {
		return nil
	}
	if sel.Sel.Name != "Map" {
		return nil
	}

	// 需要至少一个参数
	if len(callExpr.Args) < 1 {
		return nil
	}

	// 检查第一个参数是否是方法调用
	return m.extractMethodCallInfo(callExpr.Args[0])
}
