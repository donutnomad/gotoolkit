package automap

import (
	"fmt"
	"sort"
	"strings"
)

// Generator2 基于新映射分析方案的代码生成器
type Generator2 struct {
	result      *ParseResult2
	genFuncName string
	imports     map[string]bool

	// 源类型信息（从解析获得）
	sourcePackage string // 源类型的包名（如果是外部包）
	sourceType    string // 源类型名

	// 接收者信息
	receiverType string
	receiverVar  string
}

// NewGenerator2 创建新的代码生成器
func NewGenerator2(result *ParseResult2, genFuncName string) *Generator2 {
	receiverVar := "p"
	if len(result.ReceiverType) > 0 {
		receiverVar = strings.ToLower(result.ReceiverType[:1])
	}

	g := &Generator2{
		result:        result,
		genFuncName:   genFuncName,
		imports:       make(map[string]bool),
		receiverType:  result.ReceiverType,
		receiverVar:   receiverVar,
		sourceType:    result.SourceType,
		sourcePackage: result.SourceTypePackage,
	}

	// 如果源类型是外部包，添加导入
	if result.SourceTypeImportPath != "" {
		g.imports[result.SourceTypeImportPath] = true
	}

	return g
}

// Generate 生成代码
// 返回: (带imports的完整代码, 纯函数代码)
func (g *Generator2) Generate() (string, string) {
	var funcBuilder strings.Builder

	// 生成函数签名
	g.generateFunctionSignature(&funcBuilder)

	// 生成函数体
	g.generateFunctionBody(&funcBuilder)

	// 生成返回语句
	funcBuilder.WriteString("\t_ = b\n")
	funcBuilder.WriteString("\t_ = fields\n")
	funcBuilder.WriteString("\treturn values\n")
	funcBuilder.WriteString("}\n")

	funcCode := funcBuilder.String()

	// 验证字段覆盖情况，添加 Missing fields 注释
	missingComment := g.validateFieldCoverage(funcCode)
	if missingComment != "" {
		funcCode = missingComment + funcCode
	}

	// 生成带 imports 的完整代码
	var fullBuilder strings.Builder
	if len(g.imports) > 0 {
		fullBuilder.WriteString("import (\n")
		importList := make([]string, 0, len(g.imports))
		for imp := range g.imports {
			importList = append(importList, imp)
		}
		sort.Strings(importList)
		for _, imp := range importList {
			fullBuilder.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
		}
		fullBuilder.WriteString(")\n\n")
	}
	fullBuilder.WriteString(funcCode)

	return fullBuilder.String(), funcCode
}

// generateFunctionSignature 生成函数签名
func (g *Generator2) generateFunctionSignature(builder *strings.Builder) {
	sourceTypeName := g.sourceType
	if g.sourcePackage != "" {
		sourceTypeName = g.sourcePackage + "." + g.sourceType
	}

	builder.WriteString(fmt.Sprintf("func (%s *%s) %s(input *%s) map[string]any {\n",
		g.receiverVar, g.receiverType, g.genFuncName, sourceTypeName))
}

// generateFunctionBody 生成函数体
func (g *Generator2) generateFunctionBody(builder *strings.Builder) {
	// 调用 ToPO
	builder.WriteString(fmt.Sprintf("\tb := %s.%s(input)\n", g.receiverVar, g.result.FuncName))

	// 获取 patch 字段
	builder.WriteString("\tfields := input.ExportPatch()\n")

	// 初始化 values
	totalMappings := len(g.result.AllMappings)
	builder.WriteString(fmt.Sprintf("\tvalues := make(map[string]any, %d)\n", totalMappings))

	// 按类型生成映射代码
	for _, group := range g.result.Groups {
		switch group.Type {
		case OneToOne:
			g.generateOneToOneMappings(builder, group)
		case Embedded:
			g.generateEmbeddedMappings(builder, group)
		case ManyToOne:
			g.generateManyToOneMappings(builder, group)
		case OneToMany:
			g.generateOneToManyMappings(builder, group)
		case MethodCall:
			g.generateMethodCallMappings(builder, group)
		}
	}
}

// generateOneToOneMappings 生成一对一映射代码
func (g *Generator2) generateOneToOneMappings(builder *strings.Builder, group MappingGroup) {
	for _, mapping := range group.Mappings {
		g.writeFieldMapping(builder, mapping.SourcePath, mapping.TargetPath, mapping.ColumnName)
	}
}

// generateEmbeddedMappings 生成嵌入字段映射代码
func (g *Generator2) generateEmbeddedMappings(builder *strings.Builder, group MappingGroup) {
	builder.WriteString(fmt.Sprintf("\t// Embedded: %s\n", group.TargetField))
	for _, mapping := range group.Mappings {
		g.writeFieldMapping(builder, mapping.SourcePath, mapping.TargetPath, mapping.ColumnName)
	}
}

// generateManyToOneMappings 生成多对一(JSON)映射代码
func (g *Generator2) generateManyToOneMappings(builder *strings.Builder, group MappingGroup) {
	// 需要 gsql 包
	g.imports["github.com/donutnomad/gsql"] = true

	// 获取 column 名称
	columnName := ""
	if len(group.Mappings) > 0 {
		columnName = group.Mappings[0].ColumnName
	}

	builder.WriteString(fmt.Sprintf("\t// B.%s\n", group.TargetField))
	builder.WriteString("\t{\n")
	builder.WriteString(fmt.Sprintf("\t\tset := gsql.JSONSet(\"%s\")\n", columnName))
	builder.WriteString(fmt.Sprintf("\t\tfield := b.%s.Data()\n", group.TargetField))

	// 按 JSONPath 的前缀分组（用于嵌套结构的注释）
	prefixGroups := g.groupByJSONPathPrefix(group.Mappings)

	for prefix, mappings := range prefixGroups {
		if prefix != "" {
			builder.WriteString(fmt.Sprintf("\t\t// %s\n", prefix))
		}
		for _, mapping := range mappings {
			// 从 TargetPath 获取字段路径 (去掉 JSON 字段名前缀)
			fieldPath := g.getJSONFieldPath(mapping)
			g.writeJSONFieldMapping(builder, mapping.SourcePath, mapping.JSONPath, fieldPath)
		}
	}

	builder.WriteString("\t\tif set.Len() > 0 {\n")
	builder.WriteString(fmt.Sprintf("\t\t\tvalues[\"%s\"] = set\n", columnName))
	builder.WriteString("\t\t}\n")
	builder.WriteString("\t}\n")
}

// generateOneToManyMappings 生成一对多映射代码
func (g *Generator2) generateOneToManyMappings(builder *strings.Builder, group MappingGroup) {
	builder.WriteString(fmt.Sprintf("\t// OneToMany: %s\n", group.SourceField))

	// 一对多：一个源字段展开为多个目标字段
	// 使用源字段的第一部分作为条件检查
	if len(group.Mappings) > 0 {
		builder.WriteString(fmt.Sprintf("\tif fields.%s.IsPresent() {\n", group.SourceField))
		for _, mapping := range group.Mappings {
			builder.WriteString(fmt.Sprintf("\t\tvalues[\"%s\"] = b.%s\n", mapping.ColumnName, mapping.TargetPath))
		}
		builder.WriteString("\t}\n")
	}
}

// generateMethodCallMappings 生成方法调用映射代码
func (g *Generator2) generateMethodCallMappings(builder *strings.Builder, group MappingGroup) {
	builder.WriteString(fmt.Sprintf("\t// MethodCall: %s() -> %s\n", group.MethodName, group.TargetField))

	// 方法调用映射：多个源字段通过方法组合为一个目标字段
	// 如果任一源字段被修改，则更新目标字段
	if len(group.Mappings) > 0 {
		// 生成条件检查：任一字段 IsPresent
		var conditions []string
		for _, mapping := range group.Mappings {
			conditions = append(conditions, fmt.Sprintf("fields.%s.IsPresent()", mapping.SourcePath))
		}

		columnName := group.Mappings[0].ColumnName
		builder.WriteString(fmt.Sprintf("\tif %s {\n", strings.Join(conditions, " || ")))
		builder.WriteString(fmt.Sprintf("\t\tvalues[\"%s\"] = b.%s\n", columnName, group.TargetField))
		builder.WriteString("\t}\n")
	}
}

// writeFieldMapping 写入字段映射
func (g *Generator2) writeFieldMapping(builder *strings.Builder, sourcePath, targetPath, columnName string) {
	builder.WriteString(fmt.Sprintf("\tif fields.%s.IsPresent() {\n", sourcePath))
	builder.WriteString(fmt.Sprintf("\t\tvalues[\"%s\"] = b.%s\n", columnName, targetPath))
	builder.WriteString("\t}\n")
}

// writeJSONFieldMapping 写入 JSON 字段映射
func (g *Generator2) writeJSONFieldMapping(builder *strings.Builder, sourcePath, jsonPath, fieldPath string) {
	builder.WriteString(fmt.Sprintf("\t\tif fields.%s.IsPresent() {\n", sourcePath))
	builder.WriteString(fmt.Sprintf("\t\t\tset.Set(\"%s\", field.%s)\n", jsonPath, fieldPath))
	builder.WriteString("\t\t}\n")
}

// groupByJSONPathPrefix 按 JSONPath 前缀分组
func (g *Generator2) groupByJSONPathPrefix(mappings []FieldMapping2) map[string][]FieldMapping2 {
	result := make(map[string][]FieldMapping2)

	for _, mapping := range mappings {
		prefix := ""
		if idx := strings.Index(mapping.JSONPath, "."); idx > 0 {
			prefix = mapping.JSONPath[:idx]
		}
		result[prefix] = append(result[prefix], mapping)
	}

	return result
}

// getJSONFieldPath 获取 JSON 字段路径
func (g *Generator2) getJSONFieldPath(mapping FieldMapping2) string {
	// JSONPath 是 json tag 路径，如 "author.name"
	// 需要转换为 Go 字段路径，如 "Author.Name"
	parts := strings.Split(mapping.JSONPath, ".")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, ".")
}

// validateFieldCoverage 验证字段覆盖情况，返回 Missing fields 注释
func (g *Generator2) validateFieldCoverage(generatedCode string) string {
	if len(g.result.TargetColumns) == 0 {
		return ""
	}

	// 收集所有被赋值的数据库字段名
	assignedFields := make(map[string]bool)

	// 查找所有的 values["字段名"] 赋值
	lines := strings.Split(generatedCode, "\n")
	for _, line := range lines {
		// 简单的字符串匹配来查找 values["字段名"] 模式
		if strings.Contains(line, "values[") && strings.Contains(line, "] =") {
			// 提取字段名
			start := strings.Index(line, `values["`) + 8
			if start > 7 { // 确保找到了 values["
				end := strings.Index(line[start:], `"]`)
				if end > 0 {
					fieldName := line[start : start+end]
					assignedFields[fieldName] = true
				}
			}
		}
	}

	// 检查哪些字段缺失
	var missingFields []string
	for _, expectedField := range g.result.TargetColumns {
		if !assignedFields[expectedField] {
			missingFields = append(missingFields, expectedField)
		}
	}

	if len(missingFields) == 0 {
		return ""
	}

	sort.Strings(missingFields)
	return fmt.Sprintf("// Missing fields: %s\n", strings.Join(missingFields, ", "))
}

// Generate2 使用新方案生成代码
// filePath: 源文件路径
// receiverType: 接收者类型名（如 "ListingPO"）
// funcName: 原函数名（如 "ToPO"）
// genFuncName: 生成的函数名（如 "ToPatch"）
func Generate2(filePath, receiverType, funcName, genFuncName string) (string, string, error) {
	// 解析映射关系
	result, err := Parse(filePath, receiverType, funcName)
	if err != nil {
		return "", "", fmt.Errorf("解析失败: %w", err)
	}

	// 生成代码
	generator := NewGenerator2(result, genFuncName)
	fullCode, funcCode := generator.Generate()

	return fullCode, funcCode, nil
}

// Generate2WithOptions 使用新方案生成代码（兼容旧 API 调用方式）
// funcNameWithReceiver: "ReceiverType.FuncName" 格式，如 "ListingPO.ToPO"
// genFuncName: 生成的函数名（如 "ToPatch"）
// options: 选项，支持 WithFileContext
func Generate2WithOptions(funcNameWithReceiver, genFuncName string, options ...Option) (string, string, error) {
	// 解析函数名格式
	parts := strings.Split(funcNameWithReceiver, ".")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("无效的函数名格式，期望 'ReceiverType.FuncName'，得到 '%s'", funcNameWithReceiver)
	}
	receiverType := parts[0]
	funcName := parts[1]

	// 获取文件路径
	var filePath string
	for _, opt := range options {
		if fp := opt(""); fp != "" {
			filePath = fp
		}
	}
	if filePath == "" {
		return "", "", fmt.Errorf("需要通过 WithFileContext 指定文件路径")
	}

	return Generate2(filePath, receiverType, funcName, genFuncName)
}
