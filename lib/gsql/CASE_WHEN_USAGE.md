# CASE WHEN 构建器使用指南

## 概述

CASE WHEN 构建器提供了类型安全的方式来构建 SQL CASE 表达式，支持搜索式和简单式两种形式。

## API 设计原则

### 1. 使用 `Primitive()` 表示字面量
```go
// ✅ 推荐：使用 Primitive 表示字面量值
gsql.Primitive("VIP")
gsql.Primitive(100)
gsql.Primitive(0.7)

// ❌ 不推荐：使用 Expr("?", value)
gsql.Expr("?", "VIP")
```

### 2. 使用字段的操作符方法
```go
// ✅ 推荐：定义字段并使用类型安全的操作符
amount := field.NewComparable[int64]("", "amount")
userLevel := field.NewPattern[string]("", "user_level")

gsql.Case().
    When(amount.Gt(1000), gsql.Primitive("High")).
    When(userLevel.Eq("VIP"), gsql.Primitive("Premium"))

// ❌ 不推荐：使用字符串拼接
gsql.Case().
    When(gsql.Expr("amount > ?", 1000), gsql.Expr("?", "High")).
    When(gsql.Expr("user_level = ?", "VIP"), gsql.Expr("?", "Premium"))
```

## 基本用法

### 搜索式 CASE（CASE WHEN ... THEN ... END）

```go
amount := field.NewComparable[int64]("", "amount")

amountLevel := gsql.Case().
    When(amount.Gt(10000), gsql.Primitive("VIP")).
    When(amount.Gt(5000), gsql.Primitive("Premium")).
    When(amount.Gt(1000), gsql.Primitive("Standard")).
    Else(gsql.Primitive("Basic")).
    End().AsF("customer_level")
```

**生成的 SQL：**
```sql
CASE 
    WHEN amount > 10000 THEN 'VIP'
    WHEN amount > 5000 THEN 'Premium'
    WHEN amount > 1000 THEN 'Standard'
    ELSE 'Basic'
END AS customer_level
```

### 简单式 CASE（CASE value WHEN ... THEN ... END）

```go
status := field.NewComparable[int]("", "status")

statusDesc := gsql.CaseValue(status.ToExpr()).
    When(gsql.Primitive(0), gsql.Primitive("待处理")).
    When(gsql.Primitive(1), gsql.Primitive("处理中")).
    When(gsql.Primitive(2), gsql.Primitive("已完成")).
    Else(gsql.Primitive("未知状态")).
    End().AsF("status_desc")
```

**生成的 SQL：**
```sql
CASE status
    WHEN 0 THEN '待处理'
    WHEN 1 THEN '处理中'
    WHEN 2 THEN '已完成'
    ELSE '未知状态'
END AS status_desc
```

## 高级用法

### 1. 复杂条件（使用 AND/OR）

```go
userLevel := field.NewPattern[string]("", "user_level")
amount := field.NewComparable[int64]("", "amount")
firstOrder := field.NewComparable[bool]("", "first_order")

discount := gsql.Case().
    When(
        gsql.And(
            userLevel.Eq("VIP"),
            amount.Gt(10000),
        ),
        gsql.Primitive(0.7), // 7折
    ).
    When(
        gsql.And(
            userLevel.Eq("Premium"),
            amount.Gt(5000),
        ),
        gsql.Primitive(0.85), // 85折
    ).
    When(firstOrder.Eq(true), gsql.Primitive(0.9)). // 首单9折
    Else(gsql.Primitive(1.0)). // 原价
    End().AsF("discount_rate")
```

### 2. 在 GROUP BY 中使用

```go
amount := field.NewComparable[int64]("", "amount")

amountRange := gsql.Case().
    When(amount.Lt(100), gsql.Primitive("0-100")).
    When(amount.Lt(500), gsql.Primitive("100-500")).
    When(amount.Lt(1000), gsql.Primitive("500-1000")).
    Else(gsql.Primitive("1000+")).
    End().AsF("amount_range")

sql := gsql.Select(
    amountRange,
    gsql.COUNT().AsF("order_count"),
    gsql.SUM(amount.ToExpr()).AsF("total_amount"),
).From(gsql.TableName("orders").Ptr()).
    GroupBy(amountRange).
    ToSQL()
```

### 3. 在 ORDER BY 中使用（自定义排序优先级）

```go
status := field.NewPattern[string]("", "status")

priority := gsql.Case().
    When(status.Eq("urgent"), gsql.Primitive(1)).
    When(status.Eq("high"), gsql.Primitive(2)).
    When(status.Eq("normal"), gsql.Primitive(3)).
    Else(gsql.Primitive(4)).
    End().AsF("priority")

sql := gsql.Select(
    gsql.Field("id"),
    gsql.Field("status"),
    priority,
).From(gsql.TableName("tasks").Ptr()).
    Order(priority, true). // 按优先级升序
    ToSQL()
```

### 4. 嵌套 CASE 表达式

```go
userType := field.NewPattern[string]("", "user_type")

// 季节性折扣
seasonDiscount := gsql.Case().
    When(gsql.Expr("MONTH(created_at) IN (11,12)"), gsql.Primitive(0.8)).
    When(gsql.Expr("MONTH(created_at) = ?", 6), gsql.Primitive(0.9)).
    Else(gsql.Primitive(1.0)).
    End()

// VIP 在季节性折扣基础上再打 95 折
finalDiscount := gsql.Case().
    When(
        userType.Eq("vip"),
        gsql.Expr("(?) * 0.95", seasonDiscount),
    ).
    Else(seasonDiscount).
    End().AsF("final_discount")
```

## 可用的字段操作符

### Comparable 类型字段
- `Eq(value)` - 等于
- `Not(value)` - 不等于
- `Gt(value)` - 大于（仅数值类型）
- `Gte(value)` - 大于等于（仅数值类型）
- `Lt(value)` - 小于（仅数值类型）
- `Lte(value)` - 小于等于（仅数值类型）
- `In(values...)` - 在列表中
- `NotIn(values...)` - 不在列表中

### Pattern 类型字段（字符串）
- `Eq(value)` - 等于
- `Not(value)` - 不等于
- `Like(value)` - 模糊匹配
- `NotLike(value)` - 不模糊匹配
- `Contains(value)` - 包含
- `HasPrefix(value)` - 前缀匹配
- `HasSuffix(value)` - 后缀匹配

## API 参考

### `Case()` - 创建搜索式 CASE
```go
func Case() *CaseBuilder
```

### `CaseValue(expr)` - 创建简单式 CASE
```go
func CaseValue(value field.Expression) *CaseBuilder
```

### `When(condition, result)` - 添加 WHEN...THEN 子句
```go
func (c *CaseBuilder) When(condition, result field.Expression) *CaseBuilder
```

### `Else(value)` - 添加 ELSE 子句
```go
func (c *CaseBuilder) Else(value field.Expression) *CaseBuilder
```

### `End()` - 结束 CASE 表达式
```go
func (c *CaseBuilder) End() field.ExpressionTo
```

### `AsF(alias)` - 设置字段别名
```go
// End() 返回 field.ExpressionTo，可调用 AsF
caseExpr := gsql.Case().
    When(..., ...).
    End().AsF("my_alias")
```

## 最佳实践

1. **始终使用 `Primitive()` 表示字面量值**
2. **定义字段变量并使用类型安全的操作符方法**
3. **为 CASE 表达式设置有意义的别名（使用 `AsF()`）**
4. **对于复杂条件，使用 `gsql.And()` 和 `gsql.Or()` 组合**
5. **嵌套 CASE 时，将内层 CASE 赋值给变量以提高可读性**

## 示例：完整查询

```go
package main

import (
    "github.com/donutnomad/gotoolkit/lib/gsql"
    "github.com/donutnomad/gotoolkit/lib/gsql/field"
)

func main() {
    // 定义字段
    amount := field.NewComparable[int64]("orders", "amount")
    userLevel := field.NewPattern[string]("orders", "user_level")
    
    // 构建折扣规则
    discount := gsql.Case().
        When(
            gsql.And(
                userLevel.Eq("VIP"),
                amount.Gt(10000),
            ),
            gsql.Primitive(0.7),
        ).
        When(
            gsql.And(
                userLevel.Eq("Premium"),
                amount.Gt(5000),
            ),
            gsql.Primitive(0.85),
        ).
        Else(gsql.Primitive(1.0)).
        End().AsF("discount_rate")
    
    // 构建查询
    sql := gsql.Select(
        gsql.Field("id"),
        amount,
        userLevel,
        discount,
    ).From(gsql.TableName("orders").Ptr()).
        Where(amount.Gt(0)).
        Order(amount, false).
        ToSQL()
    
    println(sql)
}
```

