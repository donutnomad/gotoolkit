package automap

import (
	"fmt"
	"testing"
)

func TestDebugResult(t *testing.T) {
	result, err := Parse("MapAToB")
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	fmt.Printf("=== 解析结果详情 ===\n")
	fmt.Printf("函数名: %s\n", result.FuncSignature.FuncName)
	fmt.Printf("映射关系数量: %d\n", len(result.MappingRelations))

	for i, rel := range result.MappingRelations {
		fmt.Printf("映射 %d: %s -> %v (JSON: %v)\n", i, rel.AField, rel.BFields, rel.IsJSONType)
	}

	fmt.Printf("一对一映射: %d\n", len(result.FieldMapping.OneToOne))
	fmt.Printf("一对多映射: %d\n", len(result.FieldMapping.OneToMany))
	fmt.Printf("JSON字段映射: %d\n", len(result.FieldMapping.JSONFields))

	for aField, bField := range result.FieldMapping.OneToOne {
		fmt.Printf("  %s -> %s\n", aField, bField)
	}

	for aField, bFields := range result.FieldMapping.OneToMany {
		fmt.Printf("  %s -> %v\n", aField, bFields)
	}

	for bField, jsonMapping := range result.FieldMapping.JSONFields {
		fmt.Printf("  JSON字段 %s: %s\n", bField, jsonMapping.FieldName)
		for aField, jsonField := range jsonMapping.SubFields {
			fmt.Printf("    %s -> %s\n", aField, jsonField)
		}
	}
}
