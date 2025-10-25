package automap

import (
	"go/token"
	"testing"
)

func TestAutoMap_Parse(t *testing.T) {
	tests := []struct {
		name        string
		funcName    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "解析MapAToB函数",
			funcName:    "MapAToB",
			expectError: false,
		},
		{
			name:        "解析不存在的函数",
			funcName:    "NonExistentFunc",
			expectError: true,
			errorMsg:    "未找到函数",
		},
		{
			name:        "空函数名",
			funcName:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			automap := New()
			result, err := automap.Parse(tt.funcName)

			if tt.expectError {
				if err == nil {
					t.Errorf("期望有错误，但没有错误发生")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("错误消息不匹配，期望包含: %s, 实际: %s", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("解析失败: %v", err)
				return
			}

			// 验证结果
			if result.FuncSignature.FuncName != "MapAToB" {
				t.Errorf("函数名不匹配，期望: MapAToB, 实际: %s", result.FuncSignature.FuncName)
			}

			if result.AType.Name != "A" {
				t.Errorf("A类型名不匹配，期望: A, 实际: %s", result.AType.Name)
			}

			if result.BType.Name != "B" {
				t.Errorf("B类型名不匹配，期望: B, 实际: %s", result.BType.Name)
			}

			if !result.HasExportPatch {
				t.Error("期望有ExportPatch方法，但没有找到")
			}

			if result.GeneratedCode == "" {
				t.Error("生成的代码为空")
			}

			// 打印生成的代码用于调试
			t.Logf("生成的代码:\n%s", result.GeneratedCode)
		})
	}
}

func TestAutoMap_ParseAndGenerate(t *testing.T) {
	automap := New()
	code, err := automap.ParseAndGenerate("MapAToB")

	if err != nil {
		t.Fatalf("解析并生成代码失败: %v", err)
	}

	if code == "" {
		t.Fatal("生成的代码为空")
	}

	// 检查代码是否包含关键部分
	expectedParts := []string{
		"func Do(input *A) map[string]any {",
		"b := MapAToB(input)",
		"fields := input.ExportPatch()",
		"var ret = make(map[string]any)",
		"return ret",
	}

	for _, part := range expectedParts {
		if !contains(code, part) {
			t.Errorf("生成的代码缺少关键部分: %s", part)
		}
	}

	// 打印完整代码用于调试
	t.Logf("完整生成的代码:\n%s", code)
}

func TestAutoMap_ValidateFunction(t *testing.T) {
	tests := []struct {
		name        string
		funcName    string
		expectError bool
	}{
		{
			name:        "验证有效的MapAToB函数",
			funcName:    "MapAToB",
			expectError: false,
		},
		{
			name:        "验证不存在的函数",
			funcName:    "NonExistentFunc",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			automap := New()
			err := automap.ValidateFunction(tt.funcName)

			if tt.expectError && err == nil {
				t.Errorf("期望验证失败，但验证通过了")
			}

			if !tt.expectError && err != nil {
				t.Errorf("期望验证通过，但验证失败: %v", err)
			}
		})
	}
}

func TestAutoMap_GetFunctionSignature(t *testing.T) {
	automap := New()
	sig, err := automap.GetFunctionSignature("MapAToB")

	if err != nil {
		t.Fatalf("获取函数签名失败: %v", err)
	}

	if sig.FuncName != "MapAToB" {
		t.Errorf("函数名不匹配，期望: MapAToB, 实际: %s", sig.FuncName)
	}

	if sig.InputType.Name != "A" {
		t.Errorf("输入类型不匹配，期望: A, 实际: %s", sig.InputType.Name)
	}

	if sig.OutputType.Name != "B" {
		t.Errorf("输出类型不匹配，期望: B, 实际: %s", sig.OutputType.Name)
	}

	if !sig.InputType.IsPointer {
		t.Error("输入类型应该是指针类型")
	}

	if !sig.OutputType.IsPointer {
		t.Error("输出类型应该是指针类型")
	}
}

func TestAutoMap_GetTypeInfo(t *testing.T) {
	automap := New()

	// 测试获取A类型信息
	aType, err := automap.GetTypeInfo("A")
	if err != nil {
		t.Fatalf("获取A类型信息失败: %v", err)
	}

	if aType.Name != "A" {
		t.Errorf("类型名不匹配，期望: A, 实际: %s", aType.Name)
	}

	if len(aType.Fields) == 0 {
		t.Error("A类型应该有字段")
	}

	// 检查是否有ExportPatch方法
	hasExportPatch := false
	for _, method := range aType.Methods {
		if method.Name == "ExportPatch" {
			hasExportPatch = true
			break
		}
	}

	if !hasExportPatch {
		t.Error("A类型应该有ExportPatch方法")
	}
}

func TestMappingAnalyzer(t *testing.T) {
	// 这个测试需要实际的AST节点，比较复杂
	// 简单起见，我们测试创建MappingAnalyzer
	fset := token.NewFileSet()
	analyzer := NewMappingAnalyzer(fset)

	if analyzer == nil {
		t.Fatal("创建MappingAnalyzer失败")
	}

	// 检查初始状态
	if len(analyzer.mappingRel) != 0 {
		t.Errorf("初始映射关系应该为空，实际有: %d", len(analyzer.mappingRel))
	}

	if len(analyzer.fieldMapping.OneToOne) != 0 {
		t.Errorf("初始一对一映射应该为空，实际有: %d", len(analyzer.fieldMapping.OneToOne))
	}
}

func TestCodeGenerator(t *testing.T) {
	generator := NewCodeGenerator(nil)

	if generator == nil {
		t.Fatal("创建CodeGenerator失败")
	}

	// 创建一个简单的ParseResult用于测试
	result := &ParseResult{
		FuncSignature: FuncSignature{
			FuncName: "TestFunc",
		},
		AType: TypeInfo{
			Name: "A",
		},
		BType: TypeInfo{
			Name: "B",
		},
		FieldMapping: FieldMapping{
			OneToOne: map[string]string{
				"Field1": "FieldA",
				"Field2": "FieldB",
			},
			OneToMany: map[string][]string{
				"Field3": {"FieldC", "FieldD"},
			},
		},
		HasExportPatch: true,
	}

	code := generator.Generate(result)

	if code == "" {
		t.Error("生成的代码为空")
	}

	// 检查生成的代码是否包含预期的部分
	expectedParts := []string{
		"func Do(input *A) map[string]any",
		"b := TestFunc(input)",
		"fields := input.ExportPatch()",
		"var ret = make(map[string]any)",
		"return ret",
	}

	for _, part := range expectedParts {
		if !contains(code, part) {
			t.Errorf("生成的代码缺少关键部分: %s", part)
		}
	}

	t.Logf("生成的测试代码:\n%s", code)
}

func TestTypeResolver(t *testing.T) {
	resolver := NewTypeResolver()

	if resolver == nil {
		t.Fatal("创建TypeResolver失败")
	}

	// 检查初始状态
	if len(resolver.cache) != 0 {
		t.Errorf("初始缓存应该为空，实际有: %d", len(resolver.cache))
	}

	if len(resolver.importMap) != 0 {
		t.Errorf("初始import映射应该为空，实际有: %d", len(resolver.importMap))
	}
}

func TestValidator(t *testing.T) {
	resolver := NewTypeResolver()
	validator := NewValidator(resolver)

	if validator == nil {
		t.Fatal("创建Validator失败")
	}

	if validator.typeResolver != resolver {
		t.Error("typeResolver设置不正确")
	}
}

// 辅助函数：检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// 基准测试
func BenchmarkAutoMap_Parse(b *testing.B) {
	automap := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := automap.Parse("MapAToB")
		if err != nil {
			b.Fatalf("解析失败: %v", err)
		}
	}
}

func BenchmarkAutoMap_ParseAndGenerate(b *testing.B) {
	automap := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := automap.ParseAndGenerate("MapAToB")
		if err != nil {
			b.Fatalf("解析并生成代码失败: %v", err)
		}
	}
}

// 示例测试
func ExampleAutoMap_Parse() {
	automap := New()
	result, err := automap.Parse("MapAToB")
	if err != nil {
		panic(err)
	}

	println("函数名:", result.FuncSignature.FuncName)
	println("A类型:", result.AType.Name)
	println("B类型:", result.BType.Name)
	if result.HasExportPatch {
		println("是否有ExportPatch方法: 是")
	} else {
		println("是否有ExportPatch方法: 否")
	}
	println("生成的代码长度:", len(result.GeneratedCode))
}

func ExampleAutoMap_ParseAndGenerate() {
	code, err := ParseAndGenerate("MapAToB")
	if err != nil {
		panic(err)
	}

	println("生成的代码:")
	println(code)
}
