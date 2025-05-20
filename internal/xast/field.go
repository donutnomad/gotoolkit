package xast

import "go/ast"

type MyField struct {
	*ast.Field
	StructType *StructType
}

func (m *MyField) CollectImports() []string {
	return m.StructType.CollectImports(m.Field.Type)
}
