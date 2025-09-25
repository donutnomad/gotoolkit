package main

import (
	"fmt"
	"os"
)

func main() {
	// 测试解析功能
	structInfo, err := parseStruct("test_simple.go", "TestStruct")
	if err != nil {
		fmt.Printf("解析失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("结构体名称: %s\n", structInfo.Name)
	fmt.Printf("字段数量: %d\n", len(structInfo.Fields))

	for i, field := range structInfo.Fields {
		fmt.Printf("字段 %d: %s (%s) - Tag: %s\n", i+1, field.Name, field.Type, field.Tag)
	}
}
