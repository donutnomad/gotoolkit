package automap

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestDebugParser(t *testing.T) {
	fset := token.NewFileSet()

	// 解析当前目录
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.AllErrors)
	if err != nil {
		t.Fatalf("解析目录失败: %v", err)
	}

	fmt.Printf("查找MapAToB函数:\n")

	for pkgName, pkg := range pkgs {
		fmt.Printf("检查包: %s\n", pkgName)

		for filename, file := range pkg.Files {
			fmt.Printf("  文件: %s\n", filename)

			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					fmt.Printf("    函数: %s\n", fn.Name.Name)
					if fn.Name.Name == "MapAToB" {
						fmt.Printf("    找到MapAToB函数!\n")

						// 打印函数参数和返回值
						if fn.Type.Params != nil && len(fn.Type.Params.List) > 0 {
							param := fn.Type.Params.List[0]
							fmt.Printf("    参数: %T\n", param.Type)
						}

						if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
							result := fn.Type.Results.List[0]
							fmt.Printf("    返回值: %T\n", result.Type)
						}
					}
				}
			}
		}
	}
}
