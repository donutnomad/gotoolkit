package automap

import (
	"go/ast"
	"os"
	"path/filepath"
	"strings"

	"github.com/donutnomad/gotoolkit/internal/gormparse"
)

// isExported 检查字段名是否为导出字段
func isExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	return name[0] >= 'A' && name[0] <= 'Z'
}

// toSnakeCase 驼峰转蛇形（使用 gormparse 的实现，正确处理 ID 等缩写）
func toSnakeCase(s string) string {
	return gormparse.ToSnakeCase(s)
}

// GoFileIterator 遍历 Go 文件的迭代器
type GoFileIterator struct {
	baseDir     string
	skipCurrent string // 需要跳过的文件（通常是当前文件）
}

// NewGoFileIterator 创建新的 Go 文件迭代器
func NewGoFileIterator(filePath string) *GoFileIterator {
	return &GoFileIterator{
		baseDir:     filepath.Dir(filePath),
		skipCurrent: filePath,
	}
}

// Iterate 遍历目录中的 Go 文件
// fn 返回 false 时停止遍历
func (it *GoFileIterator) Iterate(fn func(filePath string) bool) error {
	entries, err := os.ReadDir(it.baseDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		fullPath := filepath.Join(it.baseDir, name)
		if fullPath == it.skipCurrent {
			continue
		}

		if !fn(fullPath) {
			return nil
		}
	}
	return nil
}

// IterateIncludeCurrent 遍历目录中的所有 Go 文件（包括当前文件）
func (it *GoFileIterator) IterateIncludeCurrent(fn func(filePath string) bool) error {
	entries, err := os.ReadDir(it.baseDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		fullPath := filepath.Join(it.baseDir, name)
		if !fn(fullPath) {
			return nil
		}
	}
	return nil
}

// extractTypeName 提取类型名（去掉指针）
func extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return extractTypeName(t.X)
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return ""
}

// extractTypeNameWithPackage 提取类型名和包名
// 返回: (typeName, packageName)
// 例如: *domain.ListingDomain -> ("ListingDomain", "domain")
// 例如: *ListingDomain -> ("ListingDomain", "")
func extractTypeNameWithPackage(expr ast.Expr) (typeName, packageName string) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, ""
	case *ast.StarExpr:
		return extractTypeNameWithPackage(t.X)
	case *ast.SelectorExpr:
		// pkg.TypeName
		if pkgIdent, ok := t.X.(*ast.Ident); ok {
			return t.Sel.Name, pkgIdent.Name
		}
		return t.Sel.Name, ""
	}
	return "", ""
}

// getExprString 获取表达式的字符串表示
func getExprString(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return getExprString(e.X) + "." + e.Sel.Name
	case *ast.StarExpr:
		return "*" + getExprString(e.X)
	case *ast.IndexExpr:
		return getExprString(e.X) + "[" + getExprString(e.Index) + "]"
	case *ast.ArrayType:
		return "[]" + getExprString(e.Elt)
	}
	return ""
}
