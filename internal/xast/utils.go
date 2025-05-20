package xast

import (
	"fmt"
	"go/ast"
	"strings"
)

func CollectImportsFromType(getPkgPathBySelector func(expr *ast.SelectorExpr) string, expr ast.Expr) (out []string) {
	return collectImportsFromType(getPkgPathBySelector, expr)
}

// collectImportsFromType recursively collects imports from a type expression
func collectImportsFromType(getPkgPathBySelector func(expr *ast.SelectorExpr) string, expr ast.Expr) (out []string) {
	switch t := expr.(type) {
	case *ast.SelectorExpr:
		if pkgPath := getPkgPathBySelector(t); pkgPath != "" {
			out = append(out, pkgPath)
		}
		collectImportsFromType(getPkgPathBySelector, t.X)
	case *ast.StarExpr:
		collectImportsFromType(getPkgPathBySelector, t.X)
	case *ast.ArrayType:
		collectImportsFromType(getPkgPathBySelector, t.Elt)
		if t.Len != nil {
			collectImportsFromType(getPkgPathBySelector, t.Len)
		}
	case *ast.MapType:
		collectImportsFromType(getPkgPathBySelector, t.Key)
		collectImportsFromType(getPkgPathBySelector, t.Value)
	case *ast.IndexExpr:
		out = append(out, collectImportsFromType(getPkgPathBySelector, t.X)...)
		out = append(out, collectImportsFromType(getPkgPathBySelector, t.Index)...)
	case *ast.IndexListExpr:
		out = append(out, collectImportsFromType(getPkgPathBySelector, t.X)...)
		for _, expr := range t.Indices {
			out = append(out, collectImportsFromType(getPkgPathBySelector, expr)...)
		}
	case *ast.FuncType:
		if t.Params != nil {
			for _, param := range t.Params.List {
				collectImportsFromType(getPkgPathBySelector, param.Type)
			}
		}
		if t.Results != nil {
			for _, result := range t.Results.List {
				collectImportsFromType(getPkgPathBySelector, result.Type)
			}
		}
	}
	return out
}

func GetFieldType(expr ast.Expr, getAlias func(*ast.SelectorExpr) string) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + GetFieldType(t.X, getAlias)
	case *ast.SelectorExpr:
		x := GetFieldType(t.X, getAlias)
		if getAlias != nil {
			if alias := getAlias(t); alias != "" {
				x = alias
			}
		}
		return fmt.Sprintf("%s.%s", x, t.Sel.Name)
	case *ast.ArrayType:
		if t.Len == nil {
			// Slice type
			return "[]" + GetFieldType(t.Elt, getAlias)
		}
		// Array type
		return fmt.Sprintf("[%s]%s", GetFieldType(t.Len, getAlias), GetFieldType(t.Elt, getAlias))
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", GetFieldType(t.Key, getAlias), GetFieldType(t.Value, getAlias))
	case *ast.InterfaceType:
		return "any"
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + GetFieldType(t.Value, getAlias)
		case ast.RECV:
			return "<-chan " + GetFieldType(t.Value, getAlias)
		default:
			return "chan " + GetFieldType(t.Value, getAlias)
		}
	case *ast.BasicLit:
		// Used for array length literals
		return t.Value
	case *ast.FuncType:
		return GetFuncType(t, getAlias)
	case *ast.StructType:
		return "struct{}"
	case *ast.IndexExpr:
		// 处理泛型类型，如 SomeType[T]
		baseType := GetFieldType(t.X, getAlias)
		indexType := GetFieldType(t.Index, getAlias)
		return fmt.Sprintf("%s[%s]", baseType, indexType)
	case *ast.IndexListExpr:
		// 处理多参数泛型类型，如 SomeType[T1, T2]
		baseType := GetFieldType(t.X, getAlias)
		var typeParams []string
		for _, expr := range t.Indices {
			typeParams = append(typeParams, GetFieldType(expr, getAlias))
		}
		return fmt.Sprintf("%s[%s]", baseType, strings.Join(typeParams, ", "))
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func GetFuncType(t *ast.FuncType, aliasMap func(*ast.SelectorExpr) string) string {
	var params, results []string

	// Handle parameters
	if t.Params != nil {
		for _, param := range t.Params.List {
			paramType := GetFieldType(param.Type, aliasMap)
			if len(param.Names) == 0 {
				params = append(params, paramType)
			} else {
				for range param.Names {
					params = append(params, paramType)
				}
			}
		}
	}

	// Handle return values
	if t.Results != nil {
		for _, result := range t.Results.List {
			resultType := GetFieldType(result.Type, aliasMap)
			if len(result.Names) == 0 {
				results = append(results, resultType)
			} else {
				for range result.Names {
					results = append(results, resultType)
				}
			}
		}
	}

	// Build function type string
	funcStr := "func(" + strings.Join(params, ", ") + ")"
	if len(results) > 0 {
		if len(results) == 1 {
			funcStr += " " + results[0]
		} else {
			funcStr += " (" + strings.Join(results, ", ") + ")"
		}
	}

	return funcStr
}
