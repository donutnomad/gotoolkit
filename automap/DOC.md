# AutoMap 映射识别文档

AutoMap 用于识别 ToPO 函数中 Domain 到 PO 的映射关系，支持多种映射类型的自动识别。

## 映射类型

| 类型 | 说明 | 示例 |
|------|------|------|
| `OneToOne` | 一对一映射 | `Domain.Name → PO.Name → column:name` |
| `OneToMany` | 一对多映射（Domain 结构体字段 → 多个数据库列） | `Domain.Location → PO.Country, Province, City` |
| `ManyToOne` | 多对一映射（多个 Domain 字段 → JSON 列） | `Domain.Phone, Address → PO.Contact (JSON)` |
| `Embedded` | GORM 嵌入映射 | `PO.Model → id, created_at, updated_at` |
| `MethodCall` | 方法调用映射（分析方法体使用的字段） | `Domain.GetAddress() → PO.Address` |

## API 使用

```go
result, err := automap.Parse("path/to/file.go", "UserPO", "ToPO")
if err != nil {
    // 处理错误
}

for _, group := range result.Groups {
    fmt.Printf("映射类型: %s\n", group.Type)
    for _, m := range group.Mappings {
        fmt.Printf("  %s → %s (column: %s)\n", m.SourcePath, m.TargetPath, m.ColumnName)
    }
}
```

## 测试场景

### 场景1: 一对一映射 (OneToOne)

Domain 字段直接映射到 PO 字段。

```go
// Domain
type SimpleUserDomain struct {
    ID    uint64
    Name  string
    Email string
    Age   int
}

// PO
type SimpleUserPO struct {
    ID    uint64 `gorm:"column:id;primaryKey"`
    Name  string `gorm:"column:name"`
    Email string `gorm:"column:email"`
    Age   int    `gorm:"column:age"`
}

// ToPO
func (p *SimpleUserPO) ToPO(d *SimpleUserDomain) *SimpleUserPO {
    return &SimpleUserPO{
        ID:    d.ID,
        Name:  d.Name,
        Email: d.Email,
        Age:   d.Age,
    }
}
```

**识别结果:**
| SourcePath | TargetPath | ColumnName |
|------------|------------|------------|
| ID | ID | id |
| Name | Name | name |
| Email | Email | email |
| Age | Age | age |

---

### 场景2: GORM Embedded (嵌入字段)

PO 中的嵌入字段在数据库中对应多个独立的 column。

```go
// 嵌入模型
type Model struct {
    ID        uint64    `gorm:"column:id;primaryKey"`
    CreatedAt time.Time `gorm:"column:created_at"`
    UpdatedAt time.Time `gorm:"column:updated_at"`
}

// Domain
type EmbeddedUserDomain struct {
    ID        uint64
    CreatedAt time.Time
    UpdatedAt time.Time
    Name      string
    Status    int
}

// PO
type EmbeddedUserPO struct {
    Model        // gorm embedded
    Name   string `gorm:"column:name"`
    Status int    `gorm:"column:status"`
}

// ToPO
func (p *EmbeddedUserPO) ToPO(d *EmbeddedUserDomain) *EmbeddedUserPO {
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
```

**识别结果:**
| Type | SourcePath | TargetPath | ColumnName |
|------|------------|------------|------------|
| Embedded | ID | Model.ID | id |
| Embedded | CreatedAt | Model.CreatedAt | created_at |
| Embedded | UpdatedAt | Model.UpdatedAt | updated_at |
| OneToOne | Name | Name | name |
| OneToOne | Status | Status | status |

---

### 场景3: 多对一映射 - JSON字段 (ManyToOne)

Domain 中的多个字段存储为数据库的一个 JSON 字段。

```go
// JSON 子结构
type ContactInfo struct {
    Phone   string `json:"phone"`
    Address string `json:"address"`
    City    string `json:"city"`
}

// Domain
type ProfileDomain struct {
    ID      uint64
    Name    string
    Phone   string  // 合并到 JSON
    Address string  // 合并到 JSON
    City    string  // 合并到 JSON
    Score   int
}

// PO
type ProfilePO struct {
    ID      uint64                          `gorm:"column:id;primaryKey"`
    Name    string                          `gorm:"column:name"`
    Contact datatypes.JSONType[ContactInfo] `gorm:"column:contact;type:json"`
    Score   int                             `gorm:"column:score"`
}

// ToPO
func (p *ProfilePO) ToPO(d *ProfileDomain) *ProfilePO {
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
```

**识别结果:**
| Type | SourcePath | TargetPath | ColumnName | JSONPath |
|------|------------|------------|------------|----------|
| OneToOne | ID | ID | id | |
| OneToOne | Name | Name | name | |
| ManyToOne | Phone | Contact | contact | phone |
| ManyToOne | Address | Contact | contact | address |
| ManyToOne | City | Contact | contact | city |
| OneToOne | Score | Score | score | |

---

### 场景4: 一对多映射 (OneToMany)

Domain 中的一个结构体字段映射到数据库的多个独立列。

```go
// Domain 嵌套结构
type LocationInfo struct {
    Country  string
    Province string
    City     string
    District string
}

// Domain
type CompanyDomain struct {
    ID       uint64
    Name     string
    Location LocationInfo  // 一个结构体字段
}

// PO
type CompanyPO struct {
    ID       uint64 `gorm:"column:id;primaryKey"`
    Name     string `gorm:"column:name"`
    Country  string `gorm:"column:country"`
    Province string `gorm:"column:province"`
    City     string `gorm:"column:city"`
    District string `gorm:"column:district"`
}

// ToPO
func (p *CompanyPO) ToPO(d *CompanyDomain) *CompanyPO {
    return &CompanyPO{
        ID:       d.ID,
        Name:     d.Name,
        Country:  d.Location.Country,
        Province: d.Location.Province,
        City:     d.Location.City,
        District: d.Location.District,
    }
}
```

**识别结果:**
| Type | SourcePath | TargetPath | ColumnName |
|------|------------|------------|------------|
| OneToOne | ID | ID | id |
| OneToOne | Name | Name | name |
| OneToMany | Location.Country | Country | country |
| OneToMany | Location.Province | Province | province |
| OneToMany | Location.City | City | city |
| OneToMany | Location.District | District | district |

---

### 场景5: GORM Embedded with Prefix (带前缀的嵌入)

嵌入字段使用 `embeddedPrefix` 指定列名前缀。

```go
// 嵌入结构
type Audit struct {
    CreatedBy string    `gorm:"column:created_by"`
    CreatedAt time.Time `gorm:"column:created_at"`
    UpdatedBy string    `gorm:"column:updated_by"`
    UpdatedAt time.Time `gorm:"column:updated_at"`
}

// Domain
type AuditDomain struct {
    ID        uint64
    Title     string
    CreatedBy string
    CreatedAt time.Time
    UpdatedBy string
    UpdatedAt time.Time
}

// PO
type AuditPO struct {
    ID    uint64 `gorm:"column:id;primaryKey"`
    Title string `gorm:"column:title"`
    Audit Audit  `gorm:"embedded;embeddedPrefix:audit_"`  // 带前缀
}

// ToPO
func (p *AuditPO) ToPO(d *AuditDomain) *AuditPO {
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
```

**识别结果:**
| Type | SourcePath | TargetPath | ColumnName |
|------|------------|------------|------------|
| OneToOne | ID | ID | id |
| OneToOne | Title | Title | title |
| Embedded | CreatedBy | Audit.CreatedBy | audit_created_by |
| Embedded | CreatedAt | Audit.CreatedAt | audit_created_at |
| Embedded | UpdatedBy | Audit.UpdatedBy | audit_updated_by |
| Embedded | UpdatedAt | Audit.UpdatedAt | audit_updated_at |

---

### 场景6: 复杂嵌套 JSON (多层嵌套)

JSON 字段内部包含嵌套结构。

```go
// JSON 子结构
type AuthorInfo struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

type Metadata struct {
    Tags     []string   `json:"tags"`
    Author   AuthorInfo `json:"author"`
    Version  string     `json:"version"`
    Priority int        `json:"priority"`
}

// Domain
type ArticleDomain struct {
    ID          uint64
    Title       string
    Content     string
    Tags        []string
    AuthorName  string   // -> Metadata.Author.Name
    AuthorEmail string   // -> Metadata.Author.Email
    Version     string
    Priority    int
    ViewCount   int
}

// PO
type ArticlePO struct {
    ID        uint64                       `gorm:"column:id;primaryKey"`
    Title     string                       `gorm:"column:title"`
    Content   string                       `gorm:"column:content"`
    Metadata  datatypes.JSONType[Metadata] `gorm:"column:metadata;type:json"`
    ViewCount int                          `gorm:"column:view_count"`
}

// ToPO
func (p *ArticlePO) ToPO(d *ArticleDomain) *ArticlePO {
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
```

**识别结果:**
| Type | SourcePath | TargetPath | ColumnName | JSONPath |
|------|------------|------------|------------|----------|
| OneToOne | ID | ID | id | |
| OneToOne | Title | Title | title | |
| OneToOne | Content | Content | content | |
| ManyToOne | Tags | Metadata | metadata | tags |
| ManyToOne | AuthorName | Metadata | metadata | author.name |
| ManyToOne | AuthorEmail | Metadata | metadata | author.email |
| ManyToOne | Version | Metadata | metadata | version |
| ManyToOne | Priority | Metadata | metadata | priority |
| OneToOne | ViewCount | ViewCount | view_count | |

---

### 场景7: 类型转换映射

字段存在类型转换，如 `time.Time` 转 `int64`。

```go
// Domain
type TimestampDomain struct {
    ID         uint64
    Name       string
    CreateTime time.Time
    UpdateTime time.Time
    ExpireTime time.Time
}

// PO
type TimestampPO struct {
    ID         uint64 `gorm:"column:id;primaryKey"`
    Name       string `gorm:"column:name"`
    CreateTime int64  `gorm:"column:create_time"`
    UpdateTime int64  `gorm:"column:update_time"`
    ExpireTime int64  `gorm:"column:expire_time"`
}

// ToPO
func (p *TimestampPO) ToPO(d *TimestampDomain) *TimestampPO {
    return &TimestampPO{
        ID:         d.ID,
        Name:       d.Name,
        CreateTime: d.CreateTime.Unix(),
        UpdateTime: d.UpdateTime.Unix(),
        ExpireTime: d.ExpireTime.Unix(),
    }
}
```

**识别结果:**
| SourcePath | TargetPath | ColumnName | ConvertExpr |
|------------|------------|------------|-------------|
| ID | ID | id | |
| Name | Name | name | |
| CreateTime | CreateTime | create_time | .Unix() |
| UpdateTime | UpdateTime | update_time | .Unix() |
| ExpireTime | ExpireTime | expire_time | .Unix() |

---

### 场景8: 混合映射（综合场景）

同时包含 Embedded、OneToOne、ManyToOne 和类型转换。

```go
// JSON 子结构
type Settings struct {
    Theme    string `json:"theme"`
    Language string `json:"language"`
    Timezone string `json:"timezone"`
}

// Domain
type AccountDomain struct {
    ID        uint64
    CreatedAt time.Time
    UpdatedAt time.Time
    Username  string
    Email     string
    Theme     string
    Language  string
    Timezone  string
    Status    int
    LastLogin time.Time
}

// PO
type AccountPO struct {
    Model                                  // embedded
    Username  string                       `gorm:"column:username"`
    Email     string                       `gorm:"column:email"`
    Settings  datatypes.JSONType[Settings] `gorm:"column:settings;type:json"`
    Status    int                          `gorm:"column:status"`
    LastLogin int64                        `gorm:"column:last_login"`
}

// ToPO
func (p *AccountPO) ToPO(d *AccountDomain) *AccountPO {
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
```

**识别结果:**
| Type | SourcePath | TargetPath | ColumnName | JSONPath | ConvertExpr |
|------|------------|------------|------------|----------|-------------|
| Embedded | ID | Model.ID | id | | |
| Embedded | CreatedAt | Model.CreatedAt | created_at | | |
| Embedded | UpdatedAt | Model.UpdatedAt | updated_at | | |
| OneToOne | Username | Username | username | | |
| OneToOne | Email | Email | email | | |
| OneToOne | Status | Status | status | | |
| OneToOne | LastLogin | LastLogin | last_login | | .Unix() |
| ManyToOne | Theme | Settings | settings | theme | |
| ManyToOne | Language | Settings | settings | language | |
| ManyToOne | Timezone | Settings | settings | timezone | |

---

### 场景9: 局部变量映射

通过局部变量间接赋值，能够追踪变量来源。

```go
// Domain
type ProductDomain struct {
    ID          uint64
    Name        string
    Description string
    Price       int
    Stock       int
}

// PO
type ProductPO struct {
    ID          uint64 `gorm:"column:id;primaryKey"`
    Name        string `gorm:"column:name"`
    Description string `gorm:"column:description"`
    Price       int    `gorm:"column:price"`
    Stock       int    `gorm:"column:stock"`
}

// ToPO
func (p *ProductPO) ToPO(d *ProductDomain) *ProductPO {
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
        Stock:       d.Stock,
    }
}
```

**识别结果:**
| SourcePath | TargetPath | ColumnName |
|------------|------------|------------|
| ID | ID | id |
| Name | Name | name |
| Description | Description | description |
| Price | Price | price |
| Stock | Stock | stock |

**局部变量追踪:**
- `name` → `d.Name`
- `desc` → `d.Description`
- `price` → `d.Price`
- `finalPrice` → `d.Price` (通过 `price`)

---

### 场景10: 局部变量 + JSON 映射

局部变量与 JSON 字段结合使用。

```go
// JSON 子结构
type OrderInfo struct {
    OrderNo    string `json:"order_no"`
    CustomerID uint64 `json:"customer_id"`
    Remark     string `json:"remark"`
}

// Domain
type OrderDomain struct {
    ID         uint64
    OrderNo    string
    CustomerID uint64
    Remark     string
    Amount     int
}

// PO
type OrderPO struct {
    ID     uint64                        `gorm:"column:id;primaryKey"`
    Info   datatypes.JSONType[OrderInfo] `gorm:"column:info;type:json"`
    Amount int                           `gorm:"column:amount"`
}

// ToPO
func (p *OrderPO) ToPO(d *OrderDomain) *OrderPO {
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
```

**识别结果:**
| Type | SourcePath | TargetPath | ColumnName | JSONPath |
|------|------------|------------|------------|----------|
| OneToOne | ID | ID | id | |
| OneToOne | Amount | Amount | amount | |
| ManyToOne | OrderNo | Info | info | order_no |
| ManyToOne | CustomerID | Info | info | customer_id |
| ManyToOne | Remark | Info | info | remark |

---

### 场景11: 方法调用映射 (MethodCall)

Domain 有方法，ToPO 中调用该方法获取值。需要进入方法体分析使用的字段。

```go
// Domain
type CustomerDomain struct {
    ID       uint64
    Name     string
    Country  string
    Province string
    City     string
    Street   string
}

// Domain 方法
func (c *CustomerDomain) GetAddress() string {
    return c.Country + " " + c.Province + " " + c.City + " " + c.Street
}

// PO
type CustomerPO struct {
    ID      uint64 `gorm:"column:id;primaryKey"`
    Name    string `gorm:"column:name"`
    Address string `gorm:"column:address"`
}

// ToPO
func (p *CustomerPO) ToPO(d *CustomerDomain) *CustomerPO {
    return &CustomerPO{
        ID:      d.ID,
        Name:    d.Name,
        Address: d.GetAddress(), // 调用方法
    }
}
```

**识别结果:**
| Type | SourcePath | TargetPath | ColumnName | MethodName |
|------|------------|------------|------------|------------|
| OneToOne | ID | ID | id | |
| OneToOne | Name | Name | name | |
| MethodCall | City | Address | address | GetAddress |
| MethodCall | Country | Address | address | GetAddress |
| MethodCall | Province | Address | address | GetAddress |
| MethodCall | Street | Address | address | GetAddress |

**说明:**
- 方法调用映射会分析方法体内使用的字段
- 使用的字段按字母顺序排列以保证稳定性
- MethodName 记录被调用的方法名称

---

### 场景12: 方法调用 + 局部变量映射

方法调用的结果先赋值给局部变量，再使用该变量。

```go
// Domain
type ShippingDomain struct {
    ID           uint64
    ReceiverName string
    Country      string
    Province     string
    City         string
    Detail       string
}

// Domain 方法
func (s *ShippingDomain) GetShippingAddress() string {
    return s.Country + " " + s.Province + " " + s.City + " " + s.Detail
}

// PO
type ShippingPO struct {
    ID           uint64 `gorm:"column:id;primaryKey"`
    ReceiverName string `gorm:"column:receiver_name"`
    Address      string `gorm:"column:address"`
}

// ToPO
func (p *ShippingPO) ToPO(d *ShippingDomain) *ShippingPO {
    // 先赋值给局部变量
    addr := d.GetShippingAddress()

    return &ShippingPO{
        ID:           d.ID,
        ReceiverName: d.ReceiverName,
        Address:      addr, // 使用局部变量
    }
}
```

**识别结果:**
| Type | SourcePath | TargetPath | ColumnName | MethodName |
|------|------------|------------|------------|------------|
| OneToOne | ID | ID | id | |
| OneToOne | ReceiverName | ReceiverName | receiver_name | |
| MethodCall | City | Address | address | GetShippingAddress |
| MethodCall | Country | Address | address | GetShippingAddress |
| MethodCall | Detail | Address | address | GetShippingAddress |
| MethodCall | Province | Address | address | GetShippingAddress |

**说明:**
- 支持方法调用结果通过局部变量传递
- 变量追踪会识别方法调用并记录方法信息

---

## 实现原理

### 1. AST 解析

使用 Go 的 `go/ast` 包解析源文件，提取函数定义和结构体类型。

### 2. 变量追踪

在分析 return 语句之前，先扫描函数体中的所有赋值语句，构建变量映射表：

```go
varMap map[string]string  // 变量名 -> 源路径
```

支持多级追踪：
```go
price := d.Price      // varMap["price"] = "Price"
finalPrice := price   // varMap["finalPrice"] = "Price"
```

### 3. 映射识别

分析 return 语句中的结构体字面量，识别各种映射模式：

- **SelectorExpr**: `d.Field` → 直接字段访问
- **CallExpr**: `d.Field.Unix()` → 方法调用（类型转换）
- **Ident**: `varName` → 局部变量（查表追踪）
- **CompositeLit**: `Type{...}` → 嵌套结构体

### 4. GORM 标签解析

从结构体字段的 tag 中提取数据库列名：

```go
`gorm:"column:user_name"`           → column: user_name
`gorm:"embedded;embeddedPrefix:a_"` → embedded with prefix
```

### 5. JSON 标签解析

从 JSON 结构体字段的 tag 中提取 JSON 路径：

```go
`json:"order_no"` → JSONPath: order_no
```

### 6. 方法调用分析

当检测到 `d.MethodName()` 形式的调用时：

1. 从 `methodDecls` 缓存中查找方法定义
2. 遍历方法体 AST，提取所有 `recv.Field` 形式的字段访问
3. 按字母顺序排列字段以保证稳定性
4. 生成 `MethodCall` 类型的映射组

```go
// 方法定义缓存
methodDecls map[string]map[string]*ast.FuncDecl  // receiverType -> methodName -> funcDecl

// 方法调用信息
methodCallMap map[string]methodCallInfo  // varName -> method call info
```

支持两种使用方式：
```go
// 直接调用
Address: d.GetAddress()

// 通过局部变量
addr := d.GetAddress()
Address: addr
```
