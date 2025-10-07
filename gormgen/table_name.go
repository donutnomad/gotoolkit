package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/donutnomad/gotoolkit/internal/gormparse"
)

// inferTableName 推导表名
func inferTableName(filename, structName string) (string, error) {
	// 首先尝试查找TableName方法
	tableName, err := extractTableNameFromMethod(filename, structName)
	if err == nil && tableName != "" {
		return tableName, nil
	}

	// 如果没有TableName方法,使用默认规则: 结构体名的复数形式 + 蛇形命名
	return gormparse.ToSnakeCase(structName) + "s", nil
}

// extractTableNameFromMethod 从TableName方法中提取表名
func extractTableNameFromMethod(filename, structName string) (string, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return "", err
	}

	var tableName string
	ast.Inspect(node, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// 检查是否是TableName方法
			if funcDecl.Name.Name == "TableName" && funcDecl.Recv != nil {
				// 检查接收者类型
				if len(funcDecl.Recv.List) > 0 {
					recvType := ""
					switch t := funcDecl.Recv.List[0].Type.(type) {
					case *ast.StarExpr:
						if ident, ok := t.X.(*ast.Ident); ok {
							recvType = ident.Name
						}
					case *ast.Ident:
						recvType = t.Name
					}

					if recvType == structName {
						// 提取返回值
						if funcDecl.Body != nil {
							for _, stmt := range funcDecl.Body.List {
								if retStmt, ok := stmt.(*ast.ReturnStmt); ok {
									if len(retStmt.Results) > 0 {
										if lit, ok := retStmt.Results[0].(*ast.BasicLit); ok {
											tableName = strings.Trim(lit.Value, `"`)
											return false
										}
									}
								}
							}
						}
					}
				}
			}
		}
		return true
	})

	return tableName, nil
}
