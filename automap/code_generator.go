package automap

import (
	"fmt"
	"strings"
)

// CodeGenerator 代码生成器
type CodeGenerator struct {
	result       *ParseResult
	typeResolver *TypeResolver
}

// NewCodeGenerator 创建代码生成器
func NewCodeGenerator(typeResolver *TypeResolver) *CodeGenerator {
	return &CodeGenerator{
		typeResolver: typeResolver,
	}
}

// Generate 生成映射代码
func (cg *CodeGenerator) Generate(result *ParseResult) string {
	cg.result = result

	var builder strings.Builder

	// 生成函数签名
	cg.generateFunctionSignature(&builder)

	// 生成函数体
	cg.generateFunctionBody(&builder)

	// 生成返回语句
	cg.generateReturnStatement(&builder)

	return builder.String()
}

// generateFunctionSignature 生成函数签名
func (cg *CodeGenerator) generateFunctionSignature(builder *strings.Builder) {
	aTypeName := cg.result.AType.Name
	if cg.result.AType.Package != "" && cg.result.AType.Package != cg.result.FuncSignature.PackageName {
		aTypeName = cg.result.AType.Package + "." + aTypeName
	}

	// 检查是否有接收者
	if cg.result.FuncSignature.Receiver != "" {
		// 生成带接收者的函数签名
		receiverVar := strings.ToLower(cg.result.FuncSignature.Receiver[:1])
		builder.WriteString(fmt.Sprintf("func (%s *%s) Do(input *%s) map[string]any {\n",
			receiverVar, cg.result.FuncSignature.Receiver, aTypeName))
	} else {
		// 生成普通函数签名
		builder.WriteString(fmt.Sprintf("func Do(input *%s) map[string]any {\n", aTypeName))
	}
}

// generateFunctionBody 生成函数体
func (cg *CodeGenerator) generateFunctionBody(builder *strings.Builder) {
	// 生成调用映射函数的代码
	cg.generateMapperCall(builder)

	// 生成获取patch的代码
	cg.generatePatchCall(builder)

	// 生成初始化返回值的代码
	cg.generateResultInit(builder)

	// 生成字段映射代码
	cg.generateFieldMappings(builder)
}

// generateMapperCall 生成调用映射函数的代码
func (cg *CodeGenerator) generateMapperCall(builder *strings.Builder) {
	funcName := cg.result.FuncSignature.FuncName
	if cg.result.FuncSignature.Receiver != "" {
		// 使用接收者变量调用方法
		receiverVar := strings.ToLower(cg.result.FuncSignature.Receiver[:1])
		funcName = receiverVar + "." + funcName
	}

	builder.WriteString(fmt.Sprintf("\tb := %s(input)\n", funcName))
}

// generatePatchCall 生成获取patch的代码
func (cg *CodeGenerator) generatePatchCall(builder *strings.Builder) {
	builder.WriteString("\tfields := input.ExportPatch()\n")
}

// generateResultInit 生成初始化返回值的代码
func (cg *CodeGenerator) generateResultInit(builder *strings.Builder) {
	builder.WriteString("\tvar ret = make(map[string]any)\n")
}

// generateFieldMappings 生成字段映射代码
func (cg *CodeGenerator) generateFieldMappings(builder *strings.Builder) {
	// 按照函数中的赋值顺序生成字段映射
	for _, rel := range cg.result.FieldMapping.OrderedRelations {
		if len(rel.BFields) == 1 {
			// 一对一映射
			builder.WriteString(fmt.Sprintf("\tif fields.%s.IsPresent() {\n", rel.AField))
			bFieldKey := cg.getBFieldKey(rel.BFields[0])
			builder.WriteString(fmt.Sprintf("\t\tret[\"%s\"] = b.%s\n", bFieldKey, rel.BFields[0]))
			builder.WriteString("\t}")
		} else if len(rel.BFields) > 1 {
			// 一对多映射
			builder.WriteString(fmt.Sprintf("\t// A的一个字段，对应B的多字段\n"))
			builder.WriteString(fmt.Sprintf("\tif fields.%s.IsPresent() {\n", rel.AField))
			for _, bField := range rel.BFields {
				bFieldKey := cg.getBFieldKey(bField)
				builder.WriteString(fmt.Sprintf("\t\tret[\"%s\"] = b.%s\n", bFieldKey, bField))
			}
			builder.WriteString("\t}")
		}
		builder.WriteString("\n")
	}

	// 生成JSON字段映射（多对一）
	cg.generateJSONFieldMappings(builder)
}

// generateOneToOneMappings 生成一对一映射
func (cg *CodeGenerator) generateOneToOneMappings(builder *strings.Builder) {
	for aField, bField := range cg.result.FieldMapping.OneToOne {
		// 添加注释
		cg.generateMappingComment(builder, aField, []string{bField}, MappingOneToOne)

		// 生成条件检查和赋值代码
		builder.WriteString(fmt.Sprintf("\tif fields.%s.IsPresent() {\n", cg.toPascalCase(aField)))

		// 获取B字段的键名
		bFieldKey := cg.getBFieldKey(bField)
		builder.WriteString(fmt.Sprintf("\t\tret[\"%s\"] = b.%s\n", bFieldKey, bField))
		builder.WriteString("\t}\n\n")
	}
}

// generateOneToManyMappings 生成一对多映射
func (cg *CodeGenerator) generateOneToManyMappings(builder *strings.Builder) {
	for aField, bFields := range cg.result.FieldMapping.OneToMany {
		// 添加注释
		cg.generateMappingComment(builder, aField, bFields, MappingOneToMany)

		// 生成条件检查
		builder.WriteString(fmt.Sprintf("\tif fields.%s.IsPresent() {\n", cg.toPascalCase(aField)))

		// 生成多个字段的赋值
		for _, bField := range bFields {
			bFieldKey := cg.getBFieldKey(bField)
			builder.WriteString(fmt.Sprintf("\t\tret[\"%s\"] = b.%s\n", bFieldKey, bField))
		}
		builder.WriteString("\t}\n\n")
	}
}

// generateJSONFieldMappings 生成JSON字段映射
func (cg *CodeGenerator) generateJSONFieldMappings(builder *strings.Builder) {
	// 检查是否有JSON字段映射
	if len(cg.result.FieldMapping.JSONFields) == 0 {
		return
	}

	// 直接按照JSON字段的顺序生成映射，这样更可靠
	for bFieldName, jsonMapping := range cg.result.FieldMapping.JSONFields {
		// 检查是否有子字段，如果没有子字段说明是简单类型，直接赋值
		if len(jsonMapping.SubFields) == 0 {
			// 简单类型的JSON字段，直接赋值
			jsonColumnName := cg.getBFieldColumnKey(bFieldName)
			if jsonColumnName == "" {
				jsonColumnName = cg.toSnakeCase(bFieldName)
			}

			// 找到对应的A字段名
			var aFieldName string
			for relName, relBFields := range cg.result.FieldMapping.OneToMany {
				for _, bf := range relBFields {
					if bf == bFieldName {
						aFieldName = relName
						break
					}
				}
			}
			if aFieldName == "" {
				// 如果找不到，尝试从OrderedRelations中查找
				for _, rel := range cg.result.FieldMapping.OrderedRelations {
					for _, bf := range rel.BFields {
						if bf == bFieldName {
							aFieldName = rel.AField
							break
						}
					}
				}
			}

			if aFieldName != "" {
				builder.WriteString(fmt.Sprintf("\tif fields.%s.IsPresent() {\n", aFieldName))
				builder.WriteString(fmt.Sprintf("\t\tret[\"%s\"] = b.%s\n", jsonColumnName, bFieldName))
				builder.WriteString("\t}\n")
			}
		} else {
			// 复杂类型的JSON字段，使用JSONSet
			// 添加注释
			builder.WriteString(fmt.Sprintf("\t// B.%s字段对应A的多字段\n", bFieldName))
			builder.WriteString("\t{\n")

			// 获取JSON字段在数据库中的列名（使用B类型字段的ColumnName）
			jsonColumnName := cg.getBFieldColumnKey(bFieldName)
			if jsonColumnName == "" {
				jsonColumnName = jsonMapping.FieldName
				if jsonColumnName == "" {
					jsonColumnName = cg.toSnakeCase(bFieldName)
				}
			}

			// 分析嵌套字段结构
			nestedFields := cg.analyzeNestedFields(jsonMapping.SubFields)

			// 生成简单字段的映射
			if len(nestedFields.SimpleFields) > 0 {
				builder.WriteString(fmt.Sprintf("\t\tset := datatypes.JSONSet(\"%s\")\n", jsonColumnName))

				for aField, jsonSubField := range nestedFields.SimpleFields {
					jsonFieldValue := cg.buildJSONFieldValue(bFieldName, jsonSubField)
					builder.WriteString(fmt.Sprintf("\t\tif fields.%s.IsPresent() {\n", aField))
					builder.WriteString(fmt.Sprintf("\t\t\tret[\"%s\"] = set.Set(\"%s\", %s)\n",
						jsonColumnName, jsonSubField, jsonFieldValue))
					builder.WriteString("\t\t}\n")
				}
			}

			// 生成嵌套字段的映射
			for nestedField, subFieldMapping := range nestedFields.NestedFields {
				// 构建嵌套对象的设置条件
				var conditions []string
				for aField := range subFieldMapping {
					conditions = append(conditions, fmt.Sprintf("fields.%s.IsPresent()", aField))
				}

				if len(conditions) > 0 {
					conditionStr := strings.Join(conditions, " && ")
					builder.WriteString(fmt.Sprintf("\t\tif %s {\n", conditionStr))

					// 构建嵌套对象值
					nestedValue := cg.buildNestedObjectValue(bFieldName, nestedField, subFieldMapping)
					// 使用snake_case的JSON字段名
					jsonNestedField := cg.toSnakeCase(nestedField)
					builder.WriteString(fmt.Sprintf("\t\t\tret[\"%s\"] = set.Set(\"%s\", %s)\n",
						jsonColumnName, jsonNestedField, nestedValue))
					builder.WriteString("\t\t}\n")
				}
			}

			builder.WriteString("\t}")
		}
	}
}

// generateMappingComment 生成映射注释
func (cg *CodeGenerator) generateMappingComment(builder *strings.Builder, aField string, bFields []string, mappingType MappingType) {
	switch mappingType {
	case MappingOneToOne:
		builder.WriteString(fmt.Sprintf("\t// A的一个字段，对应B的一个字段\n"))
	case MappingOneToMany:
		builder.WriteString(fmt.Sprintf("\t// A的一个字段，对应B的多个字段\n"))
	case MappingManyToOne:
		builder.WriteString(fmt.Sprintf("\t// B的一个字段，对应A的多个字段\n"))
	}
}

// generateReturnStatement 生成返回语句
func (cg *CodeGenerator) generateReturnStatement(builder *strings.Builder) {
	builder.WriteString("\t\nreturn ret\n")
	builder.WriteString("}\n") // 函数结束括号
}

// getBFieldKey 获取B字段的键名
func (cg *CodeGenerator) getBFieldKey(fieldName string) string {
	return cg.getBFieldColumnKey(fieldName)
}

// getBFieldColumnKey 获取B字段的GORM列名
func (cg *CodeGenerator) getBFieldColumnKey(fieldName string) string {
	// 处理嵌入字段（如 Model.ID）- 直接使用内部字段名
	if dotIndex := strings.Index(fieldName, "."); dotIndex != -1 {
		subFieldName := fieldName[dotIndex+1:]
		// 对于嵌入字段，直接使用内部字段名的列名
		return cg.toSnakeCase(subFieldName)
	}

	// 在B类型中查找字段
	for _, field := range cg.result.BType.Fields {
		if field.Name == fieldName {
			if field.ColumnName != "" {
				return field.ColumnName
			}
			return cg.toSnakeCase(field.Name)
		}
	}

	// 如果找不到字段，使用默认规则
	return cg.toSnakeCase(fieldName)
}

// getEmbeddedFieldColumnKey 获取嵌入字段中子字段的GORM列名
func (cg *CodeGenerator) getEmbeddedFieldColumnKey(embeddedType, subFieldName string) string {
	// 先尝试一些常见的内置类型
	if embeddedType == "Model" || embeddedType == "orm.Model" {
		// 对于orm.Model，我们知道它的字段结构
		switch subFieldName {
		case "ID":
			return "id"
		case "CreatedAt":
			return "created_at"
		case "UpdatedAt":
			return "updated_at"
		case "DeletedAt":
			return "deleted_at"
		default:
			return cg.toSnakeCase(subFieldName)
		}
	}

	if cg.typeResolver == nil {
		return cg.toSnakeCase(subFieldName)
	}

	// 尝试解析嵌入类型
	embeddedTypeInfo := &TypeInfo{
		Name: embeddedType,
	}

	// 使用当前目录解析嵌入类型
	err := cg.typeResolver.ResolveType(embeddedTypeInfo, ".")
	if err != nil {
		// 解析失败，使用默认转换
		return cg.toSnakeCase(subFieldName)
	}

	// 在嵌入类型的字段中查找子字段
	for _, field := range embeddedTypeInfo.Fields {
		if field.Name == subFieldName {
			if field.ColumnName != "" {
				return field.ColumnName
			}
			return cg.toSnakeCase(field.Name)
		}
	}

	// 找不到字段，使用默认转换
	return cg.toSnakeCase(subFieldName)
}

// toPascalCase 转换为PascalCase
func (cg *CodeGenerator) toPascalCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// toSnakeCase 转换为snake_case
func (cg *CodeGenerator) toSnakeCase(s string) string {
	// 处理常见缩写词和全大写的情况
	if s == "ID" {
		return "id"
	}

	// 处理以ID结尾的情况，如UserID -> user_id
	if strings.HasSuffix(s, "ID") && len(s) > 2 {
		prefix := s[:len(s)-2]
		return cg.toSnakeCase(prefix) + "_id"
	}

	// 处理其他常见缩写
	commonAbbrevs := map[string]string{
		"CreatedAt": "created_at",
		"UpdatedAt": "updated_at",
		"DeletedAt": "deleted_at",
	}

	if replacement, exists := commonAbbrevs[s]; exists {
		return replacement
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

// buildJSONFieldValue 构建B结构体中JSON字段的访问路径
func (cg *CodeGenerator) buildJSONFieldValue(bField, jsonSubField string) string {
	// 将JSON子字段名转换为Go字段名格式
	// 例如：name -> Name, symbol -> Symbol, total_supply -> TotalSupply
	goFieldName := cg.jsonNameToGoFieldName(jsonSubField)

	// 对于datatypes.JSONType，需要调用Data()方法来获取JSON数据
	return fmt.Sprintf("b.%s.Data().%s", bField, goFieldName)
}

// jsonNameToGoFieldName 将JSON字段名转换为Go字段名
func (cg *CodeGenerator) jsonNameToGoFieldName(jsonName string) string {
	// 转换：首字母大写，下划线后字母大写
	parts := strings.Split(jsonName, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}

// GenerateWithTemplate 使用模板生成代码
func (cg *CodeGenerator) GenerateWithTemplate(result *ParseResult, template string) (string, error) {
	// TODO: 实现基于模板的代码生成
	cg.result = result

	// 简单的模板替换
	code := template
	code = strings.ReplaceAll(code, "{{.ATypeName}}", result.AType.Name)
	code = strings.ReplaceAll(code, "{{.BTypeName}}", result.BType.Name)
	code = strings.ReplaceAll(code, "{{.FuncName}}", result.FuncSignature.FuncName)
	code = strings.ReplaceAll(code, "{{.MapperCall}}", cg.generateMapperCallString())
	code = strings.ReplaceAll(code, "{{.FieldMappings}}", cg.generateFieldMappingsString())

	return code, nil
}

// generateMapperCallString 生成映射调用字符串
func (cg *CodeGenerator) generateMapperCallString() string {
	funcName := cg.result.FuncSignature.FuncName
	if cg.result.FuncSignature.Receiver != "" {
		funcName = cg.result.FuncSignature.Receiver + "." + funcName
	}
	return fmt.Sprintf("b := %s(input)", funcName)
}

// generateFieldMappingsString 生成字段映射字符串
func (cg *CodeGenerator) generateFieldMappingsString() string {
	var builder strings.Builder
	cg.generateFieldMappings(&builder)
	return builder.String()
}

// NestedFieldAnalysis 嵌套字段分析结果
type NestedFieldAnalysis struct {
	SimpleFields map[string]string            // 简单字段：A字段 -> JSON字段
	NestedFields map[string]map[string]string // 嵌套字段：嵌套名 -> (A字段 -> JSON子字段)
}

// analyzeNestedFields 分析嵌套字段结构
func (cg *CodeGenerator) analyzeNestedFields(subFields map[string]string) NestedFieldAnalysis {
	analysis := NestedFieldAnalysis{
		SimpleFields: make(map[string]string),
		NestedFields: make(map[string]map[string]string),
	}

	// 通过分析JSON字段名来识别嵌套结构
	// 例如：issuer.issuer_name 表示嵌套在 issuer 中的 issuer_name
	for aField, jsonField := range subFields {
		if dotIndex := strings.Index(jsonField, "."); dotIndex != -1 {
			// 这是一个嵌套字段
			nestedField := jsonField[:dotIndex]
			subFieldName := jsonField[dotIndex+1:]

			if analysis.NestedFields[nestedField] == nil {
				analysis.NestedFields[nestedField] = make(map[string]string)
			}
			analysis.NestedFields[nestedField][aField] = subFieldName
		} else {
			// 这是一个简单字段
			analysis.SimpleFields[aField] = jsonField
		}
	}

	return analysis
}

// buildNestedObjectValue 构建嵌套对象值
func (cg *CodeGenerator) buildNestedObjectValue(bFieldName, nestedField string, subFieldMapping map[string]string) string {
	// 直接使用嵌套对象，例如：b.Attribute.Data().SupportResource 或 b.Attribute.Data().Issuer
	return fmt.Sprintf("b.%s.Data().%s", bFieldName, nestedField)
}

// getNestedType 获取嵌套类型名
func (cg *CodeGenerator) getNestedType(nestedField string) string {
	// 根据JSON字段名推断嵌套类型
	switch nestedField {
	case "SupportResource":
		return "SupportResourceInfo"
	case "Issuer":
		return "IssuerInfo"
	default:
		// 默认使用首字母大写的格式
		return cg.toPascalCase(nestedField)
	}
}

// GenerateImports 生成需要的导入语句
func (cg *CodeGenerator) GenerateImports(result *ParseResult) []string {
	imports := make(map[string]bool)

	// 检查是否需要datatypes包
	for _, field := range result.BType.Fields {
		if field.IsJSONType {
			imports["gorm.io/datatypes"] = true
			break
		}
	}

	// 检查是否需要其他包
	if strings.Contains(result.GeneratedCode, "datatypes.JSONSet") {
		imports["gorm.io/datatypes"] = true
	}

	var resultImports []string
	for imp := range imports {
		resultImports = append(resultImports, imp)
	}

	return resultImports
}

// GenerateFullCode 生成完整的代码（包含导入）
func (cg *CodeGenerator) GenerateFullCode(result *ParseResult) string {
	imports := cg.GenerateImports(result)
	generatedCode := cg.Generate(result)

	var builder strings.Builder

	// 生成导入语句
	if len(imports) > 0 {
		builder.WriteString("import (\n")
		for _, imp := range imports {
			builder.WriteString(fmt.Sprintf("\t\"%s\"\n", imp))
		}
		builder.WriteString(")\n\n")
	}

	// 生成生成的代码
	builder.WriteString(generatedCode)

	return builder.String()
}

// ValidateGeneratedCode 验证生成的代码
func (cg *CodeGenerator) ValidateGeneratedCode(code string) error {
	// TODO: 实现生成代码的验证逻辑
	// 可以使用go/parser解析生成的代码，检查语法是否正确
	return nil
}
