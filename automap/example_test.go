package automap

import (
	"fmt"
	"testing"
)

// TestExampleUsage 测试示例用法
func TestExampleUsage(t *testing.T) {
	// 使用默认实例解析映射函数
	result, err := Parse("MapAToB")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 验证结果
	if result.FuncSignature.FuncName != "MapAToB" {
		t.Errorf("期望函数名: MapAToB, 实际: %s", result.FuncSignature.FuncName)
	}

	if !result.HasExportPatch {
		t.Error("期望有ExportPatch方法")
	}

	// 打印结果信息
	fmt.Printf("=== 解析结果 ===\n")
	fmt.Printf("函数名: %s\n", result.FuncSignature.FuncName)
	fmt.Printf("输入类型: %s\n", result.AType.Name)
	fmt.Printf("输出类型: %s\n", result.BType.Name)
	fmt.Printf("是否有ExportPatch: %v\n", result.HasExportPatch)
	fmt.Printf("字段映射数量: 一对一(%d), 一对多(%d), JSON字段(%d)\n",
		len(result.FieldMapping.OneToOne),
		len(result.FieldMapping.OneToMany),
		len(result.FieldMapping.JSONFields))

	// 生成完整代码
	fullCode, err := ParseAndGenerate("MapAToB")
	if err != nil {
		t.Fatalf("生成代码失败: %v", err)
	}

	fmt.Printf("\n=== 生成的代码 ===\n")
	fmt.Printf("%s\n", fullCode)
}

// TestDetailedMappingAnalysis 测试详细的映射分析
func TestDetailedMappingAnalysis(t *testing.T) {
	automap := New()
	result, err := automap.Parse("MapAToB")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	fmt.Printf("\n=== 详细映射分析 ===\n")

	// 显示一对一映射
	fmt.Printf("一对一映射:\n")
	for aField, bField := range result.FieldMapping.OneToOne {
		fmt.Printf("  %s -> %s\n", aField, bField)
	}

	// 显示一对多映射
	fmt.Printf("一对多映射:\n")
	for aField, bFields := range result.FieldMapping.OneToMany {
		fmt.Printf("  %s -> %v\n", aField, bFields)
	}

	// 显示JSON字段映射
	fmt.Printf("JSON字段映射:\n")
	for bField, jsonMapping := range result.FieldMapping.JSONFields {
		fmt.Printf("  %s (%s):\n", bField, jsonMapping.FieldName)
		for aField, jsonField := range jsonMapping.SubFields {
			fmt.Printf("    %s -> %s\n", aField, jsonField)
		}
	}

	// 显示A类型字段
	fmt.Printf("\nA类型字段:\n")
	for _, field := range result.AType.Fields {
		fmt.Printf("  %s: %s (嵌入: %v)\n", field.Name, field.Type, field.IsEmbedded)
	}

	// 显示B类型字段
	fmt.Printf("\nB类型字段:\n")
	for _, field := range result.BType.Fields {
		gormInfo := ""
		if field.GormTag != "" {
			gormInfo = fmt.Sprintf(" [gorm: %s]", field.GormTag)
		}
		if field.ColumnName != "" {
			gormInfo += fmt.Sprintf(" [column: %s]", field.ColumnName)
		}
		fmt.Printf("  %s: %s%s (JSONType: %v)\n", field.Name, field.Type, gormInfo, field.IsJSONType)
	}
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	automap := New()

	// 测试解析不存在的函数
	_, err := automap.Parse("NonExistentFunction")
	if err == nil {
		t.Error("期望解析不存在的函数会失败，但成功了")
	} else {
		fmt.Printf("预期的错误: %v\n", err)
	}

	// 测试空函数名
	_, err = automap.Parse("")
	if err == nil {
		t.Error("期望空函数名会失败，但成功了")
	} else {
		fmt.Printf("预期的错误: %v\n", err)
	}
}
