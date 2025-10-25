package automap

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"runtime"
	"testing"
)

func TestDebugParse(t *testing.T) {
	fset := token.NewFileSet()

	// 解析当前目录
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.AllErrors)
	if err != nil {
		t.Fatalf("解析目录失败: %v", err)
	}

	fmt.Printf("找到的包:\n")
	for pkgName, pkg := range pkgs {
		fmt.Printf("- %s\n", pkgName)

		fmt.Printf("  文件:\n")
		for filename := range pkg.Files {
			fmt.Printf("    - %s\n", filename)
		}

		fmt.Printf("  类型定义:\n")
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				if genDecl, ok := decl.(*ast.GenDecl); ok {
					for _, spec := range genDecl.Specs {
						if typeSpec, ok := spec.(*ast.TypeSpec); ok {
							fmt.Printf("    - %s\n", typeSpec.Name.Name)
						}
					}
				}
			}
		}
	}
}

func TestDebugCallerFile(t *testing.T) {
	// 测试getCallerFile函数
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("无法获取调用者文件")
	}
	fmt.Printf("调用者文件: %s\n", file)

	// 测试解析器中的getCallerFile
	automap := New()
	callerFile := automap.getCallerFile()
	fmt.Printf("解析器获取的调用者文件: %s\n", callerFile)
}

func TestDebugFindType(t *testing.T) {
	automap := New()
	resolver := automap.typeResolver

	// 测试查找A类型
	currentFile := "mod.go" // 直接指定mod.go
	fmt.Printf("尝试在文件中查找类型: %s\n", currentFile)

	typeSpec, filePath, err := resolver.findTypeDefinition(".", "A")
	if err != nil {
		t.Fatalf("查找类型A失败: %v", err)
	}

	fmt.Printf("找到类型A: %s, 文件路径: %s\n", typeSpec.Name.Name, filePath)
}
