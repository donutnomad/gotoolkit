package automap

// ============================================================================
// 新的映射结果类型定义
// ============================================================================

// MappingType 映射类型
type MappingType string

const (
	// OneToOne 一对一映射：Domain.Field -> PO.Field -> Column
	OneToOne MappingType = "one_to_one"

	// OneToMany 一对多映射：Domain.Struct -> PO.(多个字段) -> 多个Column
	// 例如：Domain.Location -> PO.Country, Province, City, District
	OneToMany MappingType = "one_to_many"

	// ManyToOne 多对一映射：Domain.(多个字段) -> PO.JSONField -> 一个Column
	// 例如：Domain.Phone, Address, City -> PO.Contact (JSON)
	ManyToOne MappingType = "many_to_one"

	// Embedded 嵌入映射：Domain.(多个字段) -> PO.Embedded -> 多个Column
	// 例如：Domain.ID, CreatedAt, UpdatedAt -> PO.Model -> id, created_at, updated_at
	Embedded MappingType = "embedded"

	// MethodCall 方法调用映射：Domain.(多个字段) -> Method() -> PO.Field -> 一个Column
	// 例如：Domain.Country, Province, City, Street -> GetAddress() -> PO.Address
	MethodCall MappingType = "method_call"

	// EmbeddedOneToMany 嵌入一对多映射：Domain.Field -> PO.Embedded -> 多个Column
	// 例如：Domain.Account -> PO.Account (gorm:embedded) -> namespace, reference, address
	// 这与 OneToMany 的区别是：目标是一个带 gorm:"embedded" 标签的字段
	// 生成的代码：values["namespace"] = b.Account.Namespace
	EmbeddedOneToMany MappingType = "embedded_one_to_many"
)

// FieldMapping2 单个字段的映射关系（新版本）
type FieldMapping2 struct {
	// SourcePath Domain字段路径，支持嵌套如 "Location.Country"
	SourcePath string

	// TargetPath PO字段路径，如 "Model.ID" 或 "Contact"
	TargetPath string

	// ColumnName 数据库列名
	ColumnName string

	// ConvertExpr 转换表达式，如 ".Unix()", "decimal.NewFromBigInt(...)"
	ConvertExpr string

	// JSONPath JSON内部路径（仅ManyToOne时有效），如 "author.name"（json tag 路径）
	JSONPath string

	// GoFieldPath JSON内部Go字段路径（仅ManyToOne时有效），如 "Author.Name"（真实Go字段名）
	GoFieldPath string

	// FieldPosition 在 PO 结构体中的字段位置（用于排序）
	FieldPosition int
}

// MappingGroup 映射组（表示一组相关的映射）
type MappingGroup struct {
	// Type 映射类型
	Type MappingType

	// SourceField 源字段名（对于OneToMany，这是Domain中的结构体字段名）
	SourceField string

	// TargetField 目标字段名（对于ManyToOne/Embedded/MethodCall，这是PO中的字段名）
	TargetField string

	// MethodName 方法名（对于MethodCall，这是调用的方法名）
	MethodName string

	// FieldPosition 在 PO 结构体中的字段位置（用于排序，确保生成代码顺序与 PO 定义一致）
	FieldPosition int

	// Mappings 具体的字段映射列表
	Mappings []FieldMapping2
}

// ParseResult2 解析结果（新版本）
type ParseResult2 struct {
	// FuncName 函数名
	FuncName string

	// ReceiverType 接收者类型
	ReceiverType string

	// SourceType 源类型（Domain）
	SourceType string

	// SourceTypePackage 源类型所在的包名（如果是外部包）
	// 例如：源类型是 domain.ListingDomain，则 SourceTypePackage = "domain"
	SourceTypePackage string

	// SourceTypeImportPath 源类型的完整导入路径
	// 例如："github.com/donutnomad/project/domain"
	SourceTypeImportPath string

	// TargetType 目标类型（PO）
	TargetType string

	// TargetColumns 目标类型（PO）的所有数据库列名
	// 用于验证字段覆盖情况
	TargetColumns []string

	// TargetFieldPositions 目标类型字段位置映射：列名 -> 位置
	// 用于生成代码时按 PO 结构体字段顺序排序
	TargetFieldPositions map[string]int

	// Groups 映射组列表
	Groups []MappingGroup

	// AllMappings 所有映射的扁平列表（便于遍历）
	AllMappings []FieldMapping2
}

// Parse 解析 ToPO 函数，返回映射关系
// filePath: 源文件路径
// receiverType: 接收者类型名（如 "SimpleUserPO"）
// funcName: 函数名（如 "ToPO"）
func Parse(filePath string, receiverType string, funcName string) (*ParseResult2, error) {
	mapper := NewMapper(filePath)
	return mapper.Parse(receiverType, funcName)
}
