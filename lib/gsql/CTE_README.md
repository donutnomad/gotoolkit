# CTE (Common Table Expressions) 功能

## 概述

成功为 gsql 库添加了 CTE（公共表表达式）支持，兼容 MySQL 8.0+ 的 `WITH` 和 `WITH RECURSIVE` 语法。

## 实现方式

### 1. 核心设计

- 在 `QueryBuilderG[T]` 结构中添加了 `ctes []CTEDefinition` 字段
- 创建了 `CTEClause` 实现 GORM 的 `clause.Interface`
- 将 "CTE" 添加到 `queryClauses` 构建顺序中（位于 SELECT 之前）

### 2. API

```go
// 添加 CTE
func (b *QueryBuilder) With(name string, query *QueryBuilder, columns ...string) *QueryBuilder

// 添加递归 CTE
func (b *QueryBuilder) WithRecursive(name string, query *QueryBuilder, columns ...string) *QueryBuilder
```

## 使用示例

### 基本 CTE

```go
u := UserSchema

// WITH user_summary AS (
//   SELECT id, name FROM users WHERE age > 18
// )
// SELECT * FROM users
query := gsql.Select(gsql.Star).
    From(u).
    With("user_summary",
        gsql.Select(u.ID, u.Name).
            From(u).
            Where(u.Age.Gt(18)),
    )

sql := query.ToSQL()
// 输出: WITH `user_summary` AS (SELECT ...) SELECT ...
```

### 多个 CTE

```go
// WITH 
//   young_users AS (SELECT * FROM users WHERE age < 30),
//   old_users AS (SELECT * FROM users WHERE age >= 30)
// SELECT * FROM users
query := gsql.Select(gsql.Star).
    From(u).
    With("young_users",
        gsql.Select(u.Star()).From(u).Where(u.Age.Lt(30)),
    ).
    With("old_users",
        gsql.Select(u.Star()).From(u).Where(u.Age.Gte(30)),
    )
```

### 指定列名

```go
// WITH user_info (user_id, user_name) AS (...)
query := gsql.Select(gsql.Star).
    From(u).
    With("user_info",
        gsql.Select(u.ID, u.Name).From(u),
        "user_id", "user_name", // 指定列名
    )
```

### 与其他子句结合

CTE 可以与所有现有的查询子句自由组合：

```go
query := gsql.Select(u.ID, u.Name).
    From(u).
    With("active_users",
        gsql.Select(u.ID).From(u).Where(u.Status.Eq("active")),
    ).
    Join(...).           // JOIN
    Where(...).          // WHERE
    GroupBy(...).        // GROUP BY
    Having(...).         // HAVING
    Order(...).          // ORDER BY
    Limit(10)            // LIMIT
```

## 技术细节

### 文件修改

1. **clause_cte.go** (新增)
   - `CTEDefinition` - CTE 定义结构
   - `CTEClause` - 实现 clause.Interface

2. **query_g.go**
   - 添加 `ctes []CTEDefinition` 字段
   - 添加 `With()` 和 `WithRecursive()` 方法
   - 在 `buildStmt()` 中构建 CTE 子句
   - 在 `Clone()` 中复制 ctes

3. **query.go**
   - 添加 `With()` 和 `WithRecursive()` 代理方法

4. **public.go**
   - 更新 `queryClauses` 包含 "CTE"

### SQL 构建流程

```
QueryBuilder.With("cte_name", subquery)
    ↓
添加到 QueryBuilderG.ctes
    ↓
buildStmt() 时通过 stmt.AddClause(CTEClause{...})
    ↓
GORM 按 queryClauses 顺序构建
    ↓
WITH cte_name AS (...) SELECT ...
```

## 限制

1. **MySQL 版本**: 需要 MySQL 8.0+
2. **递归判断**: 当前实现暂未自动检测递归，需要手动使用 `WithRecursive()`
3. **CTE 引用**: 需要手动使用 `gsql.TableName("cte_name")` 来引用 CTE 表

## 未来增强

1. 自动检测递归 CTE
2. 类型安全的 CTE 表引用
3. CTE 物化提示（MATERIALIZED / NOT MATERIALIZED）

## 总结

✅ 完全集成到现有的 QueryBuilder 架构
✅ 不破坏任何现有 API
✅ 支持链式调用
✅ 与所有查询子句兼容
✅ 遵循 GORM clause 机制

