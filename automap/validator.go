package automap

import (
	"fmt"
)

// Validator 验证器
type Validator struct {
	typeResolver *TypeResolver
}

// NewValidator 创建验证器
func NewValidator(typeResolver *TypeResolver) *Validator {
	return &Validator{
		typeResolver: typeResolver,
	}
}

// Validate 验证解析结果
//func (v *Validator) Validate(result *ParseResult) error {
// 验证ExportPatch方法存在
//if err := v.validateExportPatchMethod(&result.AType); err != nil {
//	return fmt.Errorf("ExportPatch方法验证失败: %w", err)
//}

//// 验证字段映射
//if err := v.validateFieldMapping(result); err != nil {
//	return fmt.Errorf("字段映射验证失败: %w", err)
//}

//// 验证映射关系完整性
//if err := v.validateMappingIntegrity(result); err != nil {
//	return fmt.Errorf("映射关系完整性验证失败: %w", err)
//}
//
//return nil
//}

//// validateExportPatchMethod 验证ExportPatch方法
//func (v *Validator) validateExportPatchMethod(typeInfo *TypeInfo) error {
//	if !v.typeResolver.HasExportPatchMethod(typeInfo) {
//		return fmt.Errorf("类型 %s 缺少ExportPatch方法", typeInfo.FullName)
//	}
//
//	// 进一步验证方法签名
//	exportPatchMethod := v.findExportPatchMethod(typeInfo)
//	if exportPatchMethod == nil {
//		return fmt.Errorf("未找到ExportPatch方法")
//	}
//
//	// 检查参数：应该没有参数
//	if len(exportPatchMethod.Params) != 0 {
//		return fmt.Errorf("ExportPatch方法不应该有参数")
//	}
//
//	// 检查返回值：应该有一个返回值，类型为*Patch
//	if len(exportPatchMethod.Returns) != 1 {
//		return fmt.Errorf("ExportPatch方法应该有且只有一个返回值")
//	}
//
//	returnType := exportPatchMethod.Returns[0]
//	if !v.isPatchType(returnType) {
//		return fmt.Errorf("ExportPatch方法应该返回*Patch类型，实际返回: %s", returnType.Name)
//	}
//
//	return nil
//}

//// findExportPatchMethod 查找ExportPatch方法
//func (v *Validator) findExportPatchMethod(typeInfo *TypeInfo) *MethodInfo {
//	for _, method := range typeInfo.Methods {
//		if method.Name == "ExportPatch" && method.IsExported {
//			return &method
//		}
//	}
//	return nil
//}

//// isPatchType 检查是否为Patch类型
//func (v *Validator) isPatchType(typeInfo TypeInfo) bool {
//	// 检查是否为指针类型
//	if !strings.HasPrefix(typeInfo.Name, "*") {
//		return false
//	}
//
//	// 检查类型名是否以Patch结尾
//	baseType := strings.TrimPrefix(typeInfo.Name, "*")
//	return strings.HasSuffix(baseType, "Patch")
//}

//// validateFieldMapping 验证字段映射
//func (v *Validator) validateFieldMapping(result *ParseResult) error {
//	// 验证一对一映射，跳过无效的映射
//	for aField, bField := range result.FieldMapping.OneToOne {
//		if !v.isValidAField(aField, &result.AType) {
//			continue // 跳过无效的A字段
//		}
//		if !v.isValidBField(bField, &result.BType) {
//			continue // 跳过无效的B字段
//		}
//	}
//
//	// 验证一对多映射
//	for aField, bFields := range result.FieldMapping.OneToMany {
//		if !v.isValidAField(aField, &result.AType) {
//			return fmt.Errorf("无效的A字段: %s", aField)
//		}
//		for _, bField := range bFields {
//			if !v.isValidBField(bField, &result.BType) {
//				return fmt.Errorf("无效的B字段: %s", bField)
//			}
//		}
//	}
//
//	// 验证多对一映射
//	for bField, aFields := range result.FieldMapping.ManyToOne {
//		if !v.isValidBField(bField, &result.BType) {
//			return fmt.Errorf("无效的B字段: %s", bField)
//		}
//		for _, aField := range aFields {
//			if !v.isValidAField(aField, &result.AType) {
//				return fmt.Errorf("无效的A字段: %s", aField)
//			}
//		}
//	}
//
//	// 验证JSON字段映射
//	for bField, jsonMapping := range result.FieldMapping.JSONFields {
//		if !v.isValidBField(bField, &result.BType) {
//			return fmt.Errorf("无效的B字段: %s", bField)
//		}
//		if jsonMapping.FieldName == "" {
//			return fmt.Errorf("JSON字段映射缺少字段名")
//		}
//		if len(jsonMapping.SubFields) == 0 {
//			return fmt.Errorf("JSON字段映射缺少子字段映射")
//		}
//		for aField := range jsonMapping.SubFields {
//			if !v.isValidAField(aField, &result.AType) {
//				return fmt.Errorf("无效的A字段: %s", aField)
//			}
//		}
//	}
//
//	return nil
//}

//// isValidAField 检查是否为有效的A字段
//func (v *Validator) isValidAField(fieldName string, aType *TypeInfo) bool {
//	if fieldName == "" {
//		return false
//	}
//
//	// 检查字段是否存在于A类型中
//	for _, field := range aType.Fields {
//		if field.Name == fieldName {
//			return true
//		}
//	}
//
//	return false
//}

//// isValidBField 检查是否为有效的B字段
//func (v *Validator) isValidBField(fieldName string, bType *TypeInfo) bool {
//	if fieldName == "" {
//		return false
//	}
//
//	// 检查简单字段名
//	for _, field := range bType.Fields {
//		if field.Name == fieldName {
//			return true
//		}
//	}
//
//	// 检查嵌入字段（如 Model.ID）
//	if dotIndex := strings.Index(fieldName, "."); dotIndex != -1 {
//		embeddedFieldName := fieldName[:dotIndex]
//		subFieldName := fieldName[dotIndex+1:]
//
//		// 查找嵌入字段，这里应该更宽松地匹配
//		for _, field := range bType.Fields {
//			// 检查字段名或类型名是否匹配
//			if field.IsEmbedded && (field.Name == embeddedFieldName || field.Type == embeddedFieldName ||
//				strings.Contains(field.Type, embeddedFieldName) || strings.Contains(embeddedFieldName, field.Type)) {
//				// 这里简单检查子字段是否合理，更完整的实现需要解析嵌入结构体
//				if subFieldName != "" {
//					return true
//				}
//			}
//		}
//	}
//
//	return false
//}

//// validateMappingIntegrity 验证映射关系完整性
//func (v *Validator) validateMappingIntegrity(result *ParseResult) error {
//	// 检查是否有重复的映射关系
//	visitedPairs := make(map[string]bool)
//
//	// 检查一对一映射
//	for aField, bField := range result.FieldMapping.OneToOne {
//		pair := aField + "->" + bField
//		if visitedPairs[pair] {
//			return fmt.Errorf("重复的映射关系: %s", pair)
//		}
//		visitedPairs[pair] = true
//	}
//
//	// 检查一对多映射
//	for aField, bFields := range result.FieldMapping.OneToMany {
//		for _, bField := range bFields {
//			pair := aField + "->" + bField
//			if visitedPairs[pair] {
//				return fmt.Errorf("重复的映射关系: %s", pair)
//			}
//			visitedPairs[pair] = true
//		}
//	}
//
//	// 检查多对一映射
//	for bField, aFields := range result.FieldMapping.ManyToOne {
//		for _, aField := range aFields {
//			pair := aField + "->" + bField
//			if visitedPairs[pair] {
//				return fmt.Errorf("重复的映射关系: %s", pair)
//			}
//			visitedPairs[pair] = true
//		}
//	}
//
//	// 验证映射关系的逻辑一致性
//	if err := v.validateMappingConsistency(result); err != nil {
//		return err
//	}
//
//	return nil
//}

//// validateMappingConsistency 验证映射关系逻辑一致性
//func (v *Validator) validateMappingConsistency(result *ParseResult) error {
//	// 检查是否存在逻辑冲突
//	// 例如：一个字段既被标记为一对一，又被标记为一对多
//
//	allMappings := make(map[string][]string)
//
//	// 收集所有映射关系
//	for aField, bField := range result.FieldMapping.OneToOne {
//		allMappings[aField] = append(allMappings[aField], bField)
//	}
//
//	for aField, bFields := range result.FieldMapping.OneToMany {
//		allMappings[aField] = append(allMappings[aField], bFields...)
//	}
//
//	// 检查是否存在冲突
//	for aField, bFields := range allMappings {
//		uniqueBFields := make(map[string]bool)
//		for _, bField := range bFields {
//			if uniqueBFields[bField] {
//				return fmt.Errorf("字段 %s 存在重复映射到 %s", aField, bField)
//			}
//			uniqueBFields[bField] = true
//		}
//	}
//
//	return nil
//}

// ValidateFunctionSignature 验证函数签名
func (v *Validator) ValidateFunctionSignature(sig *FuncSignature) error {
	// 验证函数名
	if sig.FuncName == "" {
		return fmt.Errorf("函数名不能为空")
	}

	// 验证输入类型
	if sig.InputType.Name == "" {
		return fmt.Errorf("输入类型不能为空")
	}

	// 验证输出类型
	if sig.OutputType.Name == "" {
		return fmt.Errorf("输出类型不能为空")
	}

	// 验证输入输出类型不能相同
	if sig.InputType.FullName == sig.OutputType.FullName {
		return fmt.Errorf("输入类型和输出类型不能相同")
	}

	return nil
}

// ValidateTypes 验证类型信息
func (v *Validator) ValidateTypes(aType, bType *TypeInfo) error {
	// 验证A类型
	if err := v.validateSingleType(aType, "A"); err != nil {
		return err
	}

	// 验证B类型
	if err := v.validateSingleType(bType, "B"); err != nil {
		return err
	}

	return nil
}

// validateSingleType 验证单个类型
func (v *Validator) validateSingleType(typeInfo *TypeInfo, typeName string) error {
	if typeInfo.Name == "" {
		return fmt.Errorf("%s类型名不能为空", typeName)
	}

	if typeInfo.FullName == "" {
		return fmt.Errorf("%s类型全名不能为空", typeName)
	}

	if len(typeInfo.Fields) == 0 {
		return fmt.Errorf("%s类型没有字段定义", typeName)
	}

	// 检查字段名重复
	fieldNames := make(map[string]bool)
	for _, field := range typeInfo.Fields {
		if field.Name == "" {
			continue // 跳过嵌入字段的空名
		}
		if fieldNames[field.Name] {
			return fmt.Errorf("%s类型存在重复字段名: %s", typeName, field.Name)
		}
		fieldNames[field.Name] = true
	}

	return nil
}
