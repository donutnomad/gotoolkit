package testdata

import (
	"time"

	"github.com/donutnomad/xchain/caip10"
	"github.com/samber/lo"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ============================================================================
// ORM Model - 嵌入式基础模型
// ============================================================================

// Model 基础模型（模拟 gorm.io/gorm 的 Model）
type Model struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// ============================================================================
// 测试场景1: 一对一映射 (OneToOne)
// Domain 字段直接映射到 PO 字段
// ============================================================================

// SimpleUserDomain 简单用户领域模型
type SimpleUserDomain struct {
	ID    uint64
	Name  string
	Email string
	Age   int
}

// SimpleUserPO 简单用户持久化模型
type SimpleUserPO struct {
	ID    uint64 `gorm:"column:id;primaryKey"`
	Name  string `gorm:"column:name"`
	Email string `gorm:"column:email"`
	Age   int    `gorm:"column:age"`
}

// ToPO 一对一映射示例
func (p *SimpleUserPO) ToPO(d *SimpleUserDomain) *SimpleUserPO {
	if d == nil {
		return nil
	}
	return &SimpleUserPO{
		ID:    d.ID,
		Name:  d.Name,
		Email: d.Email,
		Age:   d.Age,
	}
}

// ============================================================================
// 测试场景2: GORM Embedded (一对多 - PO嵌入字段展开为多个数据库列)
// PO 中的 orm.Model 嵌入字段在数据库中对应多个独立的 column
// ============================================================================

// EmbeddedUserDomain 带时间戳的用户领域模型
type EmbeddedUserDomain struct {
	ID        uint64
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string
	Status    int
}

// EmbeddedUserPO 带嵌入字段的用户持久化模型
type EmbeddedUserPO struct {
	Model         // gorm embedded - 展开为 id, created_at, updated_at 三个列
	Name   string `gorm:"column:name"`
	Status int    `gorm:"column:status"`
}

// ToPO Embedded字段映射示例
// Domain 的多个字段 -> PO 的嵌入结构体
// 这是一种"多对一"视角：多个 domain 字段组合成 PO 的一个嵌入字段
// 同时也是"一对多"视角：PO 的嵌入字段展开为数据库的多个列
func (p *EmbeddedUserPO) ToPO(d *EmbeddedUserDomain) *EmbeddedUserPO {
	if d == nil {
		return nil
	}
	return &EmbeddedUserPO{
		Model: Model{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		},
		Name:   d.Name,
		Status: d.Status,
	}
}

// ============================================================================
// 测试场景3: 多对一映射 - JSON字段 (ManyToOne)
// Domain 中的多个字段存储为数据库的一个 JSON 字段
// ============================================================================

// ContactInfo JSON 子结构
type ContactInfo struct {
	Phone   string `json:"phone"`
	Address string `json:"address"`
	City    string `json:"city"`
}

// ProfileDomain 用户资料领域模型（字段分散）
type ProfileDomain struct {
	ID      uint64
	Name    string
	Phone   string // 这些字段将被合并到 JSON
	Address string // 这些字段将被合并到 JSON
	City    string // 这些字段将被合并到 JSON
	Score   int
}

// ProfilePO 用户资料持久化模型
type ProfilePO struct {
	ID      uint64                          `gorm:"column:id;primaryKey"`
	Name    string                          `gorm:"column:name"`
	Contact datatypes.JSONType[ContactInfo] `gorm:"column:contact;type:json"` // 多个字段合并为一个JSON
	Score   int                             `gorm:"column:score"`
}

// ToPO 多对一映射示例
// Domain 的 Phone, Address, City -> PO 的 Contact (JSON)
func (p *ProfilePO) ToPO(d *ProfileDomain) *ProfilePO {
	if d == nil {
		return nil
	}
	return &ProfilePO{
		ID:   d.ID,
		Name: d.Name,
		Contact: datatypes.NewJSONType(ContactInfo{
			Phone:   d.Phone,
			Address: d.Address,
			City:    d.City,
		}),
		Score: d.Score,
	}
}

// ============================================================================
// 测试场景4: 一对多映射 (OneToMany)
// Domain 中的一个结构体字段映射到数据库的多个独立列
// ============================================================================

// LocationInfo 位置信息（嵌套在 Domain 中）
type LocationInfo struct {
	Country  string
	Province string
	City     string
	District string
}

// CompanyDomain 公司领域模型
type CompanyDomain struct {
	ID       uint64
	Name     string
	Location LocationInfo // 一个结构体字段
}

// CompanyPO 公司持久化模型
type CompanyPO struct {
	ID       uint64 `gorm:"column:id;primaryKey"`
	Name     string `gorm:"column:name"`
	Country  string `gorm:"column:country"`  // Location.Country
	Province string `gorm:"column:province"` // Location.Province
	City     string `gorm:"column:city"`     // Location.City
	District string `gorm:"column:district"` // Location.District
}

// ToPO 一对多映射示例
// Domain 的 Location (struct) -> PO 的 Country, Province, City, District (4个独立列)
func (p *CompanyPO) ToPO(d *CompanyDomain) *CompanyPO {
	if d == nil {
		return nil
	}
	return &CompanyPO{
		ID:       d.ID,
		Name:     d.Name,
		Country:  d.Location.Country,
		Province: d.Location.Province,
		City:     d.Location.City,
		District: d.Location.District,
	}
}

// ============================================================================
// 测试场景5: GORM Embedded with Prefix (带前缀的嵌入)
// ============================================================================

// Audit 审计信息
type Audit struct {
	CreatedBy string    `gorm:"column:created_by"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedBy string    `gorm:"column:updated_by"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// AuditDomain 审计领域模型
type AuditDomain struct {
	ID        uint64
	Title     string
	CreatedBy string
	CreatedAt time.Time
	UpdatedBy string
	UpdatedAt time.Time
}

// AuditPO 带审计嵌入的持久化模型
type AuditPO struct {
	ID    uint64 `gorm:"column:id;primaryKey"`
	Title string `gorm:"column:title"`
	Audit Audit  `gorm:"embedded;embeddedPrefix:audit_"` // 嵌入带前缀
}

// ToPO 嵌入带前缀映射示例
func (p *AuditPO) ToPO(d *AuditDomain) *AuditPO {
	if d == nil {
		return nil
	}
	return &AuditPO{
		ID:    d.ID,
		Title: d.Title,
		Audit: Audit{
			CreatedBy: d.CreatedBy,
			CreatedAt: d.CreatedAt,
			UpdatedBy: d.UpdatedBy,
			UpdatedAt: d.UpdatedAt,
		},
	}
}

// ============================================================================
// 测试场景6: 复杂嵌套 JSON (多层嵌套的多对一)
// ============================================================================

// AuthorInfo 作者信息
type AuthorInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Metadata 元数据
type Metadata struct {
	Tags     []string   `json:"tags"`
	Author   AuthorInfo `json:"author"`
	Version  string     `json:"version"`
	Priority int        `json:"priority"`
}

// ArticleDomain 文章领域模型
type ArticleDomain struct {
	ID          uint64
	Title       string
	Content     string
	Tags        []string // -> Metadata.Tags
	AuthorName  string   // -> Metadata.Author.Name
	AuthorEmail string   // -> Metadata.Author.Email
	Version     string   // -> Metadata.Version
	Priority    int      // -> Metadata.Priority
	ViewCount   int
}

// ArticlePO 文章持久化模型
type ArticlePO struct {
	ID        uint64                       `gorm:"column:id;primaryKey"`
	Title     string                       `gorm:"column:title"`
	Content   string                       `gorm:"column:content"`
	Metadata  datatypes.JSONType[Metadata] `gorm:"column:metadata;type:json"` // 嵌套JSON
	ViewCount int                          `gorm:"column:view_count"`
}

// ToPO 复杂嵌套JSON映射示例
func (p *ArticlePO) ToPO(d *ArticleDomain) *ArticlePO {
	if d == nil {
		return nil
	}
	return &ArticlePO{
		ID:      d.ID,
		Title:   d.Title,
		Content: d.Content,
		Metadata: datatypes.NewJSONType(Metadata{
			Tags: d.Tags,
			Author: AuthorInfo{
				Name:  d.AuthorName,
				Email: d.AuthorEmail,
			},
			Version:  d.Version,
			Priority: d.Priority,
		}),
		ViewCount: d.ViewCount,
	}
}

// ============================================================================
// 测试场景7: 类型转换映射
// ============================================================================

// TimestampDomain 时间戳领域模型
type TimestampDomain struct {
	ID         uint64
	Name       string
	CreateTime time.Time
	UpdateTime time.Time
	ExpireTime time.Time
}

// TimestampPO 时间戳持久化模型（存储为 Unix 时间戳）
type TimestampPO struct {
	ID         uint64 `gorm:"column:id;primaryKey"`
	Name       string `gorm:"column:name"`
	CreateTime int64  `gorm:"column:create_time"` // time.Time -> int64
	UpdateTime int64  `gorm:"column:update_time"` // time.Time -> int64
	ExpireTime int64  `gorm:"column:expire_time"` // time.Time -> int64
}

// ToPO 类型转换映射示例
func (p *TimestampPO) ToPO(d *TimestampDomain) *TimestampPO {
	if d == nil {
		return nil
	}
	return &TimestampPO{
		ID:         d.ID,
		Name:       d.Name,
		CreateTime: d.CreateTime.Unix(),
		UpdateTime: d.UpdateTime.Unix(),
		ExpireTime: d.ExpireTime.Unix(),
	}
}

// ============================================================================
// 测试场景8: 混合映射（综合场景）
// ============================================================================

// Settings 设置信息
type Settings struct {
	Theme    string `json:"theme"`
	Language string `json:"language"`
	Timezone string `json:"timezone"`
}

// AccountDomain 账户领域模型
type AccountDomain struct {
	// 基础字段
	ID        uint64
	CreatedAt time.Time
	UpdatedAt time.Time
	// 账户信息
	Username string
	Email    string
	// 设置信息（将合并为JSON）
	Theme    string
	Language string
	Timezone string
	// 状态
	Status    int
	LastLogin time.Time
}

// AccountPO 账户持久化模型
type AccountPO struct {
	Model                                  // embedded -> id, created_at, updated_at
	Username  string                       `gorm:"column:username"`
	Email     string                       `gorm:"column:email"`
	Settings  datatypes.JSONType[Settings] `gorm:"column:settings;type:json"` // 多对一 JSON
	Status    int                          `gorm:"column:status"`
	LastLogin int64                        `gorm:"column:last_login"` // time.Time -> int64
}

// ToPO 混合映射示例
func (p *AccountPO) ToPO(d *AccountDomain) *AccountPO {
	if d == nil {
		return nil
	}
	return &AccountPO{
		Model: Model{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		},
		Username: d.Username,
		Email:    d.Email,
		Settings: datatypes.NewJSONType(Settings{
			Theme:    d.Theme,
			Language: d.Language,
			Timezone: d.Timezone,
		}),
		Status:    d.Status,
		LastLogin: d.LastLogin.Unix(),
	}
}

// ============================================================================
// 测试场景9: 局部变量映射
// 通过局部变量间接赋值，需要追踪变量来源
// ============================================================================

// ProductDomain 产品领域模型
type ProductDomain struct {
	ID          uint64
	Name        string
	Description string
	Price       int
	Stock       int
}

// ProductPO 产品持久化模型
type ProductPO struct {
	ID          uint64 `gorm:"column:id;primaryKey"`
	Name        string `gorm:"column:name"`
	Description string `gorm:"column:description"`
	Price       int    `gorm:"column:price"`
	Stock       int    `gorm:"column:stock"`
}

// ToPO 局部变量映射示例
// 使用局部变量进行中间处理，需要追踪变量来源
func (p *ProductPO) ToPO(d *ProductDomain) *ProductPO {
	if d == nil {
		return nil
	}

	// 局部变量赋值
	name := d.Name
	desc := d.Description

	// 条件处理（不影响来源追踪）
	if len(name) > 100 {
		name = name[:100]
	}

	// 多级赋值
	price := d.Price
	finalPrice := price

	return &ProductPO{
		ID:          d.ID,
		Name:        name,       // 来源: d.Name
		Description: desc,       // 来源: d.Description
		Price:       finalPrice, // 来源: d.Price (通过 price -> finalPrice)
		Stock:       d.Stock,    // 直接赋值
	}
}

// ============================================================================
// 测试场景10: 局部变量 + JSON 映射
// 局部变量与 JSON 字段结合使用
// ============================================================================

// OrderInfo 订单信息
type OrderInfo struct {
	OrderNo    string `json:"order_no"`
	CustomerID uint64 `json:"customer_id"`
	Remark     string `json:"remark"`
}

// OrderDomain 订单领域模型
type OrderDomain struct {
	ID         uint64
	OrderNo    string
	CustomerID uint64
	Remark     string
	Amount     int
}

// OrderPO 订单持久化模型
type OrderPO struct {
	ID     uint64                        `gorm:"column:id;primaryKey"`
	Info   datatypes.JSONType[OrderInfo] `gorm:"column:info;type:json"`
	Amount int                           `gorm:"column:amount"`
}

// ToPO 局部变量 + JSON 映射示例
func (p *OrderPO) ToPO(d *OrderDomain) *OrderPO {
	if d == nil {
		return nil
	}

	// 局部变量
	orderNo := d.OrderNo
	custID := d.CustomerID
	remark := d.Remark

	return &OrderPO{
		ID: d.ID,
		Info: datatypes.NewJSONType(OrderInfo{
			OrderNo:    orderNo, // 来源: d.OrderNo
			CustomerID: custID,  // 来源: d.CustomerID
			Remark:     remark,  // 来源: d.Remark
		}),
		Amount: d.Amount,
	}
}

// ============================================================================
// 测试场景11: 方法调用映射（多对一）
// 调用 Domain 的方法，方法内部使用多个字段，映射到 PO 的一个字段
// ============================================================================

// CustomerDomain 客户领域模型
type CustomerDomain struct {
	ID       uint64
	Name     string
	Country  string
	Province string
	City     string
	Street   string
	ZipCode  string
}

// GetAddress 获取完整地址（组合多个字段）
func (c *CustomerDomain) GetAddress() string {
	return c.Country + " " + c.Province + " " + c.City + " " + c.Street
}

// GetFullAddress 获取带邮编的完整地址
func (c *CustomerDomain) GetFullAddress() string {
	return c.Country + " " + c.Province + " " + c.City + " " + c.Street + " " + c.ZipCode
}

// CustomerPO 客户持久化模型
type CustomerPO struct {
	ID      uint64 `gorm:"column:id;primaryKey"`
	Name    string `gorm:"column:name"`
	Address string `gorm:"column:address"` // 由 GetAddress() 方法生成
}

// ToPO 方法调用映射示例
// d.GetAddress() 内部使用了 Country, Province, City, Street 四个字段
func (p *CustomerPO) ToPO(d *CustomerDomain) *CustomerPO {
	if d == nil {
		return nil
	}
	return &CustomerPO{
		ID:      d.ID,
		Name:    d.Name,
		Address: d.GetAddress(), // 来源: d.Country, d.Province, d.City, d.Street
	}
}

// ============================================================================
// 测试场景12: 方法调用 + 局部变量
// 方法调用结果存入局部变量后再使用
// ============================================================================

// ShippingDomain 发货领域模型
type ShippingDomain struct {
	ID           uint64
	ReceiverName string
	Country      string
	Province     string
	City         string
	Detail       string
}

// GetShippingAddress 获取发货地址
func (s *ShippingDomain) GetShippingAddress() string {
	return s.Country + s.Province + s.City + s.Detail
}

// ShippingPO 发货持久化模型
type ShippingPO struct {
	ID           uint64 `gorm:"column:id;primaryKey"`
	ReceiverName string `gorm:"column:receiver_name"`
	Address      string `gorm:"column:address"`
}

// ToPO 方法调用 + 局部变量示例
func (p *ShippingPO) ToPO(d *ShippingDomain) *ShippingPO {
	if d == nil {
		return nil
	}

	// 方法调用结果存入局部变量
	addr := d.GetShippingAddress()

	return &ShippingPO{
		ID:           d.ID,
		ReceiverName: d.ReceiverName,
		Address:      addr, // 来源: d.Country, d.Province, d.City, d.Detail
	}
}

// ============================================================================
// 测试场景13: 缺失字段测试 (Missing Fields)
// ToPO 函数没有映射所有 PO 字段，用于验证 Missing fields 注释生成
// ============================================================================

// PartialUserDomain 部分用户领域模型
type PartialUserDomain struct {
	ID    uint64
	Name  string
	Email string
}

// PartialUserPO 部分用户持久化模型（有些字段在 ToPO 中未映射）
type PartialUserPO struct {
	ID        uint64 `gorm:"column:id;primaryKey"`
	Name      string `gorm:"column:name"`
	Email     string `gorm:"column:email"`
	DefaultID uint64 `gorm:"column:default_id"` // 未在 ToPO 中映射
	DeletedAt int64  `gorm:"column:deleted_at"` // 未在 ToPO 中映射
}

// ToPO 部分字段映射示例（故意缺少 DefaultID 和 DeletedAt）
func (p *PartialUserPO) ToPO(d *PartialUserDomain) *PartialUserPO {
	if d == nil {
		return nil
	}
	return &PartialUserPO{
		ID:    d.ID,
		Name:  d.Name,
		Email: d.Email,
		// DefaultID 和 DeletedAt 未映射
	}
}

// ============================================================================
// 测试场景14: 使用 gorm.io/gorm.Model 外部包嵌入类型
// 验证能正确解析第三方包中的结构体字段
// ============================================================================

// GormUserDomain 使用 gorm.Model 的用户领域模型
type GormUserDomain struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
	Username  string
	Email     string
}

// GormUserPO 使用 gorm.Model 的用户持久化模型
type GormUserPO struct {
	gorm.Model           // 嵌入 gorm.io/gorm.Model（包含 ID, CreatedAt, UpdatedAt, DeletedAt）
	Username   string    `gorm:"column:username"`
	Email      string    `gorm:"column:email"`
	LastLogin  time.Time `gorm:"column:last_login"` // 未在 ToPO 中映射，用于测试 Missing fields
}

// ToPO 使用 gorm.Model 的映射示例
func (p *GormUserPO) ToPO(d *GormUserDomain) *GormUserPO {
	if d == nil {
		return nil
	}
	return &GormUserPO{
		Model: gorm.Model{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
			DeletedAt: gorm.DeletedAt{Time: d.DeletedAt},
		},
		Username: d.Username,
		Email:    d.Email,
		// LastLogin 未映射
	}
}

// ============================================================================
// 测试场景15: JSON tag 与 Go 字段名不同
// 验证生成代码使用真实的 Go 字段名，而不是从 JSON tag 推断
// ============================================================================

// CustomTagInfo JSON 子结构（Go 字段名和 JSON tag 完全不同）
type CustomTagInfo struct {
	RealFieldA string `json:"custom_tag_a"` // Go: RealFieldA, JSON: custom_tag_a
	RealFieldB int    `json:"different_b"`  // Go: RealFieldB, JSON: different_b
	RealFieldC bool   `json:"another_c"`    // Go: RealFieldC, JSON: another_c
}

// NestedCustomInfo 嵌套结构（也有不同的命名）
type NestedCustomInfo struct {
	InnerName  string        `json:"inner_x"`  // Go: InnerName, JSON: inner_x
	InnerValue int           `json:"inner_y"`  // Go: InnerValue, JSON: inner_y
	SubInfo    CustomTagInfo `json:"sub_data"` // 嵌套的自定义结构
}

// CustomTagDomain 自定义标签领域模型
type CustomTagDomain struct {
	ID         uint64
	FieldA     string
	FieldB     int
	FieldC     bool
	InnerName  string
	InnerValue int
	SubFieldA  string
	SubFieldB  int
	SubFieldC  bool
}

// CustomTagPO 自定义标签持久化模型
type CustomTagPO struct {
	ID   uint64                               `gorm:"column:id;primaryKey"`
	Data datatypes.JSONType[NestedCustomInfo] `gorm:"column:data;type:json"`
}

// ToPO 自定义标签映射示例
// 验证生成代码使用 Go 字段名（如 RealFieldA）而不是 JSON tag（如 custom_tag_a）
func (p *CustomTagPO) ToPO(d *CustomTagDomain) *CustomTagPO {
	if d == nil {
		return nil
	}
	return &CustomTagPO{
		ID: d.ID,
		Data: datatypes.NewJSONType(NestedCustomInfo{
			InnerName:  d.InnerName,
			InnerValue: d.InnerValue,
			SubInfo: CustomTagInfo{
				RealFieldA: d.SubFieldA,
				RealFieldB: d.SubFieldB,
				RealFieldC: d.SubFieldC,
			},
		}),
	}
}

// ============================================================================
// 测试场景16: JSON 字段排序测试
// 验证生成的 JSON 字段按字母顺序排序
// ============================================================================

// SortTestInfo JSON 子结构（字段名故意乱序）
type SortTestInfo struct {
	Zebra  string `json:"zebra"`
	Apple  string `json:"apple"`
	Mango  string `json:"mango"`
	Banana string `json:"banana"`
}

// SortTestDomain 排序测试领域模型
type SortTestDomain struct {
	ID     uint64
	Zebra  string
	Apple  string
	Mango  string
	Banana string
}

// SortTestPO 排序测试持久化模型
type SortTestPO struct {
	ID   uint64                           `gorm:"column:id;primaryKey"`
	Info datatypes.JSONType[SortTestInfo] `gorm:"column:info;type:json"`
}

// ToPO 排序测试映射示例
// 注意：ToPO 中字段顺序是 Zebra, Apple, Mango, Banana（乱序）
// 但生成的 ToPatch 代码应该按字母顺序：Apple, Banana, Mango, Zebra
func (p *SortTestPO) ToPO(d *SortTestDomain) *SortTestPO {
	if d == nil {
		return nil
	}
	return &SortTestPO{
		ID: d.ID,
		Info: datatypes.NewJSONType(SortTestInfo{
			Zebra:  d.Zebra, // 故意放在第一个
			Apple:  d.Apple, // 应该排序后在第一个
			Mango:  d.Mango,
			Banana: d.Banana,
		}),
	}
}

// ============================================================================
// 测试场景17: 指针解引用测试
// 验证能正确解析 *d.Field 和 &d.Field 的情况
// ============================================================================

// PointerDomain 指针测试领域模型
type PointerDomain struct {
	ID          uint64
	Name        *string // 指针字段
	Age         *int    // 指针字段
	Score       int     // 普通字段
	TokenSupply *uint64 // 指针字段（模拟用户场景）
	MaxAmount   *int64  // 指针字段
}

// PointerPO 指针测试持久化模型
type PointerPO struct {
	ID          uint64 `gorm:"column:id;primaryKey"`
	Name        string `gorm:"column:name"`
	Age         int    `gorm:"column:age"`
	Score       int    `gorm:"column:score"`
	TokenSupply uint64 `gorm:"column:token_supply"`
	MaxAmount   int64  `gorm:"column:max_amount"`
}

// ToPO 指针解引用映射示例
// 验证能正确处理 *entity.Field 和 &entity.Field 的情况
func (p *PointerPO) ToPO(entity *PointerDomain) *PointerPO {
	if entity == nil {
		return nil
	}
	return &PointerPO{
		ID:          entity.ID,
		Name:        *entity.Name,        // 指针解引用
		Age:         *entity.Age,         // 指针解引用
		Score:       entity.Score,        // 普通字段
		TokenSupply: *entity.TokenSupply, // 指针解引用（模拟用户场景）
		MaxAmount:   *entity.MaxAmount,   // 指针解引用
	}
}

// ============================================================================
// 测试场景18: JSONSlice + lo.Map 映射（一对一）
// datatypes.NewJSONSlice(lo.Map(entity.Field, func...)) 模式
// ============================================================================

// ExchangeRule 交易规则
type ExchangeRule struct {
	TokenID string `json:"token_id"`
	Rate    int64  `json:"rate"`
}

// JSONSliceDomain 领域模型
type JSONSliceDomain struct {
	ID            uint64
	Name          string
	ExchangeRules []ExchangeRule // 会通过 lo.Map 转换
	Tags          []string       // 普通切片
}

// JSONSlicePO 持久化模型
type JSONSlicePO struct {
	ID            uint64                            `gorm:"column:id;primaryKey"`
	Name          string                            `gorm:"column:name"`
	ExchangeRules datatypes.JSONSlice[ExchangeRule] `gorm:"column:exchange_rules;type:json"`
	Tags          datatypes.JSONSlice[string]       `gorm:"column:tags;type:json"`
}

// ToPO JSONSlice + lo.Map 映射示例
// 验证 datatypes.NewJSONSlice(lo.Map(...)) 被识别为一对一映射
func (p *JSONSlicePO) ToPO(entity *JSONSliceDomain) *JSONSlicePO {
	if entity == nil {
		return nil
	}
	return &JSONSlicePO{
		ID:   entity.ID,
		Name: entity.Name,
		ExchangeRules: datatypes.NewJSONSlice(lo.Map(entity.ExchangeRules, func(item ExchangeRule, _ int) ExchangeRule {
			return ExchangeRule{
				TokenID: item.TokenID,
				Rate:    item.Rate,
			}
		})),
		Tags: datatypes.NewJSONSlice(entity.Tags),
	}
}

// ============================================================================
// 测试场景19: JSONSlice + lo.Map + 方法调用映射
// datatypes.NewJSONSlice(lo.Map(entity.GetMethod(), func...)) 模式
// 方法内部访问的字段都被视为映射关系
// ============================================================================

// RuleItem 规则项
type RuleItem struct {
	Key   string `json:"key"`
	Value int    `json:"value"`
}

// JSONSliceMethodDomain 领域模型（带方法）
type JSONSliceMethodDomain struct {
	ID        uint64
	Name      string
	RuleKey   string // 被 GetRules 方法使用
	RuleValue int    // 被 GetRules 方法使用
	Extra     string // 被 GetRules 方法使用
}

// GetRules 获取规则列表（内部使用多个字段）
func (d *JSONSliceMethodDomain) GetRules() []RuleItem {
	return []RuleItem{
		{Key: d.RuleKey, Value: d.RuleValue},
	}
}

// JSONSliceMethodPO 持久化模型
type JSONSliceMethodPO struct {
	ID    uint64                        `gorm:"column:id;primaryKey"`
	Name  string                        `gorm:"column:name"`
	Rules datatypes.JSONSlice[RuleItem] `gorm:"column:rules;type:json"`
}

// ToPO JSONSlice + lo.Map + 方法调用映射示例
// 验证 datatypes.NewJSONSlice(lo.Map(entity.GetRules(), ...)) 能识别方法内部使用的字段
func (p *JSONSliceMethodPO) ToPO(entity *JSONSliceMethodDomain) *JSONSliceMethodPO {
	if entity == nil {
		return nil
	}
	return &JSONSliceMethodPO{
		ID:   entity.ID,
		Name: entity.Name,
		Rules: datatypes.NewJSONSlice(lo.Map(entity.GetRules(), func(item RuleItem, _ int) RuleItem {
			return RuleItem{
				Key:   item.Key,
				Value: item.Value,
			}
		})),
	}
}

// ============================================================================
// 测试场景20: 外部包方法调用推断（模拟跨包方法调用）
// 当方法定义在外部包时，从方法名推断字段名
// GetExchangeRules -> ExchangeRules
// ============================================================================

// ExternalMethodDomain 模拟外部包的 Domain（方法定义在"外部"）
// 注意：这里我们不定义 GetItems 方法，模拟方法在外部包的情况
type ExternalMethodDomain struct {
	ID    uint64
	Name  string
	Items []string // 假设有 GetItems() 方法返回此字段
}

// ExternalMethodPO 持久化模型
type ExternalMethodPO struct {
	ID    uint64                      `gorm:"column:id;primaryKey"`
	Name  string                      `gorm:"column:name"`
	Items datatypes.JSONSlice[string] `gorm:"column:items;type:json"`
}

// 注意：这个 ToPO 调用了 GetItems()，但该方法未在本包定义
// automap 应该从方法名 GetItems 推断出字段名 Items
func (p *ExternalMethodPO) ToPO(entity *ExternalMethodDomain) *ExternalMethodPO {
	if entity == nil {
		return nil
	}
	return &ExternalMethodPO{
		ID:   entity.ID,
		Name: entity.Name,
		Items: datatypes.NewJSONSlice(lo.Map(entity.GetItems(), func(item string, _ int) string {
			return item
		})),
	}
}

// GetItems 这个方法定义在这里，但为了测试，我们假设它不存在
// 实际测试中，automap 应该能处理方法不在本包的情况
func (d *ExternalMethodDomain) GetItems() []string {
	return d.Items
}

// ============================================================================
// 测试场景21: 字段顺序测试 (Field Ordering)
// 验证生成的 ToPatch 方法字段顺序与 PO 结构体定义顺序一致
// ============================================================================

// TokenDetail Token详情（一对多映射用）
type TokenDetail struct {
	TokenAddress  string
	TokenName     string
	TokenSymbol   string
	TokenDecimals int
}

// FieldOrderDomain 字段顺序测试领域模型
type FieldOrderDomain struct {
	ID           uint64
	Name         string
	Token        TokenDetail // 一对多：展开为多个字段
	Status       int
	FailedReason string
	CreatedAt    time.Time
}

// FieldOrderPO 字段顺序测试持久化模型
// 注意字段顺序：ID, Name, TokenAddress, TokenName, TokenSymbol, TokenDecimals, Status, FailedReason, CreatedAt
// ToPO 中赋值顺序可能与此不同，但生成的代码应该按此顺序
type FieldOrderPO struct {
	ID            uint64 `gorm:"column:id;primaryKey"`
	Name          string `gorm:"column:name"`
	TokenAddress  string `gorm:"column:token_address"`  // Token.TokenAddress
	TokenName     string `gorm:"column:token_name"`     // Token.TokenName
	TokenSymbol   string `gorm:"column:token_symbol"`   // Token.TokenSymbol
	TokenDecimals int    `gorm:"column:token_decimals"` // Token.TokenDecimals
	Status        int    `gorm:"column:status"`
	FailedReason  string `gorm:"column:failed_reason"`
	CreatedAt     int64  `gorm:"column:created_at"` // time.Time -> int64
}

// ToPO 字段顺序测试映射示例
// 注意：这里故意按不同于 PO 结构体的顺序赋值
// Token 字段在 Status 之后赋值，但在 PO 中 Token* 字段在 Status 之前
// 生成的 ToPatch 代码应该按 PO 结构体顺序输出
func (p *FieldOrderPO) ToPO(d *FieldOrderDomain) *FieldOrderPO {
	if d == nil {
		return nil
	}
	return &FieldOrderPO{
		ID:            d.ID,
		Name:          d.Name,
		Status:        d.Status,       // 故意放在 Token 之前
		FailedReason:  d.FailedReason, // 故意放在 Token 之前
		CreatedAt:     d.CreatedAt.Unix(),
		TokenAddress:  d.Token.TokenAddress, // Token 字段放在最后
		TokenName:     d.Token.TokenName,
		TokenSymbol:   d.Token.TokenSymbol,
		TokenDecimals: d.Token.TokenDecimals,
	}
}

// ============================================================================
// 测试场景22: EmbeddedOneToMany 映射 (无前缀)
// 一个输入字段直接映射到嵌入结构体（展开为多个列）
// ============================================================================

// AccountIDColumns 账户ID列（用于嵌入）
type AccountIDColumns struct {
	Namespace string `gorm:"column:namespace"`
	Reference string `gorm:"column:reference"`
	Address   string `gorm:"column:address"`
}

// EmbeddedOneToManyDomain 领域模型
type EmbeddedOneToManyDomain struct {
	ID      uint64
	Name    string
	Account AccountIDColumns // 直接赋值给嵌入字段
}

// EmbeddedOneToManyPO 持久化模型
type EmbeddedOneToManyPO struct {
	ID      uint64           `gorm:"column:id;primaryKey"`
	Name    string           `gorm:"column:name"`
	Account AccountIDColumns `gorm:"embedded"` // 嵌入字段，展开为 namespace, reference, address
}

// ToPO EmbeddedOneToMany 映射示例（无前缀）
// Account: d.Account 直接赋值，映射到多个列
func (p *EmbeddedOneToManyPO) ToPO(d *EmbeddedOneToManyDomain) *EmbeddedOneToManyPO {
	if d == nil {
		return nil
	}
	return &EmbeddedOneToManyPO{
		ID:      d.ID,
		Name:    d.Name,
		Account: d.Account, // 直接赋值
	}
}

// ============================================================================
// 测试场景23: EmbeddedOneToMany 映射 (带前缀)
// 一个输入字段直接映射到嵌入结构体，列名带前缀
// ============================================================================

// EmbeddedPrefixDomain 领域模型
type EmbeddedPrefixDomain struct {
	ID      uint64
	Title   string
	Account AccountIDColumns // 直接赋值给嵌入字段
}

// EmbeddedPrefixPO 持久化模型
type EmbeddedPrefixPO struct {
	ID      uint64           `gorm:"column:id;primaryKey"`
	Title   string           `gorm:"column:title"`
	Account AccountIDColumns `gorm:"embedded;embeddedPrefix:acc_"` // 嵌入字段带前缀
}

// ToPO EmbeddedOneToMany 映射示例（带前缀）
// 生成的列名应该是 acc_namespace, acc_reference, acc_address
func (p *EmbeddedPrefixPO) ToPO(d *EmbeddedPrefixDomain) *EmbeddedPrefixPO {
	if d == nil {
		return nil
	}
	return &EmbeddedPrefixPO{
		ID:      d.ID,
		Title:   d.Title,
		Account: d.Account, // 直接赋值
	}
}

// ============================================================================
// 测试场景24: EmbeddedOneToMany 映射 (外部包类型)
// 使用外部包 caip10.AccountIDColumnsCompact 作为嵌入字段类型
// 验证 structparse 能正确解析外部包的结构体字段
// ============================================================================

// ExternalEmbeddedDomain 领域模型（使用接口类型）
type ExternalEmbeddedDomain struct {
	ID      uint64
	Name    string
	Account caip10.AccountID // 接口类型，有 ToColumnsCompact() 方法
}

// ExternalEmbeddedPO 持久化模型（使用外部包嵌入类型）
type ExternalEmbeddedPO struct {
	ID      uint64                         `gorm:"column:id;primaryKey"`
	Name    string                         `gorm:"column:name"`
	Account caip10.AccountIDColumnsCompact `gorm:"embedded;embeddedPrefix:account_"` // 外部包类型，展开为 account_chain_id, account_address
}

// ToPO 外部包 EmbeddedOneToMany 映射示例
// Account: entity.Account.ToColumnsCompact() 调用接口方法返回外部包类型
// 生成的列名应该是 account_chain_id, account_address
func (p *ExternalEmbeddedPO) ToPO(entity *ExternalEmbeddedDomain) *ExternalEmbeddedPO {
	if entity == nil {
		return nil
	}
	return &ExternalEmbeddedPO{
		ID:      entity.ID,
		Name:    entity.Name,
		Account: entity.Account.ToColumnsCompact(), // 方法调用返回外部包类型
	}
}

// ============================================================================
// 测试场景25: EmbeddedOneToMany 映射 (外部包类型，无前缀)
// 使用外部包 caip10.AccountIDColumnsCompact 作为嵌入字段类型，但没有 embeddedPrefix
// 验证不会错误地包含其他嵌入类型（如 gorm.Model）的字段
// ============================================================================

// ExternalNoPrefixDomain 领域模型（使用接口类型）
type ExternalNoPrefixDomain struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
	Name      string
	Account   caip10.AccountID // 接口类型，有 ToColumnsCompact() 方法
}

// ExternalNoPrefixPO 持久化模型（外部包嵌入类型，无前缀）
// 同时包含 gorm.Model 和 caip10.AccountIDColumnsCompact 两个嵌入类型
// 验证生成的代码只包含对应嵌入类型的字段
type ExternalNoPrefixPO struct {
	gorm.Model                                // 嵌入 gorm.Model（包含 ID, CreatedAt, UpdatedAt, DeletedAt）
	Name       string                         `gorm:"column:name"`
	Account    caip10.AccountIDColumnsCompact `gorm:"embedded"` // 外部包类型，无前缀，展开为 chain_id, address
}

func (p *ExternalNoPrefixPO) ToPO2(entity *ExternalNoPrefixDomain) *ExternalNoPrefixPO {
	return &ExternalNoPrefixPO{}
}

// ToPO 外部包 EmbeddedOneToMany 映射示例（无前缀）
// 关键测试：Account 的映射只应该包含 chain_id 和 address
// 不应该包含 gorm.Model 的字段（id, created_at, updated_at, deleted_at）
func (p *ExternalNoPrefixPO) ToPO(entity *ExternalNoPrefixDomain) *ExternalNoPrefixPO {
	if entity == nil {
		return nil
	}
	return &ExternalNoPrefixPO{
		Model: gorm.Model{
			ID:        entity.ID,
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
			DeletedAt: gorm.DeletedAt{Time: entity.DeletedAt},
		},
		Name:    entity.Name,
		Account: entity.Account.ToColumnsCompact(), // 方法调用返回外部包类型
	}
}
