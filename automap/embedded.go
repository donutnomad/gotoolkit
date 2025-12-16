package automap

import (
	"go/ast"
	"reflect"
	"strings"

	"github.com/donutnomad/gotoolkit/internal/gormparse"
	"github.com/donutnomad/gotoolkit/internal/structparse"
)

// analyzeEmbeddedCompositeLit 分析嵌入字段的结构体字面量
func (m *Mapper) analyzeEmbeddedCompositeLit(fieldName string, compLit *ast.CompositeLit, fieldInfo *FieldAnalysisInfo) error {
	// 获取嵌入类型的字段信息
	embeddedTypeName := extractTypeName(compLit.Type)
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

// analyzeEmbeddedOneToManyMapping 分析嵌入字段的一对多映射
// 处理场景：Account: d.Account 或 Account: d.Account.ToColumns()
// 一个输入字段映射到多个输出列（嵌入结构体的所有字段）
func (m *Mapper) analyzeEmbeddedOneToManyMapping(fieldName string, value ast.Expr, fieldInfo *FieldAnalysisInfo) error {
	// 提取源路径（如 d.Account 或方法调用）
	sourcePath, _ := m.extractSourcePath(value)
	if sourcePath == "" {
		// 尝试从方法调用中提取
		if methodInfo := m.extractMethodCallInfo(value); methodInfo != nil {
			// 从方法名推断字段名
			sourcePath = inferFieldNameFromMethod(methodInfo.methodName)
		}
	}
	if sourcePath == "" {
		return nil
	}

	// 获取嵌入类型的结构体定义
	embeddedTypeName := fieldInfo.EmbeddedTypeName
	if embeddedTypeName == "" {
		return nil
	}

	// 创建 EmbeddedOneToMany 映射组
	group := MappingGroup{
		Type:        EmbeddedOneToMany,
		TargetField: fieldName,
		SourceField: sourcePath,
	}

	// 首先尝试从当前包的 typeSpecs 中获取
	embeddedTypeSpec := m.typeSpecs[embeddedTypeName]
	embeddedStructType, ok := m.getStructType(embeddedTypeSpec)
	if ok {
		// 从 AST 解析字段
		m.extractEmbeddedFieldsFromAST(&group, embeddedStructType, fieldName, fieldInfo.EmbeddedPrefix, sourcePath)
	} else {
		// 尝试使用 structparse 解析
		// 先在同目录下查找类型定义
		if !m.extractEmbeddedFieldsFromStructparse(&group, embeddedTypeName, fieldName, fieldInfo.EmbeddedPrefix, sourcePath) {
			// 如果没找到，尝试从 PO 结构体本身解析外部包嵌入字段
			m.extractEmbeddedFieldsFromPOStruct(&group, embeddedTypeName, fieldName, fieldInfo.EmbeddedPrefix, sourcePath)
		}
	}

	if len(group.Mappings) > 0 {
		m.result.Groups = append(m.result.Groups, group)
	}
	return nil
}

// extractEmbeddedFieldsFromAST 从 AST 提取嵌入结构体的字段
func (m *Mapper) extractEmbeddedFieldsFromAST(group *MappingGroup, structType *ast.StructType, fieldName, prefix, sourcePath string) {
	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			subFieldName := name.Name
			// 跳过非导出字段
			if !isExported(subFieldName) {
				continue
			}

			// 获取列名
			columnName := toSnakeCase(subFieldName)
			if field.Tag != nil {
				tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
				gormTag := tag.Get("gorm")
				for _, part := range strings.Split(gormTag, ";") {
					if strings.HasPrefix(part, "column:") {
						columnName = strings.TrimPrefix(part, "column:")
					}
				}
			}

			// 应用嵌入前缀
			if prefix != "" {
				columnName = prefix + columnName
			}

			mapping := FieldMapping2{
				SourcePath: sourcePath,
				TargetPath: fieldName + "." + subFieldName,
				ColumnName: columnName,
			}
			group.Mappings = append(group.Mappings, mapping)
		}
	}
}

// extractEmbeddedFieldsFromStructparse 使用 structparse 解析嵌入结构体的字段（支持外部包）
// 返回 true 表示找到并解析了类型
func (m *Mapper) extractEmbeddedFieldsFromStructparse(group *MappingGroup, embeddedTypeName, fieldName, prefix, sourcePath string) bool {
	iterator := NewGoFileIterator(m.filePath)
	found := false

	_ = iterator.IterateIncludeCurrent(func(filePath string) bool {
		structInfo, err := structparse.ParseStruct(filePath, embeddedTypeName)
		if err != nil {
			return true // 继续遍历
		}

		// 找到了，提取字段
		for _, field := range structInfo.Fields {
			// 跳过非导出字段
			if !isExported(field.Name) {
				continue
			}

			// 获取列名
			columnName := gormparse.ExtractColumnNameWithPrefix(field.Name, field.Tag, "")

			// 应用嵌入前缀
			if prefix != "" {
				columnName = prefix + columnName
			}

			mapping := FieldMapping2{
				SourcePath: sourcePath,
				TargetPath: fieldName + "." + field.Name,
				ColumnName: columnName,
			}
			group.Mappings = append(group.Mappings, mapping)
		}
		found = true
		return false // 找到后停止遍历
	})

	return found
}

// extractEmbeddedFieldsFromPOStruct 从 PO 结构体解析外部包嵌入字段
// 当嵌入类型来自外部包时，通过解析 PO 结构体本身来获取嵌入字段信息
// embeddedTypeName: 嵌入类型名（不含包前缀），如 "AccountIDColumnsCompact"
func (m *Mapper) extractEmbeddedFieldsFromPOStruct(group *MappingGroup, embeddedTypeName, fieldName, prefix, sourcePath string) {
	iterator := NewGoFileIterator(m.filePath)

	_ = iterator.IterateIncludeCurrent(func(filePath string) bool {
		structInfo, err := structparse.ParseStruct(filePath, m.receiverType)
		if err != nil {
			return true // 继续遍历
		}

		// 找到 PO 结构体了，查找来自嵌入字段的字段
		// 这些字段会有非空的 SourceType 和 EmbeddedPrefix
		for _, field := range structInfo.Fields {
			// 跳过非导出字段
			if !isExported(field.Name) {
				continue
			}

			// 检查这个字段是否来自我们关注的嵌入字段
			// SourceType 包含包限定的类型名，如 "caip10.AccountIDColumnsCompact"
			// 需要同时满足：
			// 1. SourceType 以 embeddedTypeName 结尾（匹配类型名）
			// 2. EmbeddedPrefix 与期望的前缀匹配
			if field.SourceType != "" &&
				field.EmbeddedPrefix == prefix &&
				(field.SourceType == embeddedTypeName || strings.HasSuffix(field.SourceType, "."+embeddedTypeName)) {
				// 获取列名（已经包含前缀）
				columnName := gormparse.ExtractColumnNameWithPrefix(field.Name, field.Tag, field.EmbeddedPrefix)

				mapping := FieldMapping2{
					SourcePath: sourcePath,
					TargetPath: fieldName + "." + field.Name,
					ColumnName: columnName,
				}
				group.Mappings = append(group.Mappings, mapping)
			}
		}
		return false // 找到 PO 结构体后停止遍历
	})
}
