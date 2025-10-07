# GORM Query Generator

GORM查询代码生成器,基于结构体定义生成类型安全的查询辅助代码。

## 功能特性

- 支持多个结构体批量生成
- 自动推导表名(支持`TableName()`方法和默认规则)
- 自动解析gorm标签中的列名
- 支持字段类型映射:
  - 字符串/文本类型 → `field.Pattern` (支持LIKE操作)
  - 数值/日期等其他类型 → `field.Comparable` (支持比较操作)
  - JSON类型自动忽略
- 自动展开嵌入的`gorm.Model`字段
- 生成带泛型的表结构和As方法

## 安装

```bash
go build -o gormgen ./gormgen
```

## 使用方法

```bash
./gormgen -dir <目录路径> -struct <结构体名称> -prefix <前缀>
```

### 参数说明

- `-dir`: 要扫描的目录路径,默认为当前目录
- `-struct`: 结构体名称,多个使用逗号分隔,例如: `User,Book,Order`
- `-prefix`: 生成的结构体前缀,默认为`T`

### 示例

```bash
# 为User结构体生成查询代码,使用L作为前缀
./gormgen -dir . -struct User -prefix L

# 为多个结构体生成查询代码
./gormgen -dir ./models -struct User,Book,Order -prefix T
```

## 生成的代码示例

输入结构体:

```go
type User struct {
    gorm.Model
    Name string
    Age  int32
}
```

生成的代码:

```go
type LUserTable[T any] struct {
    ID        *field.Comparable[uint]
    CreatedAt *field.Comparable[time.Time]
    UpdatedAt *field.Comparable[time.Time]
    DeletedAt *field.Comparable[gorm.DeletedAt]
    Name      *field.Pattern[string]
    Age       *field.Comparable[int32]
    alias     string
    tableName string
}

func (t LUserTable[T]) TableName() string {
    return field.AS(t.tableName, t.alias)
}

func (t LUserTable[T]) As(alias string) LUserTable[T] {
    var ret = LUserTable[T]{
        alias:     alias,
        tableName: t.tableName,
    }
    tableName := gsql.TableName(alias)
    ret.ID = ret.ID.WithTable(&tableName)
    ret.CreatedAt = ret.CreatedAt.WithTable(&tableName)
    ret.UpdatedAt = ret.UpdatedAt.WithTable(&tableName)
    ret.DeletedAt = ret.DeletedAt.WithTable(&tableName)
    ret.Name = ret.Name.WithTable(&tableName)
    ret.Age = ret.Age.WithTable(&tableName)
    return ret
}

var UserTable = LUserTable[User]{
    tableName: "users",
    ID:        field.NewComparable[uint]("", "id"),
    CreatedAt: field.NewComparable[time.Time]("", "created_at"),
    UpdatedAt: field.NewComparable[time.Time]("", "updated_at"),
    DeletedAt: field.NewComparable[gorm.DeletedAt]("", "deleted_at"),
    Name:      field.NewPattern[string]("", "name"),
    Age:       field.NewComparable[int32]("", "age"),
}
```

## 使用生成的代码

```go
// 使用别名
var u = UserTable.As("u")

// 构建查询
err := gsql.Query().
    Select(u.Name).
    From(u).
    Where(u.Age.Gt(18)).
    Find(db, &users)
```

## 注意事项

1. 生成的文件以`_query.go`结尾
2. 需要手动添加必要的import(如time包)
3. 表名推导规则:
   - 优先使用`TableName()`方法返回的值
   - 否则使用结构体名的蛇形命名+复数形式(如User→users)
