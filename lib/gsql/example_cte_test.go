package gsql

import (
	"fmt"
	"testing"
)

// TestCTEExample_Basic 演示基本的 CTE 用法
func TestCTEExample_Basic(t *testing.T) {
	// 创建一个 CTE，然后在主查询中使用它
	sql := With("user_summary",
		Select(Field("id"), Field("name"), Field("age")).
			From(TableName("users").Ptr()).
			Where(Expr("age > ?", 18)),
	).Select(Star).
		From(TableName("user_summary").Ptr()).
		Where(Expr("age < ?", 30)).
		ToSQL()

	fmt.Printf("基本 CTE 用法:\n%s\n\n", sql)
}

// TestCTEExample_Multiple 演示多个 CTE
func TestCTEExample_Multiple(t *testing.T) {
	// 使用多个 CTE，分别统计年轻用户和老用户
	sql := With("young_users",
		Select(Star).
			From(TableName("users").Ptr()).
			Where(Expr("age < ?", 30)),
	).And("old_users",
		Select(Star).
			From(TableName("users").Ptr()).
			Where(Expr("age >= ?", 30)),
	).Select(Star).
		From(TableName("young_users").Ptr()).
		ToSQL()

	fmt.Printf("多个 CTE:\n%s\n\n", sql)
}

// TestCTEExample_WithColumns 演示带列名的 CTE
func TestCTEExample_WithColumns(t *testing.T) {
	// 显式指定 CTE 的列名
	sql := With("user_info",
		Select(Field("id"), Field("name"), Field("email")).
			From(TableName("users").Ptr()),
		"user_id", "user_name", "user_email", // 指定列名
	).Select(Field("user_id"), Field("user_name")).
		From(TableName("user_info").Ptr()).
		ToSQL()

	fmt.Printf("带列名的 CTE:\n%s\n\n", sql)
}

// TestCTEExample_Recursive 演示递归 CTE
func TestCTEExample_Recursive(t *testing.T) {
	// 递归 CTE 示例：生成数字序列
	sql := WithRecursive("numbers",
		Select(Field("n")).
			From(TableName("dual").Ptr()).
			Where(Expr("n = ?", 1)),
	).Select(Star).
		From(TableName("numbers").Ptr()).
		Where(Expr("n <= ?", 10)).
		ToSQL()

	fmt.Printf("递归 CTE (数字序列基础):\n%s\n\n", sql)
}

// TestCTEExample_RecursiveTree 演示递归 CTE 查询树形结构
func TestCTEExample_RecursiveTree(t *testing.T) {
	// 递归查询组织架构树的锚点部分
	sql := WithRecursive("org_tree",
		Select(
			Field("id"),
			Field("name"),
			Field("parent_id"),
			Field("level"),
		).
			From(TableName("departments").Ptr()).
			Where(Expr("parent_id IS NULL")),
	).Select(Star).
		From(TableName("org_tree").Ptr()).
		Order(Field("level"), true).
		ToSQL()

	fmt.Printf("递归 CTE (组织架构树基础):\n%s\n\n", sql)
}

// TestCTEExample_WithJoin 演示 CTE 与 JOIN
func TestCTEExample_WithJoin(t *testing.T) {
	sql := With("active_users",
		Select(Field("id"), Field("name")).
			From(TableName("users").Ptr()).
			Where(Expr("status = ?", "active")),
	).Select(
		Field("orders.id"),
		Field("orders.total"),
		Field("active_users.name"),
	).
		From(TableName("orders").Ptr()).
		Join(InnerJoin(TableName("active_users").Ptr()).
			On(Expr("orders.user_id = active_users.id"))).
		ToSQL()

	fmt.Printf("CTE 与 JOIN:\n%s\n\n", sql)
}

// TestCTEExample_Aggregation 演示 CTE 与聚合
func TestCTEExample_Aggregation(t *testing.T) {
	sql := With("monthly_sales",
		Select(
			Field("month"),
			Field("total"),
		).
			From(TableName("orders").Ptr()).
			GroupBy(Field("month")),
	).Select(
		Field("month"),
		Field("total"),
	).
		From(TableName("monthly_sales").Ptr()).
		Order(Field("month"), true).
		ToSQL()

	fmt.Printf("CTE 与聚合:\n%s\n\n", sql)
}
