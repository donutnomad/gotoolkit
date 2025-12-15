package gormparse

import (
	"regexp"
	"strings"
	"unicode"
)

// GormFieldInfo GORM字段信息
type GormFieldInfo struct {
	Name              string // 字段名
	Type              string // 字段类型
	ColumnName        string // 数据库列名
	IsEmbedded        bool   // 是否为嵌入字段
	SourceType        string // 字段来源类型,为空表示来自结构体本身,否则表示来自嵌入的结构体
	Tag               string // 字段标签
	EmbeddedPrefix    string // gorm embedded 字段的 prefix
	EmbeddedFieldName string // 原始 embedded 字段名，如 "Account"
	EmbeddedFieldType string // 原始 embedded 字段类型，如 "AccountIDColumns"
}

// GormModelInfo GORM模型信息
type GormModelInfo struct {
	Name           string          // 结构体名称
	PackageName    string          // 包名
	TableName      string          // 表名
	Prefix         string          // 生成的结构体前缀
	Fields         []GormFieldInfo // 字段列表
	Imports        []string        // 导入的包
	EmbeddedGroups []EmbeddedGroup // embedded 字段分组
}

// MethodInfo 表示方法信息
type MethodInfo struct {
	Name         string // 方法名
	ReceiverName string // 接收器名称
	ReceiverType string // 接收器类型
	ReturnType   string // 返回类型
	FilePath     string // 方法所在文件的绝对路径
}

// FieldInfo 表示结构体字段信息
type FieldInfo struct {
	Name              string // 字段名
	Type              string // 字段类型
	PkgPath           string // 类型所在包路径
	Tag               string // 字段标签
	SourceType        string // 字段来源类型,为空表示来自结构体本身,否则表示来自嵌入的结构体
	EmbeddedPrefix    string // gorm embedded 字段的 prefix
	EmbeddedFieldName string // 原始 embedded 字段名，如 "Account"
	EmbeddedFieldType string // 原始 embedded 字段类型，如 "AccountIDColumns"
}

// EmbeddedGroup 表示一组来自同一个 embedded 字段的字段
type EmbeddedGroup struct {
	FieldName string          // 原始 embedded 字段名，如 "Account"
	FieldType string          // 原始 embedded 字段类型，如 "AccountIDColumns"
	Fields    []GormFieldInfo // 属于这个 embedded 字段的所有展开字段
}

// StructInfo 表示结构体信息
type StructInfo struct {
	Name        string       // 结构体名称
	PackageName string       // 包名
	Fields      []FieldInfo  // 字段列表
	Methods     []MethodInfo // 方法列表
	Imports     []string     // 导入的包
}

// ExtractColumnName 提取列名(从gorm标签或使用默认规则)
func ExtractColumnName(fieldName, fieldTag string) string {
	return ExtractColumnNameWithPrefix(fieldName, fieldTag, "")
}

// ExtractColumnNameWithPrefix 提取列名，支持 embeddedPrefix
func ExtractColumnNameWithPrefix(fieldName, fieldTag, embeddedPrefix string) string {
	var columnName string

	if fieldTag == "" {
		columnName = ToSnakeCase(fieldName)
	} else {
		// 解析GORM标签
		gormTags := parseGormTag(fieldTag)
		if col, exists := gormTags["column"]; exists {
			columnName = col
		} else {
			// 没有找到column标签,使用默认规则
			columnName = ToSnakeCase(fieldName)
		}
	}

	// 应用 embeddedPrefix
	if embeddedPrefix != "" {
		columnName = embeddedPrefix + columnName
	}

	return columnName
}

// ParseGormModel 解析GORM模型
func ParseGormModel(structInfo *StructInfo) *GormModelInfo {
	gormModel := &GormModelInfo{
		Name:        structInfo.Name,
		PackageName: structInfo.PackageName,
		Imports:     structInfo.Imports,
	}

	// 用于构建 EmbeddedGroups 的临时 map
	embeddedGroupsMap := make(map[string]*EmbeddedGroup)

	for _, field := range structInfo.Fields {
		// 跳过特殊字段
		if shouldSkipField(field.Name) {
			continue
		}

		gormField := GormFieldInfo{
			Name:              field.Name,
			Type:              field.Type,
			SourceType:        field.SourceType,        // 复制来源信息
			Tag:               field.Tag,               // 保存标签信息
			EmbeddedPrefix:    field.EmbeddedPrefix,    // 复制 embeddedPrefix
			EmbeddedFieldName: field.EmbeddedFieldName, // 复制 embedded 字段名
			EmbeddedFieldType: field.EmbeddedFieldType, // 复制 embedded 字段类型
		}

		// 解析列名（使用 embeddedPrefix）
		gormField.ColumnName = ExtractColumnNameWithPrefix(field.Name, field.Tag, field.EmbeddedPrefix)

		gormModel.Fields = append(gormModel.Fields, gormField)

		// 构建 EmbeddedGroups
		if field.EmbeddedFieldName != "" {
			if group, exists := embeddedGroupsMap[field.EmbeddedFieldName]; exists {
				group.Fields = append(group.Fields, gormField)
			} else {
				embeddedGroupsMap[field.EmbeddedFieldName] = &EmbeddedGroup{
					FieldName: field.EmbeddedFieldName,
					FieldType: field.EmbeddedFieldType,
					Fields:    []GormFieldInfo{gormField},
				}
			}
		}
	}

	// 将 map 转换为 slice（保持顺序：按首次出现的字段顺序）
	for _, field := range gormModel.Fields {
		if field.EmbeddedFieldName != "" {
			if group, exists := embeddedGroupsMap[field.EmbeddedFieldName]; exists {
				gormModel.EmbeddedGroups = append(gormModel.EmbeddedGroups, *group)
				delete(embeddedGroupsMap, field.EmbeddedFieldName)
			}
		}
	}

	return gormModel
}

// shouldSkipField 判断是否跳过字段
func shouldSkipField(fieldName string) bool {
	skipFields := []string{
		// 移除了Model相关的跳过,因为它们现在会被扁平化
		// "Model", "gorm.Model", "orm.Model" - 这些现在通过扁平化处理

		// 保留通常不需要patch的字段
		// 注意:ID, CreatedAt, UpdatedAt, DeletedAt 现在可能需要patch,根据业务需要决定
		// 如果确实不需要patch这些字段,可以在这里添加回来
	}

	for _, skip := range skipFields {
		if fieldName == skip {
			return true
		}
	}
	return false
}

// ToSnakeCase 将驼峰命名转换为蛇形命名,正确处理连续大写字母(缩写词)
func ToSnakeCase(str string) string {
	var result strings.Builder
	runes := []rune(str)

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// 在大写字母前添加下划线,但需要考虑连续大写字母的情况
		if i > 0 && unicode.IsUpper(r) {
			// 检查是否为连续大写字母的结尾(后面跟小写字母)
			// 例如:HTTPServer 中的 P (后面是S大写,不加_) 和 S (后面是e小写,需要加_)
			if i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
				// 当前大写字母后面是小写字母,需要添加下划线
				// 但要检查这是否是连续大写字母的最后一个
				if i > 1 && unicode.IsUpper(runes[i-1]) {
					// 前面也是大写字母,说明这是连续大写字母的最后一个
					// 例如:HTTPServer 中的S,前面是P(大写),后面是e(小写)
					result.WriteRune('_')
				} else {
					// 前面不是大写字母,这是一个单独的大写字母
					// 例如:DefaultID 中的I,前面是t(小写),后面是D(大写)
					result.WriteRune('_')
				}
			} else {
				// 当前大写字母后面还是大写字母,或者是最后一个字符
				// 检查前一个字符是否为小写字母
				if i > 0 && unicode.IsLower(runes[i-1]) {
					// 前面是小写字母,这是新的大写字母序列的开始
					// 例如:DefaultID 中的I,前面是t(小写)
					result.WriteRune('_')
				}
				// 如果前面也是大写字母,则不添加下划线(连续大写字母)
			}
		}

		result.WriteRune(unicode.ToLower(r))
	}

	return result.String()
}

// parseGormTag 解析GORM标签
func parseGormTag(tag string) map[string]string {
	result := make(map[string]string)

	// 提取gorm标签内容
	re := regexp.MustCompile(`gorm:"([^"]*)"`)
	matches := re.FindStringSubmatch(tag)
	if len(matches) < 2 {
		return result
	}

	gormTag := matches[1]

	// 解析标签内的各个部分
	parts := strings.Split(gormTag, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			if len(kv) == 2 {
				result[kv[0]] = kv[1]
			}
		} else {
			result[part] = ""
		}
	}

	return result
}
