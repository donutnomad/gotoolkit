package automap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestIsExported(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"uppercase first letter", "Name", true},
		{"lowercase first letter", "name", false},
		{"empty string", "", false},
		{"single uppercase", "A", true},
		{"single lowercase", "a", false},
		{"number start", "123", false},
		{"underscore start", "_name", false},
		{"mixed case exported", "UserID", true},
		{"mixed case unexported", "userID", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isExported(tt.input)
			if result != tt.expected {
				t.Errorf("isExported(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Name", "name"},
		{"UserID", "user_id"},
		{"CreatedAt", "created_at"},
		{"XMLParser", "xml_parser"},
		{"ID", "id"},
		{"SimpleTest", "simple_test"},
		{"", ""},
		{"A", "a"},
		{"ABC", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toSnakeCase(tt.input)
			if result != tt.expected {
				t.Errorf("toSnakeCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGoFileIterator(t *testing.T) {
	// 使用 testdata 目录进行测试
	testDir := "testdata"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("testdata directory not found")
	}

	// 获取 testdata 目录下的一个 Go 文件
	entries, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatalf("Failed to read testdata: %v", err)
	}

	var goFile string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".go" && filepath.Ext(entry.Name()) != "_test.go" {
			goFile = filepath.Join(testDir, entry.Name())
			break
		}
	}

	if goFile == "" {
		t.Skip("No Go file found in testdata")
	}

	iterator := NewGoFileIterator(goFile)

	// 测试 Iterate（应跳过当前文件）
	var filesVisited []string
	err = iterator.Iterate(func(filePath string) bool {
		filesVisited = append(filesVisited, filepath.Base(filePath))
		return true
	})
	if err != nil {
		t.Errorf("Iterate failed: %v", err)
	}

	// 确保当前文件被跳过
	currentFile := filepath.Base(goFile)
	for _, f := range filesVisited {
		if f == currentFile {
			t.Errorf("Iterate should skip current file %s", currentFile)
		}
	}

	// 测试 IterateIncludeCurrent（应包含所有文件）
	var allFilesVisited []string
	err = iterator.IterateIncludeCurrent(func(filePath string) bool {
		allFilesVisited = append(allFilesVisited, filepath.Base(filePath))
		return true
	})
	if err != nil {
		t.Errorf("IterateIncludeCurrent failed: %v", err)
	}

	// IterateIncludeCurrent 应该比 Iterate 多包含文件
	if len(allFilesVisited) < len(filesVisited) {
		t.Errorf("IterateIncludeCurrent should visit more files than Iterate")
	}
}

func TestGoFileIteratorEarlyStop(t *testing.T) {
	testDir := "testdata"
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("testdata directory not found")
	}

	entries, err := os.ReadDir(testDir)
	if err != nil {
		t.Fatalf("Failed to read testdata: %v", err)
	}

	var goFile string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".go" {
			goFile = filepath.Join(testDir, entry.Name())
			break
		}
	}

	if goFile == "" {
		t.Skip("No Go file found in testdata")
	}

	iterator := NewGoFileIterator(goFile)

	// 测试提前停止
	count := 0
	_ = iterator.IterateIncludeCurrent(func(filePath string) bool {
		count++
		return count < 2 // 在第二个文件后停止
	})

	if count > 2 {
		t.Errorf("Iterator should have stopped early, but visited %d files", count)
	}
}

func TestExtractTypeName(t *testing.T) {
	src := `package test
type MyStruct struct {}
var x *MyStruct
var y pkg.ExternalType
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// 查找变量声明并测试类型提取
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			name := valueSpec.Names[0].Name
			typeName := extractTypeName(valueSpec.Type)

			switch name {
			case "x":
				if typeName != "MyStruct" {
					t.Errorf("extractTypeName for x: got %q, want %q", typeName, "MyStruct")
				}
			case "y":
				if typeName != "ExternalType" {
					t.Errorf("extractTypeName for y: got %q, want %q", typeName, "ExternalType")
				}
			}
		}
	}
}

func TestExtractTypeNameWithPackage(t *testing.T) {
	src := `package test
var a MyStruct
var b *MyStruct
var c pkg.ExternalType
var d *pkg.ExternalType
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	tests := map[string]struct {
		expectedType string
		expectedPkg  string
	}{
		"a": {"MyStruct", ""},
		"b": {"MyStruct", ""},
		"c": {"ExternalType", "pkg"},
		"d": {"ExternalType", "pkg"},
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			name := valueSpec.Names[0].Name
			expected, exists := tests[name]
			if !exists {
				continue
			}

			typeName, pkgName := extractTypeNameWithPackage(valueSpec.Type)
			if typeName != expected.expectedType {
				t.Errorf("extractTypeNameWithPackage for %s: typeName = %q, want %q", name, typeName, expected.expectedType)
			}
			if pkgName != expected.expectedPkg {
				t.Errorf("extractTypeNameWithPackage for %s: pkgName = %q, want %q", name, pkgName, expected.expectedPkg)
			}
		}
	}
}

func TestGetExprString(t *testing.T) {
	src := `package test
var a MyStruct
var b *MyStruct
var c pkg.Type
var d []int
var e map[string]int
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	tests := map[string]string{
		"a": "MyStruct",
		"b": "*MyStruct",
		"c": "pkg.Type",
		"d": "[]int",
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			name := valueSpec.Names[0].Name
			expected, exists := tests[name]
			if !exists {
				continue
			}

			result := getExprString(valueSpec.Type)
			if result != expected {
				t.Errorf("getExprString for %s: got %q, want %q", name, result, expected)
			}
		}
	}
}

func TestInferFieldNameFromMethod(t *testing.T) {
	tests := []struct {
		methodName string
		expected   string
	}{
		{"GetName", "Name"},
		{"GetUserID", "UserID"},
		{"Get", "Get"}, // Too short, return as-is
		{"SetName", "SetName"},
		{"Name", "Name"},
		{"GetExchangeRules", "ExchangeRules"},
	}

	for _, tt := range tests {
		t.Run(tt.methodName, func(t *testing.T) {
			result := inferFieldNameFromMethod(tt.methodName)
			if result != tt.expected {
				t.Errorf("inferFieldNameFromMethod(%q) = %q, want %q", tt.methodName, result, tt.expected)
			}
		})
	}
}
