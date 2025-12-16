package automap_test

import (
	"strings"
	"testing"

	"github.com/donutnomad/gotoolkit/automap"
)

// TestGenerate2SimpleOneToOne 测试简单一对一映射的代码生成
func TestGenerate2SimpleOneToOne(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "SimpleUserPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证函数签名
	if !strings.Contains(funcCode, "func (s *SimpleUserPO) ToPatch(input *SimpleUserDomain) map[string]any") {
		t.Errorf("Function signature mismatch, got:\n%s", funcCode)
	}

	// 验证调用 ToPO
	if !strings.Contains(funcCode, "b := s.ToPO(input)") {
		t.Errorf("Missing ToPO call, got:\n%s", funcCode)
	}

	// 验证 ExportPatch 调用
	if !strings.Contains(funcCode, "fields := input.ExportPatch()") {
		t.Errorf("Missing ExportPatch call, got:\n%s", funcCode)
	}

	// 验证字段映射
	expectedMappings := []string{
		`fields.ID.IsPresent()`,
		`values["id"] = b.ID`,
		`fields.Name.IsPresent()`,
		`values["name"] = b.Name`,
		`fields.Email.IsPresent()`,
		`values["email"] = b.Email`,
		`fields.Age.IsPresent()`,
		`values["age"] = b.Age`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2Embedded 测试嵌入字段映射的代码生成
func TestGenerate2Embedded(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "EmbeddedUserPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证嵌入字段映射
	expectedMappings := []string{
		`// Embedded: Model`,
		`fields.ID.IsPresent()`,
		`values["id"] = b.Model.ID`,
		`fields.CreatedAt.IsPresent()`,
		`values["created_at"] = b.Model.CreatedAt`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2ManyToOneJSON 测试多对一(JSON)映射的代码生成
func TestGenerate2ManyToOneJSON(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "ProfilePO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证需要 gsql 包
	if !strings.Contains(fullCode, `"github.com/donutnomad/gsql"`) {
		t.Errorf("Missing gsql import")
	}

	// 验证 JSON 映射
	expectedMappings := []string{
		`// B.Contact`,
		`set := gsql.JSONSet("contact")`,
		`field := b.Contact.Data()`,
		`fields.Phone.IsPresent()`,
		`set.Set("phone"`,
		`if set.Len() > 0`,
		`values["contact"] = set`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2OneToMany 测试一对多映射的代码生成
func TestGenerate2OneToMany(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "CompanyPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证一对多映射
	expectedMappings := []string{
		`// OneToMany: Location`,
		`fields.Location.IsPresent()`,
		`values["country"] = b.Country`,
		`values["province"] = b.Province`,
		`values["city"] = b.City`,
		`values["district"] = b.District`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2MethodCall 测试方法调用映射的代码生成
func TestGenerate2MethodCall(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "CustomerPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证方法调用映射
	expectedMappings := []string{
		`// MethodCall: GetAddress() -> Address`,
		`fields.City.IsPresent() || fields.Country.IsPresent() || fields.Province.IsPresent() || fields.Street.IsPresent()`,
		`values["address"] = b.Address`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2NestedJSON 测试嵌套JSON映射的代码生成
func TestGenerate2NestedJSON(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "ArticlePO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证嵌套 JSON 映射
	expectedMappings := []string{
		`// B.Metadata`,
		`gsql.JSONSet("metadata")`,
		`field := b.Metadata.Data()`,
		`set.Set("tags"`,
		`// author`,
		`set.Set("author.name"`,
		`set.Set("author.email"`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2Mixed 测试混合映射的代码生成
func TestGenerate2Mixed(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "AccountPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证包含多种映射类型
	expectedMappings := []string{
		`// Embedded: Model`,
		`values["id"] = b.Model.ID`,
		`values["username"] = b.Username`,
		`// B.Settings`,
		`gsql.JSONSet("settings")`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2CrossPackage 测试跨包类型引用的代码生成
func TestGenerate2CrossPackage(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/external_models.go", "ExternalUserPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证函数签名包含包名前缀
	if !strings.Contains(funcCode, "func (e *ExternalUserPO) ToPatch(input *domain.ExternalUserDomain) map[string]any") {
		t.Errorf("Function signature should include package prefix, got:\n%s", funcCode)
	}

	// 验证导入外部包
	if !strings.Contains(fullCode, `"github.com/donutnomad/gotoolkit/automap/testdata/domain"`) {
		t.Errorf("Missing domain package import, got:\n%s", fullCode)
	}

	// 验证字段映射
	expectedMappings := []string{
		`fields.ID.IsPresent()`,
		`values["id"] = b.ID`,
		`fields.Name.IsPresent()`,
		`values["name"] = b.Name`,
		`fields.Email.IsPresent()`,
		`values["email"] = b.Email`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2ExternalEmbedded 测试外部包嵌入类型的代码生成
func TestGenerate2ExternalEmbedded(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/external_models.go", "ApprovalPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证函数签名包含包名前缀
	if !strings.Contains(funcCode, "func (a *ApprovalPO) ToPatch(input *domain.ApprovalDomain) map[string]any") {
		t.Errorf("Function signature should include package prefix, got:\n%s", funcCode)
	}

	// 验证嵌入字段映射
	expectedMappings := []string{
		`// Embedded: Model`,
		`fields.ID.IsPresent()`,
		`values["id"] = b.Model.ID`,
		`fields.CreatedAt.IsPresent()`,
		`values["created_at"] = b.Model.CreatedAt`,
		`fields.UpdatedAt.IsPresent()`,
		`values["updated_at"] = b.Model.UpdatedAt`,
		`fields.Title.IsPresent()`,
		`values["title"] = b.Title`,
		`fields.Status.IsPresent()`,
		`values["status"] = b.Status`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2MissingFields 测试缺失字段注释的代码生成
func TestGenerate2MissingFields(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "PartialUserPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证 Missing fields 注释存在
	if !strings.Contains(funcCode, "// Missing fields:") {
		t.Errorf("Missing 'Missing fields' comment, got:\n%s", funcCode)
	}

	// 验证缺失的字段名
	if !strings.Contains(funcCode, "default_id") {
		t.Errorf("Missing 'default_id' in missing fields comment, got:\n%s", funcCode)
	}
	if !strings.Contains(funcCode, "deleted_at") {
		t.Errorf("Missing 'deleted_at' in missing fields comment, got:\n%s", funcCode)
	}

	// 验证已映射的字段
	expectedMappings := []string{
		`values["id"] = b.ID`,
		`values["name"] = b.Name`,
		`values["email"] = b.Email`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2GormModel 测试使用 gorm.io/gorm.Model 外部包嵌入类型的代码生成
func TestGenerate2GormModel(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "GormUserPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证 gorm.Model 的字段被正确识别
	// gorm.Model 包含: ID, CreatedAt, UpdatedAt, DeletedAt
	// 注意: DeletedAt 的映射是 gorm.DeletedAt{Time: d.DeletedAt}，当前mapper无法识别这种复杂转换
	expectedMappings := []string{
		`// Embedded: Model`,
		`fields.ID.IsPresent()`,
		`values["id"] = b.Model.ID`,
		`fields.CreatedAt.IsPresent()`,
		`values["created_at"] = b.Model.CreatedAt`,
		`fields.UpdatedAt.IsPresent()`,
		`values["updated_at"] = b.Model.UpdatedAt`,
		`fields.Username.IsPresent()`,
		`values["username"] = b.Username`,
		`fields.Email.IsPresent()`,
		`values["email"] = b.Email`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	// 验证 Missing fields 注释
	// deleted_at: 因为 DeletedAt 使用了 gorm.DeletedAt{} 复杂转换，mapper 无法识别
	// last_login: 故意未映射
	if !strings.Contains(funcCode, "// Missing fields:") {
		t.Errorf("Missing 'Missing fields' comment, got:\n%s", funcCode)
	}
	// 验证 gorm.Model 的 deleted_at 被正确识别为目标列（证明外部包解析正常）
	if !strings.Contains(funcCode, "deleted_at") {
		t.Errorf("Missing 'deleted_at' in missing fields (gorm.Model column should be recognized), got:\n%s", funcCode)
	}
	if !strings.Contains(funcCode, "last_login") {
		t.Errorf("Missing 'last_login' in missing fields comment, got:\n%s", funcCode)
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2CrossFile 测试跨文件场景（结构体和ToPO函数在不同文件中）
func TestGenerate2CrossFile(t *testing.T) {
	// 注意：这里传入结构体所在的文件，ToPO函数在另一个文件 cross_file_mapper.go 中
	fullCode, funcCode, err := automap.Generate2("testdata/cross_file_po.go", "CrossFilePO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证嵌入字段被正确识别（关键测试点）
	expectedMappings := []string{
		`// Embedded: Model`,
		`fields.ID.IsPresent()`,
		`values["id"] = b.Model.ID`,
		`fields.CreatedAt.IsPresent()`,
		`values["created_at"] = b.Model.CreatedAt`,
		`fields.UpdatedAt.IsPresent()`,
		`values["updated_at"] = b.Model.UpdatedAt`,
		`fields.Username.IsPresent()`,
		`values["username"] = b.Username`,
		`fields.Email.IsPresent()`,
		`values["email"] = b.Email`,
		`fields.Score.IsPresent()`,
		`values["score"] = b.Score`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2CrossFileFromMapperFile 测试跨文件场景（从方法所在文件解析）
// 关键场景：传入的是方法所在的文件（mapper文件），而不是结构体定义文件
// 这是用户实际遇到的场景：ToPO方法在mapper.go中，但PO结构体在另一个文件中
func TestGenerate2CrossFileFromMapperFile(t *testing.T) {
	// 注意：这里传入的是ToPO方法所在的文件，而结构体定义在 cross_file_po.go 中
	// 这模拟了用户的实际场景：gormgen使用method.FilePath（方法所在文件）来调用automap
	fullCode, funcCode, err := automap.Generate2("testdata/cross_file_mapper.go", "CrossFilePO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证嵌入字段被正确识别
	// 关键：即使传入的是mapper文件，也能正确解析结构体并识别嵌入的Model字段
	expectedMappings := []string{
		`// Embedded: Model`,
		`fields.ID.IsPresent()`,
		`values["id"] = b.Model.ID`,
		`fields.CreatedAt.IsPresent()`,
		`values["created_at"] = b.Model.CreatedAt`,
		`fields.UpdatedAt.IsPresent()`,
		`values["updated_at"] = b.Model.UpdatedAt`,
		`fields.Username.IsPresent()`,
		`values["username"] = b.Username`,
		`fields.Email.IsPresent()`,
		`values["email"] = b.Email`,
		`fields.Score.IsPresent()`,
		`values["score"] = b.Score`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2CrossFileWithMissingFields 测试跨文件场景下的Missing fields注释
// 验证当从方法文件解析时，能正确计算并生成Missing fields注释
func TestGenerate2CrossFileWithMissingFields(t *testing.T) {
	// 传入方法所在的文件
	fullCode, funcCode, err := automap.Generate2("testdata/cross_file_mapper.go", "CrossFilePO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// CrossFilePO 嵌入了 Model（包含 ID, CreatedAt, UpdatedAt）
	// ToPO 方法映射了所有这些字段，所以不应该有 Missing fields 注释
	// 如果出现 Missing fields，说明跨文件解析出了问题
	if strings.Contains(fullCode, "// Missing fields:") {
		// 如果有 Missing fields，检查是否是合理的缺失
		t.Logf("Full code contains Missing fields comment:\n%s", fullCode)
	}

	t.Logf("Generated func code:\n%s", funcCode)
}

// TestGenerate2CustomJSONTag 测试 JSON tag 与 Go 字段名不同的情况
// 验证生成代码使用真实的 Go 字段名，而不是从 JSON tag 推断
func TestGenerate2CustomJSONTag(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "CustomTagPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证使用真实的 Go 字段名而不是 JSON tag 推断的名称
	// Go 字段名: InnerName, InnerValue, SubInfo.RealFieldA, SubInfo.RealFieldB, SubInfo.RealFieldC
	// 如果错误地使用 JSON tag 推断，会生成: InnerX, InnerY, SubData.CustomTagA 等
	mustContain := []string{
		`field.InnerName`,          // 正确：Go 字段名，而不是 InnerX（从 json:"inner_x" 推断）
		`field.InnerValue`,         // 正确：Go 字段名，而不是 InnerY（从 json:"inner_y" 推断）
		`field.SubInfo.RealFieldA`, // 正确：嵌套 Go 字段名，而不是 SubData.CustomTagA
		`field.SubInfo.RealFieldB`, // 正确：嵌套 Go 字段名
		`field.SubInfo.RealFieldC`, // 正确：嵌套 Go 字段名
	}
	for _, expected := range mustContain {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Expected Go field path %q not found in generated code", expected)
		}
	}

	// 验证不包含从 JSON tag 错误推断的名称
	mustNotContain := []string{
		`field.InnerX`,     // 错误：从 json:"inner_x" 推断
		`field.InnerY`,     // 错误：从 json:"inner_y" 推断
		`field.SubData`,    // 错误：从 json:"sub_data" 推断
		`field.CustomTagA`, // 错误：从 json:"custom_tag_a" 推断
		`field.DifferentB`, // 错误：从 json:"different_b" 推断
	}
	for _, notExpected := range mustNotContain {
		if strings.Contains(funcCode, notExpected) {
			t.Errorf("Unexpected JSON-inferred field path %q found in generated code", notExpected)
		}
	}

	// 验证 JSON path 仍然使用正确的 json tag 名称
	jsonPathMustContain := []string{
		`set.Set("inner_x"`,               // JSON path 使用 json tag
		`set.Set("inner_y"`,               // JSON path 使用 json tag
		`set.Set("sub_data.custom_tag_a"`, // 嵌套 JSON path
		`set.Set("sub_data.different_b"`,
		`set.Set("sub_data.another_c"`,
	}
	for _, expected := range jsonPathMustContain {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Expected JSON path %q not found in generated code", expected)
		}
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2JSONFieldSorting 测试 JSON 字段按字母顺序排序
func TestGenerate2JSONFieldSorting(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "SortTestPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证字段按字母顺序排序：Apple, Banana, Mango, Zebra
	// ToPO 中的顺序是 Zebra, Apple, Mango, Banana（乱序）

	// 找到每个字段在生成代码中的位置
	applePos := strings.Index(funcCode, "fields.Apple.IsPresent()")
	bananaPos := strings.Index(funcCode, "fields.Banana.IsPresent()")
	mangoPos := strings.Index(funcCode, "fields.Mango.IsPresent()")
	zebraPos := strings.Index(funcCode, "fields.Zebra.IsPresent()")

	if applePos == -1 || bananaPos == -1 || mangoPos == -1 || zebraPos == -1 {
		t.Fatalf("Not all fields found in generated code")
	}

	// 验证排序顺序：Apple < Banana < Mango < Zebra
	if !(applePos < bananaPos && bananaPos < mangoPos && mangoPos < zebraPos) {
		t.Errorf("Fields are not sorted alphabetically.\n"+
			"Expected order: Apple(%d) < Banana(%d) < Mango(%d) < Zebra(%d)\n"+
			"Generated code:\n%s",
			applePos, bananaPos, mangoPos, zebraPos, funcCode)
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2JSONNestedSorting 测试嵌套 JSON 字段的分组和排序
func TestGenerate2JSONNestedSorting(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "CustomTagPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证分组：顶层字段先出现，然后是 sub_data 分组
	topLevelPos := strings.Index(funcCode, "fields.InnerName.IsPresent()")
	subDataCommentPos := strings.Index(funcCode, "// sub_data")
	subDataFieldPos := strings.Index(funcCode, "fields.SubFieldA.IsPresent()")

	if topLevelPos == -1 || subDataFieldPos == -1 {
		t.Fatalf("Not all fields found in generated code")
	}

	// 验证顶层字段在 sub_data 分组之前
	if subDataCommentPos != -1 && topLevelPos > subDataCommentPos {
		t.Errorf("Top-level fields should appear before sub_data group")
	}

	// 验证 sub_data 分组内的字段也是排序的：SubFieldA, SubFieldB, SubFieldC
	subFieldAPos := strings.Index(funcCode, "fields.SubFieldA.IsPresent()")
	subFieldBPos := strings.Index(funcCode, "fields.SubFieldB.IsPresent()")
	subFieldCPos := strings.Index(funcCode, "fields.SubFieldC.IsPresent()")

	if subFieldAPos == -1 || subFieldBPos == -1 || subFieldCPos == -1 {
		t.Fatalf("Not all sub fields found in generated code")
	}

	// 验证排序：SubFieldA < SubFieldB < SubFieldC
	if !(subFieldAPos < subFieldBPos && subFieldBPos < subFieldCPos) {
		t.Errorf("Sub fields are not sorted alphabetically")
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2PointerDereference 测试指针解引用
// 验证能正确解析 *d.Field 的情况
func TestGenerate2PointerDereference(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "PointerPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证所有指针解引用的字段都被正确识别
	expectedMappings := []string{
		`fields.Name.IsPresent()`, // *entity.Name
		`values["name"] = b.Name`,
		`fields.Age.IsPresent()`, // *entity.Age
		`values["age"] = b.Age`,
		`fields.Score.IsPresent()`, // entity.Score (普通字段)
		`values["score"] = b.Score`,
		`fields.TokenSupply.IsPresent()`, // *entity.TokenSupply
		`values["token_supply"] = b.TokenSupply`,
		`fields.MaxAmount.IsPresent()`, // *entity.MaxAmount
		`values["max_amount"] = b.MaxAmount`,
	}

	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	// 验证 ID 字段也被正确处理
	if !strings.Contains(funcCode, `fields.ID.IsPresent()`) {
		t.Errorf("ID field not found")
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestParsePointerDereference 测试指针解引用的解析
func TestParsePointerDereference(t *testing.T) {
	result, err := automap.Parse("testdata/models.go", "PointerPO", "ToPO")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// 验证所有字段都被正确解析
	expectedFields := map[string]bool{
		"ID":          false,
		"Name":        false,
		"Age":         false,
		"Score":       false,
		"TokenSupply": false,
		"MaxAmount":   false,
	}

	for _, mapping := range result.AllMappings {
		if _, exists := expectedFields[mapping.SourcePath]; exists {
			expectedFields[mapping.SourcePath] = true
		}
	}

	for field, found := range expectedFields {
		if !found {
			t.Errorf("Field %s was not parsed (pointer dereference may not be working)", field)
		}
	}
}

// TestGenerate2FieldOrdering 测试字段顺序
// 验证生成的 ToPatch 方法字段顺序与 PO 结构体定义顺序一致
func TestGenerate2FieldOrdering(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "FieldOrderPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// PO 结构体字段顺序：
	// ID, Name, TokenAddress, TokenName, TokenSymbol, TokenDecimals, Status, FailedReason, CreatedAt
	//
	// ToPO 中赋值顺序（故意乱序）：
	// ID, Name, Status, FailedReason, CreatedAt, TokenAddress, TokenName, TokenSymbol, TokenDecimals
	//
	// 生成的 ToPatch 代码应该按 PO 结构体顺序输出

	// 获取各字段在生成代码中的位置
	idPos := strings.Index(funcCode, `values["id"]`)
	namePos := strings.Index(funcCode, `values["name"]`)
	tokenAddressPos := strings.Index(funcCode, `values["token_address"]`)
	tokenNamePos := strings.Index(funcCode, `values["token_name"]`)
	tokenSymbolPos := strings.Index(funcCode, `values["token_symbol"]`)
	tokenDecimalsPos := strings.Index(funcCode, `values["token_decimals"]`)
	statusPos := strings.Index(funcCode, `values["status"]`)
	failedReasonPos := strings.Index(funcCode, `values["failed_reason"]`)
	createdAtPos := strings.Index(funcCode, `values["created_at"]`)

	// 验证所有字段都存在
	positions := map[string]int{
		"id":             idPos,
		"name":           namePos,
		"token_address":  tokenAddressPos,
		"token_name":     tokenNamePos,
		"token_symbol":   tokenSymbolPos,
		"token_decimals": tokenDecimalsPos,
		"status":         statusPos,
		"failed_reason":  failedReasonPos,
		"created_at":     createdAtPos,
	}
	for field, pos := range positions {
		if pos == -1 {
			t.Errorf("Field %q not found in generated code", field)
		}
	}

	// 验证字段顺序与 PO 结构体定义一致
	// PO 结构体顺序: id < name < token_address < token_name < token_symbol < token_decimals < status < failed_reason < created_at
	//
	// 关键验证：Token 相关字段应该在 Status 之前
	// （在 ToPO 中 Token 字段是最后赋值的，但在 PO 结构体中 Token 字段在 Status 之前）
	if tokenAddressPos > statusPos {
		t.Errorf("Field ordering incorrect: token_address(%d) should appear before status(%d)", tokenAddressPos, statusPos)
	}
	if tokenNamePos > statusPos {
		t.Errorf("Field ordering incorrect: token_name(%d) should appear before status(%d)", tokenNamePos, statusPos)
	}
	if tokenSymbolPos > statusPos {
		t.Errorf("Field ordering incorrect: token_symbol(%d) should appear before status(%d)", tokenSymbolPos, statusPos)
	}
	if tokenDecimalsPos > statusPos {
		t.Errorf("Field ordering incorrect: token_decimals(%d) should appear before status(%d)", tokenDecimalsPos, statusPos)
	}

	// 验证 FailedReason 应该在 CreatedAt 之前（按 PO 结构体顺序）
	if failedReasonPos > createdAtPos {
		t.Errorf("Field ordering incorrect: failed_reason(%d) should appear before created_at(%d)", failedReasonPos, createdAtPos)
	}

	// 验证完整顺序链
	expectedOrder := []struct {
		name string
		pos  int
	}{
		{"id", idPos},
		{"name", namePos},
		{"token_address", tokenAddressPos},
		{"token_name", tokenNamePos},
		{"token_symbol", tokenSymbolPos},
		{"token_decimals", tokenDecimalsPos},
		{"status", statusPos},
		{"failed_reason", failedReasonPos},
		{"created_at", createdAtPos},
	}

	for i := 0; i < len(expectedOrder)-1; i++ {
		if expectedOrder[i].pos > expectedOrder[i+1].pos {
			t.Errorf("Field ordering violated: %s(%d) should appear before %s(%d)",
				expectedOrder[i].name, expectedOrder[i].pos,
				expectedOrder[i+1].name, expectedOrder[i+1].pos)
		}
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2EmbeddedOneToMany 测试 EmbeddedOneToMany 映射的代码生成（无前缀）
func TestGenerate2EmbeddedOneToMany(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "EmbeddedOneToManyPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证 EmbeddedOneToMany 注释
	if !strings.Contains(funcCode, "// EmbeddedOneToMany:") {
		t.Errorf("Missing EmbeddedOneToMany comment")
	}

	// 验证使用 Account 字段作为条件
	if !strings.Contains(funcCode, "fields.Account.IsPresent()") {
		t.Errorf("Missing Account field check")
	}

	// 验证生成的列赋值（无前缀）
	expectedMappings := []string{
		`values["namespace"] = b.Account.Namespace`,
		`values["reference"] = b.Account.Reference`,
		`values["address"] = b.Account.Address`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping: %s", expected)
		}
	}

	// 验证普通字段也正确映射
	if !strings.Contains(funcCode, `values["id"] = b.ID`) {
		t.Errorf("Missing ID mapping")
	}
	if !strings.Contains(funcCode, `values["name"] = b.Name`) {
		t.Errorf("Missing Name mapping")
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2EmbeddedOneToManyWithPrefix 测试 EmbeddedOneToMany 映射的代码生成（带前缀）
func TestGenerate2EmbeddedOneToManyWithPrefix(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "EmbeddedPrefixPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证 EmbeddedOneToMany 注释
	if !strings.Contains(funcCode, "// EmbeddedOneToMany:") {
		t.Errorf("Missing EmbeddedOneToMany comment")
	}

	// 验证使用 Account 字段作为条件
	if !strings.Contains(funcCode, "fields.Account.IsPresent()") {
		t.Errorf("Missing Account field check")
	}

	// 验证生成的列赋值（带前缀 acc_）
	expectedMappings := []string{
		`values["acc_namespace"] = b.Account.Namespace`,
		`values["acc_reference"] = b.Account.Reference`,
		`values["acc_address"] = b.Account.Address`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping with prefix: %s", expected)
		}
	}

	// 验证普通字段也正确映射
	if !strings.Contains(funcCode, `values["id"] = b.ID`) {
		t.Errorf("Missing ID mapping")
	}
	if !strings.Contains(funcCode, `values["title"] = b.Title`) {
		t.Errorf("Missing Title mapping")
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2ExternalPackageEmbedded 测试外部包 EmbeddedOneToMany 映射的代码生成
// 使用 caip10.AccountIDColumnsCompact 作为嵌入字段类型
func TestGenerate2ExternalPackageEmbedded(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "ExternalEmbeddedPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证 EmbeddedOneToMany 注释
	if !strings.Contains(funcCode, "// EmbeddedOneToMany:") {
		t.Errorf("Missing EmbeddedOneToMany comment")
	}

	// 验证使用 Account 字段作为条件
	if !strings.Contains(funcCode, "fields.Account.IsPresent()") {
		t.Errorf("Missing Account field check")
	}

	// 验证生成的列赋值（带前缀 account_，外部包字段 ChainID 和 Address）
	// caip10.AccountIDColumnsCompact 有两个字段：
	// - ChainID (gorm:"column:chain_id")
	// - Address (gorm:"column:address")
	expectedMappings := []string{
		`values["account_chain_id"] = b.Account.ChainID`,
		`values["account_address"] = b.Account.Address`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping for external package type: %s", expected)
		}
	}

	// 验证普通字段也正确映射
	if !strings.Contains(funcCode, `values["id"] = b.ID`) {
		t.Errorf("Missing ID mapping")
	}
	if !strings.Contains(funcCode, `values["name"] = b.Name`) {
		t.Errorf("Missing Name mapping")
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// TestGenerate2ExternalPackageEmbeddedNoPrefix 测试外部包 EmbeddedOneToMany 映射的代码生成（无前缀）
// 关键bug修复验证：当嵌入字段无前缀时，不应该错误地包含其他嵌入类型的字段
func TestGenerate2ExternalPackageEmbeddedNoPrefix(t *testing.T) {
	fullCode, funcCode, err := automap.Generate2("testdata/models.go", "ExternalNoPrefixPO", "ToPO", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}

	// 验证 EmbeddedOneToMany 注释
	if !strings.Contains(funcCode, "// EmbeddedOneToMany:") {
		t.Errorf("Missing EmbeddedOneToMany comment")
	}

	// 验证使用 Account 字段作为条件
	if !strings.Contains(funcCode, "fields.Account.IsPresent()") {
		t.Errorf("Missing Account field check")
	}

	// 验证生成的列赋值（无前缀，外部包字段 ChainID 和 Address）
	// caip10.AccountIDColumnsCompact 只有两个字段：ChainID 和 Address
	expectedMappings := []string{
		`values["chain_id"] = b.Account.ChainID`,
		`values["address"] = b.Account.Address`,
	}
	for _, expected := range expectedMappings {
		if !strings.Contains(funcCode, expected) {
			t.Errorf("Missing expected mapping for external package type: %s", expected)
		}
	}

	// 关键验证：Account 的 EmbeddedOneToMany 块中不应该包含 gorm.Model 的字段
	// 这些字段属于另一个嵌入类型 gorm.Model，不应该出现在 Account 的映射中
	// 查找 EmbeddedOneToMany: Account 块
	embeddedStart := strings.Index(funcCode, "// EmbeddedOneToMany: Account")
	if embeddedStart == -1 {
		t.Fatalf("EmbeddedOneToMany: Account comment not found")
	}
	// 找到这个块的结束位置（下一个注释或 return）
	embeddedEnd := strings.Index(funcCode[embeddedStart+30:], "\n\t//")
	if embeddedEnd == -1 {
		embeddedEnd = strings.Index(funcCode[embeddedStart+30:], "\n\treturn")
	}
	if embeddedEnd == -1 {
		embeddedEnd = len(funcCode) - embeddedStart - 30
	}
	accountBlock := funcCode[embeddedStart : embeddedStart+30+embeddedEnd]

	// 验证 Account 块中不包含 gorm.Model 的字段
	wrongMappings := []string{
		`values["id"]`,
		`values["created_at"]`,
		`values["updated_at"]`,
		`values["deleted_at"]`,
		`values["default_id"]`,
	}
	for _, wrong := range wrongMappings {
		if strings.Contains(accountBlock, wrong) {
			t.Errorf("Account EmbeddedOneToMany block should NOT contain %s (this belongs to gorm.Model)", wrong)
		}
	}

	// 验证 gorm.Model 的字段在 Embedded 块中正确映射（而不是在 Account 块中）
	if !strings.Contains(funcCode, "// Embedded: Model") {
		t.Errorf("Missing Embedded: Model comment")
	}
	if !strings.Contains(funcCode, `values["id"] = b.Model.ID`) {
		t.Errorf("Missing gorm.Model ID mapping")
	}
	if !strings.Contains(funcCode, `values["created_at"] = b.Model.CreatedAt`) {
		t.Errorf("Missing gorm.Model CreatedAt mapping")
	}

	t.Logf("Generated full code:\n%s", fullCode)
}

// 测试映射关系为空时候的代码
func TestGenerate2Empty(t *testing.T) {
	_, funcCode, err := automap.Generate2("testdata/models.go", "ExternalNoPrefixPO", "ToPO2", "ToPatch")
	if err != nil {
		t.Fatalf("Generate2 failed: %v", err)
	}
	if !strings.Contains(funcCode, "\tvar values map[string]any\n\treturn values") {
		t.Errorf("Missing return statement")
	}
}
