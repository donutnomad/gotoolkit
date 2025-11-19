package main

import (
	"github.com/donutnomad/gotoolkit/internal/structparse"
)

// FieldInfo 表示结构体字段信息
type FieldInfo = structparse.FieldInfo

// StructInfo 表示结构体信息
type StructInfo = structparse.StructInfo

// parseStruct 解析指定文件中的结构体
func parseStruct(filename, structName string) (*StructInfo, error) {
	return structparse.ParseStruct(filename, structName)
}
