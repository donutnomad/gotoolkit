package automap

import (
	"fmt"
	"go/ast"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/donutnomad/gotoolkit/internal/xast"
	"github.com/samber/lo"
)

// CodeGenerator 代码生成器
type CodeGenerator struct {
	result       *ParseResult
	typeResolver *TypeResolver
	genFuncName  string
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
	cg.generateFunctionSignature(cg.genFuncName, &builder)

	// 生成函数体
	cg.generateFunctionBody(&builder)

	// 生成返回语句
	cg.generateReturnStatement(&builder)

	generatedCode := builder.String()

	// 验证字段覆盖情况
	if err := cg.validateFieldCoverage(generatedCode); err != nil {
		// 在生成的代码前添加错误注释
		errorMsg := fmt.Sprintf("// %s\n", err.Error())
		return errorMsg + generatedCode
	}

	return generatedCode
}

// generateFunctionSignature 生成函数签名
func (cg *CodeGenerator) generateFunctionSignature(genFuncName string, builder *strings.Builder) {
	aTypeName := cg.result.AType.Name
	if cg.result.AType.Package != "" && cg.result.AType.Package != cg.result.FuncSignature.PackageName {
		aTypeName = cg.result.AType.Package + "." + aTypeName
	}

	// 检查是否有接收者
	if cg.result.FuncSignature.Receiver != "" {
		// 生成带接收者的函数签名
		receiverVar := strings.ToLower(cg.result.FuncSignature.Receiver[:1])
		builder.WriteString(fmt.Sprintf("func (%s *%s) %s(input *%s) map[string]any {\n",
			receiverVar, cg.result.FuncSignature.Receiver, genFuncName, aTypeName))
	} else {
		// 生成普通函数签名
		builder.WriteString(fmt.Sprintf("func %s(input *%s) map[string]any {\n", genFuncName, aTypeName))
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
	builder.WriteString(fmt.Sprintf("\tvalues := make(map[string]any, %d)\n", len(slices.Collect(cg.result.BType.FieldIter()))))
}

func writeField(sb *strings.Builder, aField string, bFields []string, bField2Column map[string]*FieldInfo, comment string) {
	if len(comment) > 0 {
		sb.WriteString(comment)
	}
	sb.WriteString(fmt.Sprintf("\tif fields.%s.IsPresent() {\n", aField))
	// 生成多个字段的赋值
	for _, bField := range bFields {
		sb.WriteString(fmt.Sprintf("\t\tvalues[\"%s\"] = b.%s\n", bField2Column[bField].GetColumnName(), bField))
	}
	sb.WriteString("\t}\n")
}

func writeField2(sb *strings.Builder, aField string, bFieldName, bColumnName string, comment string) {
	if len(comment) > 0 {
		sb.WriteString(comment)
	}
	sb.WriteString(fmt.Sprintf("\tif fields.%s.IsPresent() {\n", aField))
	sb.WriteString(fmt.Sprintf("\t\tvalues[\"%s\"] = b.%s\n", bColumnName, bFieldName))
	sb.WriteString("\t}\n")
}

func writeField3(sb *strings.Builder,
	aField string,
	jsonTagName, bFieldPath string,
	comments ...string,
) {
	for _, comment := range comments {
		sb.WriteString(comment)
	}
	sb.WriteString(fmt.Sprintf("\tif fields.%s.IsPresent() {\n", aField))
	sb.WriteString(fmt.Sprintf("\t\tset.Set(\"%s\", %s)\n", jsonTagName, bFieldPath))
	sb.WriteString("\t}\n")
}

// generateFieldMappings 生成字段映射代码
func (cg *CodeGenerator) generateFieldMappings(builder *strings.Builder) {
	//var aFieldNames = cg.result.AType.FieldIter()
	var bField2Column = maps.Collect(cg.result.BType.FieldIter2())
	var bFieldNames []string
	for n := range cg.result.BType.FieldIter2() {
		bFieldNames = append(bFieldNames, n)
	}

	// B作为主导
	var used = make(map[string]bool)
OUT:
	for _, bField := range bFieldNames {
		for a, b := range cg.result.FieldMapping.OneToOne {
			if b == bField {
				writeField(builder, a, []string{bField}, bField2Column, "")
				continue OUT
			}
		}
		for a, bs := range cg.result.FieldMapping.OneToMany {
			if !used[a] && lo.Contains(bs, bField) {
				used[a] = true
				writeField(builder, a, bs, bField2Column, "// One To Many\n")
				continue OUT
			}
		}
	}

	// A作为主导
	//for aFieldName := range aFieldNames {
	//	bField, ok := cg.result.FieldMapping.OneToOne[aFieldName]
	//	if ok {
	//		writeField(builder, aFieldName, []string{bField}, bField2Column, "")
	//		continue
	//	}
	//	bFields, ok := cg.result.FieldMapping.OneToMany[aFieldName]
	//	if ok {
	//		writeField(builder, aFieldName, bFields, bField2Column, "// 1对多")
	//	}
	//}

	var findAField = func(bFieldName string) string {
		// 首先在一对一映射中查找
		for aField, bField := range cg.result.FieldMapping.OneToOne {
			if bField == bFieldName {
				return aField
			}
		}

		// 然后在一对多映射中查找
		for relName, relBFields := range cg.result.FieldMapping.OneToMany {
			for _, bf := range relBFields {
				if bf == bFieldName {
					return relName
				}
			}
		}

		// 最后在JSON字段映射中查找
		if _, exists := cg.result.FieldMapping.JSONFields[bFieldName]; exists {
			// 对于简单类型的JSON字段，SubFields可能为空，但我们可以通过映射关系查找
			// 查找映射关系中是否有对应的A字段
			for _, relation := range cg.result.MappingRelations {
				if relation.IsJSONType && relation.JSONField == bFieldName {
					return relation.AField
				}
			}
		}

		panic(fmt.Sprintf("找不到对应的A字段，B字段名: %s", bFieldName))
	}

	// B作为主导
	// 生成JSON字段映射（多对一）
	for bFieldName, jsonMapping := range cg.result.FieldMapping.JSONFields {
		bField, ok := bField2Column[bFieldName]
		if !ok {
			panic("cannot found b filed")
		}
		aGoField2BFieldPath := jsonMapping.SubFields
		//fmt.Println("映射为:", bFieldName, bField.GetColumnName())
		//fmt.Println("映射为:", bFieldName, bField.GetFullType())

		fullType := bField.GetFullType()

		// 检查是否有子字段，如果没有子字段说明是简单类型，直接赋值
		if len(jsonMapping.SubFields) == 0 || strings.Contains(fullType, "JSONSlice") {
			var aFieldName = findAField(bFieldName)
			var bColumnName = bField.GetColumnName()
			writeField2(builder, aFieldName, bFieldName, bColumnName, "")
			continue
		}
		if strings.Contains(fullType, "JSONType[") { // datatypes.JSONType， 使用JSONSet
			// 查找类型
			thisType := xast.GetFieldType((bField.ASTField.Type).(*ast.IndexExpr).Index, nil)
			thisTypeInfo := NewTypeInfoFromName(thisType)
			err := cg.typeResolver.ResolveTypeCurrent(thisTypeInfo)
			if err != nil {
				panic(fmt.Sprintf("cannot resolveType %s", thisType))
			}

			bColumnName := bField.GetColumnName()
			type tmpT struct {
				bFieldPath string
				aField     string
			}
			const normalKey = "0"
			var nestedMapping = make(map[string][]tmpT) // bFieldPath => aField
			for aField, bFieldPath := range aGoField2BFieldPath {
				parts := strings.Split(bFieldPath, ".")
				k := lo.Ternary(len(parts) == 1, normalKey, parts[0])
				nestedMapping[k] = append(nestedMapping[k], tmpT{bFieldPath, aField})
			}

			builder.WriteString(fmt.Sprintf("\t// B.%s\n", bFieldName))
			builder.WriteString("\t{\n")
			builder.WriteString(fmt.Sprintf("\tset := gsql.JSONSet(\"%s\")\n", bColumnName))
			builder.WriteString(fmt.Sprintf("\tfield := b.%s.Data()\n", bFieldName))

			keys := slices.Collect(maps.Keys(nestedMapping))
			sort.Strings(keys)

			for _, k := range keys {
				if k != normalKey {
					builder.WriteString(fmt.Sprintf("// %s\n", k))
					//builder.WriteString("\t{\n")
				}
				for _, tmp := range nestedMapping[k] {
					writeField3(builder, tmp.aField,
						cg.getJsonName(thisTypeInfo, tmp.bFieldPath),
						fmt.Sprintf("field.%s", tmp.bFieldPath),
					)
				}
				//if k != normalKey {
				//	builder.WriteString("\t}\n")
				//}
			}
			builder.WriteString("\tif set.Len() > 0 {\n")
			builder.WriteString(fmt.Sprintf("\tvalues[\"%s\"] = set\n", bColumnName))
			builder.WriteString("\t}\n")
			builder.WriteString("\t}")
		}
	}
}

func (cg *CodeGenerator) getJsonName(thisTypeInfo *TypeInfo, path string) (result string) {
	getF := func(typeInfo *TypeInfo, n string) *FieldInfo {
		for _, f := range typeInfo.Fields {
			if f.Name == n {
				return &f
			}
		}
		panic(fmt.Sprintf("cannot get json name %s from type %s", n, typeInfo.FullName))
	}
	var myTypeInfo = thisTypeInfo
	parts := strings.Split(path, ".")
	if len(parts) == 1 {
		return getF(myTypeInfo, path).GetJsonName()
	}
	// 使用当前类型的文件路径来解析import
	currentResolveFile := myTypeInfo.FilePath
	if currentResolveFile == "" {
		currentResolveFile = cg.typeResolver.currentFile
	}
	for i, part := range parts {
		jn := getF(myTypeInfo, part)
		if i == len(parts)-1 {
			result += jn.GetJsonName()
			continue
		}
		myTypeInfo = NewTypeInfoFromName(jn.GetFullType())
		// 使用字段定义所在文件的路径来解析类型，而不是当前文件路径
		resolveFile := currentResolveFile
		if resolveFile == "" {
			resolveFile = cg.typeResolver.currentFile
		}
		err := cg.typeResolver.ResolveType(myTypeInfo, resolveFile)
		if err != nil {
			panic(fmt.Sprintf("cannot resolveType %s %s: %v", part, jn.GetFullType(), err))
		}
		// 更新下一次解析使用的文件路径
		if myTypeInfo.FilePath != "" {
			currentResolveFile = myTypeInfo.FilePath
		}
		result += jn.GetJsonName() + "."
	}
	return result
}

// generateReturnStatement 生成返回语句
func (cg *CodeGenerator) generateReturnStatement(builder *strings.Builder) {
	builder.WriteString("\treturn values\n")
	builder.WriteString("}\n") // 函数结束括号
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
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
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
func (cg *CodeGenerator) GenerateFullCode(result *ParseResult) (string, string) {
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

	return builder.String(), generatedCode
}

// validateFieldCoverage 验证字段覆盖情况
func (cg *CodeGenerator) validateFieldCoverage(generatedCode string) error {
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

	// 获取应该在patch中检查的B类型字段名（只检查A类型patch中存在的字段对应的B字段）
	expectedFields := lo.Map(slices.Collect(
		maps.Values(maps.Collect(cg.result.BType.FieldIter2())),
	), func(item *FieldInfo, index int) string {
		return item.GetColumnName()
	})

	// 检查哪些字段缺失
	var missingFields []string
	for _, expectedField := range expectedFields {
		if !assignedFields[expectedField] {
			missingFields = append(missingFields, expectedField)
		}
	}

	sort.Strings(missingFields)
	if len(missingFields) > 0 {
		return fmt.Errorf("Missing fields: %v", strings.Join(missingFields, ", "))
	}

	return nil
}

// getEmbeddedDatabaseFields 获取嵌入结构体的数据库字段名
func (cg *CodeGenerator) getEmbeddedDatabaseFields(embeddedType string) []FieldInfo {
	typeInfo := NewTypeInfoFromName(embeddedType)
	err := cg.typeResolver.ResolveTypeCurrent(typeInfo)
	if err != nil {
		panic(fmt.Sprintf("cannot find typeInfo: %s", embeddedType))
	}
	return typeInfo.Fields
}
