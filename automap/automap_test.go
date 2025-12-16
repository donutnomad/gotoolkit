package automap_test

import (
	"testing"

	"github.com/donutnomad/gotoolkit/automap"
	"github.com/donutnomad/gotoolkit/automap/testdata"
)

// TestParseSimpleOneToOne 测试一对一映射
func TestParseSimpleOneToOne(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "SimpleUserPO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := testdata.ExpectedSimpleUserMapping
	assertParseResult(t, result, expected)
}

// TestParseEmbedded 测试嵌入字段映射
func TestParseEmbedded(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "EmbeddedUserPO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := testdata.ExpectedEmbeddedUserMapping
	assertParseResult(t, result, expected)
}

// TestParseManyToOneJSON 测试多对一（JSON）映射
func TestParseManyToOneJSON(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "ProfilePO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := testdata.ExpectedProfileMapping
	assertParseResult(t, result, expected)
}

// TestParseOneToMany 测试一对多映射
func TestParseOneToMany(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "CompanyPO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := testdata.ExpectedCompanyMapping
	assertParseResult(t, result, expected)
}

// TestParseEmbeddedWithPrefix 测试带前缀的嵌入映射
func TestParseEmbeddedWithPrefix(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "AuditPO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := testdata.ExpectedAuditMapping
	assertParseResult(t, result, expected)
}

// TestParseNestedJSON 测试复杂嵌套JSON映射
func TestParseNestedJSON(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "ArticlePO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := testdata.ExpectedArticleMapping
	assertParseResult(t, result, expected)
}

// TestParseTypeConversion 测试类型转换映射
func TestParseTypeConversion(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "TimestampPO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := testdata.ExpectedTimestampMapping
	assertParseResult(t, result, expected)
}

// TestParseMixed 测试混合映射
func TestParseMixed(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "AccountPO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := testdata.ExpectedAccountMapping
	assertParseResult(t, result, expected)
}

// TestParseLocalVariable 测试局部变量映射
func TestParseLocalVariable(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "ProductPO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := testdata.ExpectedProductMapping
	assertParseResult(t, result, expected)
}

// TestParseLocalVariableWithJSON 测试局部变量 + JSON 映射
func TestParseLocalVariableWithJSON(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "OrderPO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := testdata.ExpectedOrderMapping
	assertParseResult(t, result, expected)
}

// TestParseMethodCall 测试方法调用映射
func TestParseMethodCall(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "CustomerPO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := testdata.ExpectedCustomerMapping
	assertParseResult(t, result, expected)
}

// TestParseMethodCallWithLocalVar 测试方法调用 + 局部变量映射
func TestParseMethodCallWithLocalVar(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "ShippingPO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := testdata.ExpectedShippingMapping
	assertParseResult(t, result, expected)
}

// TestParseJSONSliceLoMap 测试 JSONSlice + lo.Map 映射
func TestParseJSONSliceLoMap(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "JSONSlicePO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := testdata.ExpectedJSONSliceMapping
	assertParseResult(t, result, expected)
}

// TestParseJSONSliceLoMapMethodCall 测试 JSONSlice + lo.Map + 方法调用映射
func TestParseJSONSliceLoMapMethodCall(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "JSONSliceMethodPO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	expected := testdata.ExpectedJSONSliceMethodMapping
	assertParseResult(t, result, expected)
}

// assertParseResult 验证解析结果
func assertParseResult(t *testing.T, result *automap.ParseResult2, expected testdata.ParseResult) {
	t.Helper()

	// 验证基本信息
	if result.FuncName != expected.FuncName {
		t.Errorf("FuncName mismatch: got %s, want %s", result.FuncName, expected.FuncName)
	}
	if result.ReceiverType != expected.ReceiverType {
		t.Errorf("ReceiverType mismatch: got %s, want %s", result.ReceiverType, expected.ReceiverType)
	}
	if result.SourceType != expected.SourceType {
		t.Errorf("SourceType mismatch: got %s, want %s", result.SourceType, expected.SourceType)
	}
	if result.TargetType != expected.TargetType {
		t.Errorf("TargetType mismatch: got %s, want %s", result.TargetType, expected.TargetType)
	}

	// 验证映射组
	if len(result.Groups) != len(expected.Groups) {
		t.Errorf("Groups count mismatch: got %d, want %d", len(result.Groups), len(expected.Groups))
		return
	}

	for i, group := range result.Groups {
		expectedGroup := expected.Groups[i]
		assertMappingGroup(t, i, group, expectedGroup)
	}
}

// assertMappingGroup 验证映射组
func assertMappingGroup(t *testing.T, idx int, result automap.MappingGroup, expected testdata.MappingGroup) {
	t.Helper()

	if string(result.Type) != string(expected.Type) {
		t.Errorf("Group[%d].Type mismatch: got %s, want %s", idx, result.Type, expected.Type)
	}
	if result.SourceField != expected.SourceField {
		t.Errorf("Group[%d].SourceField mismatch: got %s, want %s", idx, result.SourceField, expected.SourceField)
	}
	if result.TargetField != expected.TargetField {
		t.Errorf("Group[%d].TargetField mismatch: got %s, want %s", idx, result.TargetField, expected.TargetField)
	}
	if result.MethodName != expected.MethodName {
		t.Errorf("Group[%d].MethodName mismatch: got %s, want %s", idx, result.MethodName, expected.MethodName)
	}

	if len(result.Mappings) != len(expected.Mappings) {
		t.Errorf("Group[%d].Mappings count mismatch: got %d, want %d", idx, len(result.Mappings), len(expected.Mappings))
		return
	}

	for i, mapping := range result.Mappings {
		expectedMapping := expected.Mappings[i]
		assertFieldMapping(t, idx, i, mapping, expectedMapping)
	}
}

// assertFieldMapping 验证字段映射
func assertFieldMapping(t *testing.T, groupIdx, mappingIdx int, result automap.FieldMapping2, expected testdata.FieldMapping) {
	t.Helper()

	prefix := func() string {
		return "Group[%d].Mapping[%d]"
	}

	if result.SourcePath != expected.SourcePath {
		t.Errorf(prefix()+".SourcePath mismatch: got %s, want %s", groupIdx, mappingIdx, result.SourcePath, expected.SourcePath)
	}
	if result.TargetPath != expected.TargetPath {
		t.Errorf(prefix()+".TargetPath mismatch: got %s, want %s", groupIdx, mappingIdx, result.TargetPath, expected.TargetPath)
	}
	if result.ColumnName != expected.ColumnName {
		t.Errorf(prefix()+".ColumnName mismatch: got %s, want %s", groupIdx, mappingIdx, result.ColumnName, expected.ColumnName)
	}
	if result.ConvertExpr != expected.ConvertExpr {
		t.Errorf(prefix()+".ConvertExpr mismatch: got %s, want %s", groupIdx, mappingIdx, result.ConvertExpr, expected.ConvertExpr)
	}
	if result.JSONPath != expected.JSONPath {
		t.Errorf(prefix()+".JSONPath mismatch: got %s, want %s", groupIdx, mappingIdx, result.JSONPath, expected.JSONPath)
	}
}
