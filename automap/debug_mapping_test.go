package automap

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestDebugMapping(t *testing.T) {
	fset := token.NewFileSet()

	// 解析当前目录
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.AllErrors)
	if err != nil {
		t.Fatalf("解析目录失败: %v", err)
	}

	// 查找MapAToB函数
	var mapFunc *ast.FuncDecl
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "MapAToB" {
					mapFunc = fn
					break
				}
			}
			if mapFunc != nil {
				break
			}
		}
		if mapFunc != nil {
			break
		}
	}

	if mapFunc == nil {
		t.Fatal("未找到MapAToB函数")
	}

	fmt.Printf("找到MapAToB函数，参数数量: %d\n", len(mapFunc.Type.Params.List))
	fmt.Printf("返回值数量: %d\n", len(mapFunc.Type.Results.List))

	// 分析函数体
	if mapFunc.Body == nil {
		t.Fatal("函数体为空")
	}

	fmt.Printf("函数体语句数量: %d\n", len(mapFunc.Body.List))

	// 查找return语句
	for i, stmt := range mapFunc.Body.List {
		fmt.Printf("语句 %d: %T\n", i, stmt)
		if ret, ok := stmt.(*ast.ReturnStmt); ok {
			fmt.Printf("找到return语句，返回值数量: %d\n", len(ret.Results))
			for j, result := range ret.Results {
				fmt.Printf("  返回值 %d: %T\n", j, result)
				if compLit, ok := result.(*ast.CompositeLit); ok {
					fmt.Printf("    结构体字面量类型: %T\n", compLit.Type)
					fmt.Printf("    字段数量: %d\n", len(compLit.Elts))
					for k, elt := range compLit.Elts {
						fmt.Printf("      字段 %d: %T\n", k, elt)
						if kv, ok := elt.(*ast.KeyValueExpr); ok {
							fmt.Printf("        Key: %T, Value: %T\n", kv.Key, kv.Value)
							if keyIdent, ok := kv.Key.(*ast.Ident); ok {
								fmt.Printf("        Key名称: %s\n", keyIdent.Name)
							}
						}
					}
				}
			}
		}
	}
}
