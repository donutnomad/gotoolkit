package testdata

// ============================================================================
// 映射结果类型定义
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
)

// FieldMapping 单个字段的映射关系
type FieldMapping struct {
	// SourcePath Domain字段路径，支持嵌套如 "Location.Country"
	SourcePath string

	// TargetPath PO字段路径，如 "Model.ID" 或 "Contact"
	TargetPath string

	// ColumnName 数据库列名
	ColumnName string

	// ConvertExpr 转换表达式，如 ".Unix()", "decimal.NewFromBigInt(...)"
	ConvertExpr string

	// JSONPath JSON内部路径（仅ManyToOne时有效），如 "author.name"
	JSONPath string
}

// MappingGroup 映射组（表示一组相关的映射）
type MappingGroup struct {
	// Type 映射类型
	Type MappingType

	// SourceField 源字段名（对于OneToMany，这是Domain中的结构体字段名）
	SourceField string

	// TargetField 目标字段名（对于ManyToOne/Embedded，这是PO中的字段名）
	TargetField string

	// MethodName 方法名（对于MethodCall，这是调用的方法名）
	MethodName string

	// Mappings 具体的字段映射列表
	Mappings []FieldMapping
}

// ParseResult 解析结果
type ParseResult struct {
	// FuncName 函数名
	FuncName string

	// ReceiverType 接收者类型
	ReceiverType string

	// SourceType 源类型（Domain）
	SourceType string

	// TargetType 目标类型（PO）
	TargetType string

	// Groups 映射组列表
	Groups []MappingGroup

	// AllMappings 所有映射的扁平列表（便于遍历）
	AllMappings []FieldMapping
}

// ============================================================================
// 期望的测试结果
// ============================================================================

// 场景1: 一对一映射 - SimpleUserPO.ToPO
var ExpectedSimpleUserMapping = ParseResult{
	FuncName:     "ToPO",
	ReceiverType: "SimpleUserPO",
	SourceType:   "SimpleUserDomain",
	TargetType:   "SimpleUserPO",
	Groups: []MappingGroup{
		{
			Type: OneToOne,
			Mappings: []FieldMapping{
				{SourcePath: "ID", TargetPath: "ID", ColumnName: "id"},
				{SourcePath: "Name", TargetPath: "Name", ColumnName: "name"},
				{SourcePath: "Email", TargetPath: "Email", ColumnName: "email"},
				{SourcePath: "Age", TargetPath: "Age", ColumnName: "age"},
			},
		},
	},
}

// 场景2: Embedded映射 - EmbeddedUserPO.ToPO
var ExpectedEmbeddedUserMapping = ParseResult{
	FuncName:     "ToPO",
	ReceiverType: "EmbeddedUserPO",
	SourceType:   "EmbeddedUserDomain",
	TargetType:   "EmbeddedUserPO",
	Groups: []MappingGroup{
		{
			Type:        Embedded,
			TargetField: "Model",
			Mappings: []FieldMapping{
				{SourcePath: "ID", TargetPath: "Model.ID", ColumnName: "id"},
				{SourcePath: "CreatedAt", TargetPath: "Model.CreatedAt", ColumnName: "created_at"},
				{SourcePath: "UpdatedAt", TargetPath: "Model.UpdatedAt", ColumnName: "updated_at"},
			},
		},
		{
			Type: OneToOne,
			Mappings: []FieldMapping{
				{SourcePath: "Name", TargetPath: "Name", ColumnName: "name"},
				{SourcePath: "Status", TargetPath: "Status", ColumnName: "status"},
			},
		},
	},
}

// 场景3: 多对一映射（JSON）- ProfilePO.ToPO
var ExpectedProfileMapping = ParseResult{
	FuncName:     "ToPO",
	ReceiverType: "ProfilePO",
	SourceType:   "ProfileDomain",
	TargetType:   "ProfilePO",
	Groups: []MappingGroup{
		{
			Type: OneToOne,
			Mappings: []FieldMapping{
				{SourcePath: "ID", TargetPath: "ID", ColumnName: "id"},
				{SourcePath: "Name", TargetPath: "Name", ColumnName: "name"},
				{SourcePath: "Score", TargetPath: "Score", ColumnName: "score"},
			},
		},
		{
			Type:        ManyToOne,
			TargetField: "Contact",
			Mappings: []FieldMapping{
				{SourcePath: "Phone", TargetPath: "Contact", ColumnName: "contact", JSONPath: "phone"},
				{SourcePath: "Address", TargetPath: "Contact", ColumnName: "contact", JSONPath: "address"},
				{SourcePath: "City", TargetPath: "Contact", ColumnName: "contact", JSONPath: "city"},
			},
		},
	},
}

// 场景4: 一对多映射 - CompanyPO.ToPO
var ExpectedCompanyMapping = ParseResult{
	FuncName:     "ToPO",
	ReceiverType: "CompanyPO",
	SourceType:   "CompanyDomain",
	TargetType:   "CompanyPO",
	Groups: []MappingGroup{
		{
			Type: OneToOne,
			Mappings: []FieldMapping{
				{SourcePath: "ID", TargetPath: "ID", ColumnName: "id"},
				{SourcePath: "Name", TargetPath: "Name", ColumnName: "name"},
			},
		},
		{
			Type:        OneToMany,
			SourceField: "Location",
			Mappings: []FieldMapping{
				{SourcePath: "Location.Country", TargetPath: "Country", ColumnName: "country"},
				{SourcePath: "Location.Province", TargetPath: "Province", ColumnName: "province"},
				{SourcePath: "Location.City", TargetPath: "City", ColumnName: "city"},
				{SourcePath: "Location.District", TargetPath: "District", ColumnName: "district"},
			},
		},
	},
}

// 场景5: 嵌入带前缀 - AuditPO.ToPO
var ExpectedAuditMapping = ParseResult{
	FuncName:     "ToPO",
	ReceiverType: "AuditPO",
	SourceType:   "AuditDomain",
	TargetType:   "AuditPO",
	Groups: []MappingGroup{
		{
			Type: OneToOne,
			Mappings: []FieldMapping{
				{SourcePath: "ID", TargetPath: "ID", ColumnName: "id"},
				{SourcePath: "Title", TargetPath: "Title", ColumnName: "title"},
			},
		},
		{
			Type:        Embedded,
			TargetField: "Audit",
			Mappings: []FieldMapping{
				{SourcePath: "CreatedBy", TargetPath: "Audit.CreatedBy", ColumnName: "audit_created_by"},
				{SourcePath: "CreatedAt", TargetPath: "Audit.CreatedAt", ColumnName: "audit_created_at"},
				{SourcePath: "UpdatedBy", TargetPath: "Audit.UpdatedBy", ColumnName: "audit_updated_by"},
				{SourcePath: "UpdatedAt", TargetPath: "Audit.UpdatedAt", ColumnName: "audit_updated_at"},
			},
		},
	},
}

// 场景6: 复杂嵌套JSON - ArticlePO.ToPO
var ExpectedArticleMapping = ParseResult{
	FuncName:     "ToPO",
	ReceiverType: "ArticlePO",
	SourceType:   "ArticleDomain",
	TargetType:   "ArticlePO",
	Groups: []MappingGroup{
		{
			Type: OneToOne,
			Mappings: []FieldMapping{
				{SourcePath: "ID", TargetPath: "ID", ColumnName: "id"},
				{SourcePath: "Title", TargetPath: "Title", ColumnName: "title"},
				{SourcePath: "Content", TargetPath: "Content", ColumnName: "content"},
				{SourcePath: "ViewCount", TargetPath: "ViewCount", ColumnName: "view_count"},
			},
		},
		{
			Type:        ManyToOne,
			TargetField: "Metadata",
			Mappings: []FieldMapping{
				{SourcePath: "Tags", TargetPath: "Metadata", ColumnName: "metadata", JSONPath: "tags"},
				{SourcePath: "AuthorName", TargetPath: "Metadata", ColumnName: "metadata", JSONPath: "author.name"},
				{SourcePath: "AuthorEmail", TargetPath: "Metadata", ColumnName: "metadata", JSONPath: "author.email"},
				{SourcePath: "Version", TargetPath: "Metadata", ColumnName: "metadata", JSONPath: "version"},
				{SourcePath: "Priority", TargetPath: "Metadata", ColumnName: "metadata", JSONPath: "priority"},
			},
		},
	},
}

// 场景7: 类型转换 - TimestampPO.ToPO
var ExpectedTimestampMapping = ParseResult{
	FuncName:     "ToPO",
	ReceiverType: "TimestampPO",
	SourceType:   "TimestampDomain",
	TargetType:   "TimestampPO",
	Groups: []MappingGroup{
		{
			Type: OneToOne,
			Mappings: []FieldMapping{
				{SourcePath: "ID", TargetPath: "ID", ColumnName: "id"},
				{SourcePath: "Name", TargetPath: "Name", ColumnName: "name"},
				{SourcePath: "CreateTime", TargetPath: "CreateTime", ColumnName: "create_time", ConvertExpr: ".Unix()"},
				{SourcePath: "UpdateTime", TargetPath: "UpdateTime", ColumnName: "update_time", ConvertExpr: ".Unix()"},
				{SourcePath: "ExpireTime", TargetPath: "ExpireTime", ColumnName: "expire_time", ConvertExpr: ".Unix()"},
			},
		},
	},
}

// 场景8: 混合映射 - AccountPO.ToPO
var ExpectedAccountMapping = ParseResult{
	FuncName:     "ToPO",
	ReceiverType: "AccountPO",
	SourceType:   "AccountDomain",
	TargetType:   "AccountPO",
	Groups: []MappingGroup{
		{
			Type:        Embedded,
			TargetField: "Model",
			Mappings: []FieldMapping{
				{SourcePath: "ID", TargetPath: "Model.ID", ColumnName: "id"},
				{SourcePath: "CreatedAt", TargetPath: "Model.CreatedAt", ColumnName: "created_at"},
				{SourcePath: "UpdatedAt", TargetPath: "Model.UpdatedAt", ColumnName: "updated_at"},
			},
		},
		{
			Type: OneToOne,
			Mappings: []FieldMapping{
				{SourcePath: "Username", TargetPath: "Username", ColumnName: "username"},
				{SourcePath: "Email", TargetPath: "Email", ColumnName: "email"},
				{SourcePath: "Status", TargetPath: "Status", ColumnName: "status"},
				{SourcePath: "LastLogin", TargetPath: "LastLogin", ColumnName: "last_login", ConvertExpr: ".Unix()"},
			},
		},
		{
			Type:        ManyToOne,
			TargetField: "Settings",
			Mappings: []FieldMapping{
				{SourcePath: "Theme", TargetPath: "Settings", ColumnName: "settings", JSONPath: "theme"},
				{SourcePath: "Language", TargetPath: "Settings", ColumnName: "settings", JSONPath: "language"},
				{SourcePath: "Timezone", TargetPath: "Settings", ColumnName: "settings", JSONPath: "timezone"},
			},
		},
	},
}

// 场景9: 局部变量映射 - ProductPO.ToPO
var ExpectedProductMapping = ParseResult{
	FuncName:     "ToPO",
	ReceiverType: "ProductPO",
	SourceType:   "ProductDomain",
	TargetType:   "ProductPO",
	Groups: []MappingGroup{
		{
			Type: OneToOne,
			Mappings: []FieldMapping{
				{SourcePath: "ID", TargetPath: "ID", ColumnName: "id"},
				{SourcePath: "Name", TargetPath: "Name", ColumnName: "name"},                      // 通过局部变量 name
				{SourcePath: "Description", TargetPath: "Description", ColumnName: "description"}, // 通过局部变量 desc
				{SourcePath: "Price", TargetPath: "Price", ColumnName: "price"},                   // 通过局部变量 price -> finalPrice
				{SourcePath: "Stock", TargetPath: "Stock", ColumnName: "stock"},                   // 直接赋值
			},
		},
	},
}

// 场景10: 局部变量 + JSON 映射 - OrderPO.ToPO
var ExpectedOrderMapping = ParseResult{
	FuncName:     "ToPO",
	ReceiverType: "OrderPO",
	SourceType:   "OrderDomain",
	TargetType:   "OrderPO",
	Groups: []MappingGroup{
		{
			Type: OneToOne,
			Mappings: []FieldMapping{
				{SourcePath: "ID", TargetPath: "ID", ColumnName: "id"},
				{SourcePath: "Amount", TargetPath: "Amount", ColumnName: "amount"},
			},
		},
		{
			Type:        ManyToOne,
			TargetField: "Info",
			Mappings: []FieldMapping{
				{SourcePath: "OrderNo", TargetPath: "Info", ColumnName: "info", JSONPath: "order_no"},       // 通过局部变量 orderNo
				{SourcePath: "CustomerID", TargetPath: "Info", ColumnName: "info", JSONPath: "customer_id"}, // 通过局部变量 custID
				{SourcePath: "Remark", TargetPath: "Info", ColumnName: "info", JSONPath: "remark"},          // 通过局部变量 remark
			},
		},
	},
}

// 场景11: 方法调用映射 - CustomerPO.ToPO
var ExpectedCustomerMapping = ParseResult{
	FuncName:     "ToPO",
	ReceiverType: "CustomerPO",
	SourceType:   "CustomerDomain",
	TargetType:   "CustomerPO",
	Groups: []MappingGroup{
		{
			Type: OneToOne,
			Mappings: []FieldMapping{
				{SourcePath: "ID", TargetPath: "ID", ColumnName: "id"},
				{SourcePath: "Name", TargetPath: "Name", ColumnName: "name"},
			},
		},
		{
			Type:        MethodCall,
			TargetField: "Address",
			MethodName:  "GetAddress",
			Mappings: []FieldMapping{
				// GetAddress() 内部使用的字段（按字母顺序）
				{SourcePath: "City", TargetPath: "Address", ColumnName: "address"},
				{SourcePath: "Country", TargetPath: "Address", ColumnName: "address"},
				{SourcePath: "Province", TargetPath: "Address", ColumnName: "address"},
				{SourcePath: "Street", TargetPath: "Address", ColumnName: "address"},
			},
		},
	},
}

// 场景12: 方法调用 + 局部变量 - ShippingPO.ToPO
var ExpectedShippingMapping = ParseResult{
	FuncName:     "ToPO",
	ReceiverType: "ShippingPO",
	SourceType:   "ShippingDomain",
	TargetType:   "ShippingPO",
	Groups: []MappingGroup{
		{
			Type: OneToOne,
			Mappings: []FieldMapping{
				{SourcePath: "ID", TargetPath: "ID", ColumnName: "id"},
				{SourcePath: "ReceiverName", TargetPath: "ReceiverName", ColumnName: "receiver_name"},
			},
		},
		{
			Type:        MethodCall,
			TargetField: "Address",
			MethodName:  "GetShippingAddress",
			Mappings: []FieldMapping{
				// GetShippingAddress() 内部使用的字段（按字母顺序）
				{SourcePath: "City", TargetPath: "Address", ColumnName: "address"},
				{SourcePath: "Country", TargetPath: "Address", ColumnName: "address"},
				{SourcePath: "Detail", TargetPath: "Address", ColumnName: "address"},
				{SourcePath: "Province", TargetPath: "Address", ColumnName: "address"},
			},
		},
	},
}

// 场景18: JSONSlice + lo.Map 映射 - JSONSlicePO.ToPO
// datatypes.NewJSONSlice(lo.Map(...)) 和 datatypes.NewJSONSlice(entity.Field) 模式都作为一对一映射处理
var ExpectedJSONSliceMapping = ParseResult{
	FuncName:     "ToPO",
	ReceiverType: "JSONSlicePO",
	SourceType:   "JSONSliceDomain",
	TargetType:   "JSONSlicePO",
	Groups: []MappingGroup{
		{
			Type: OneToOne,
			Mappings: []FieldMapping{
				{SourcePath: "ID", TargetPath: "ID", ColumnName: "id"},
				{SourcePath: "Name", TargetPath: "Name", ColumnName: "name"},
				{SourcePath: "ExchangeRules", TargetPath: "ExchangeRules", ColumnName: "exchange_rules"}, // lo.Map 映射
				{SourcePath: "Tags", TargetPath: "Tags", ColumnName: "tags"},                             // 直接传入字段
			},
		},
	},
}

// 场景19: JSONSlice + lo.Map + 方法调用映射 - JSONSliceMethodPO.ToPO
// datatypes.NewJSONSlice(lo.Map(entity.GetMethod(), ...)) 模式，方法内部使用的字段被识别
var ExpectedJSONSliceMethodMapping = ParseResult{
	FuncName:     "ToPO",
	ReceiverType: "JSONSliceMethodPO",
	SourceType:   "JSONSliceMethodDomain",
	TargetType:   "JSONSliceMethodPO",
	Groups: []MappingGroup{
		{
			Type: OneToOne,
			Mappings: []FieldMapping{
				{SourcePath: "ID", TargetPath: "ID", ColumnName: "id"},
				{SourcePath: "Name", TargetPath: "Name", ColumnName: "name"},
			},
		},
		{
			Type:        MethodCall,
			TargetField: "Rules",
			MethodName:  "GetRules",
			Mappings: []FieldMapping{
				// GetRules() 内部使用的字段（按字母顺序）
				{SourcePath: "RuleKey", TargetPath: "Rules", ColumnName: "rules"},
				{SourcePath: "RuleValue", TargetPath: "Rules", ColumnName: "rules"},
			},
		},
	},
}
