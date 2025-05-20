package xast

import (
	"go/ast"
	"path/filepath"
	"strings"
)

// StructType 当前结构体和结构体所在文件的imports
type StructType struct {
	*ast.StructType
	Imports []*ast.ImportSpec
}

func (s *StructType) CollectImports(expr ast.Expr) (out []string) {
	return collectImportsFromType(s.GetPkgPathBySelector, expr)
}

func (s *StructType) GetPkgPathBySelector(expr *ast.SelectorExpr) string {
	ident, ok := expr.X.(*ast.Ident)
	if !ok {
		return ""
	}
	// 寻找字段对应的import路径，例如mo.Option[bool], 那么此处就是从imports中寻找到到mo
	for _, item := range s.Imports {
		importPath := strings.Trim(item.Path.Value, `"`)
		n := filepath.Base(importPath)
		if item.Name != nil {
			n = item.Name.Name
		}
		if n == ident.Name {
			return importPath
		}
	}
	return ""
}
