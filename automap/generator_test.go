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
