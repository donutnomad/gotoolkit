package gsql

import (
	"testing"
)

// TestCTEBasic 测试基本的 CTE 用法
func TestCTEBasic(t *testing.T) {
	// WITH user_summary AS (
	//   SELECT id, name FROM users WHERE age > 18
	// )
	// SELECT * FROM users
	sql := With("user_summary",
		Select(Field("id"), Field("name")).
			From(TableName("users").Ptr()).
			Where(Expr("age > ?", 18)),
	).Select(Star).
		From(TableName("users").Ptr()).
		ToSQL()

	t.Logf("CTE Basic SQL:\n%s", sql)

	if !contains(sql, "WITH") {
		t.Errorf("Expected SQL to contain WITH")
	}
	if !contains(sql, "user_summary") {
		t.Errorf("Expected SQL to contain user_summary")
	}
}

// TestCTEMultiple 测试多个 CTE
func TestCTEMultiple(t *testing.T) {
	// WITH
	//   young AS (SELECT * FROM users WHERE age < 30),
	//   old AS (SELECT * FROM users WHERE age >= 30)
	// SELECT * FROM combined
	sql := With("young",
		Select(Star).
			From(TableName("users").Ptr()).
			Where(Expr("age < ?", 30)),
	).And("old",
		Select(Star).
			From(TableName("users").Ptr()).
			Where(Expr("age >= ?", 30)),
	).Select(Star).
		From(TableName("combined").Ptr()).
		ToSQL()

	t.Logf("Multiple CTE SQL:\n%s", sql)

	if !contains(sql, "WITH") {
		t.Errorf("Expected SQL to contain WITH")
	}
	if !contains(sql, "young") {
		t.Errorf("Expected SQL to contain young")
	}
	if !contains(sql, "old") {
		t.Errorf("Expected SQL to contain old")
	}
}

// TestCTEWithColumns 测试带列名的 CTE
func TestCTEWithColumns(t *testing.T) {
	// WITH user_info (user_id, user_name) AS (
	//   SELECT id, name FROM users
	// )
	// SELECT * FROM users
	sql := With("user_info",
		Select(Field("id"), Field("name")).
			From(TableName("users").Ptr()),
		"user_id", "user_name", // 指定列名
	).Select(Star).
		From(TableName("users").Ptr()).
		ToSQL()

	t.Logf("CTE with columns SQL:\n%s", sql)

	if !contains(sql, "user_info") {
		t.Errorf("Expected SQL to contain user_info")
	}
	if !contains(sql, "user_id") {
		t.Errorf("Expected SQL to contain user_id")
	}
}

// TestCTERecursive 测试递归 CTE
func TestCTERecursive(t *testing.T) {
	// WITH RECURSIVE numbers AS (
	//   SELECT 1 as n
	// )
	// SELECT * FROM numbers
	sql := WithRecursive("numbers",
		Select(Primitive(1).AsF("n")).
			From(TableName("dual").Ptr()),
	).Select(Star).
		From(TableName("numbers").Ptr()).
		ToSQL()

	t.Logf("Recursive CTE SQL:\n%s", sql)

	if !contains(sql, "WITH RECURSIVE") {
		t.Errorf("Expected SQL to contain WITH RECURSIVE")
	}
	if !contains(sql, "numbers") {
		t.Errorf("Expected SQL to contain numbers")
	}
}

// TestCTEWithJoin 测试 CTE 与 JOIN
func TestCTEWithJoin(t *testing.T) {
	// WITH active_users AS (
	//   SELECT id FROM users WHERE status = 'active'
	// )
	// SELECT * FROM orders o
	// INNER JOIN active_users au ON o.user_id = au.id
	sql := With("active_users",
		Select(Field("id")).
			From(TableName("users").Ptr()).
			Where(Expr("status = ?", "active")),
	).Select(Star).
		From(TableName("orders").Ptr()).
		Join(InnerJoin(TableName("active_users").Ptr()).
			On(Expr("orders.user_id = active_users.id"))).
		ToSQL()

	t.Logf("CTE with JOIN SQL:\n%s", sql)

	if !contains(sql, "WITH") {
		t.Errorf("Expected SQL to contain WITH")
	}
	if !contains(sql, "active_users") {
		t.Errorf("Expected SQL to contain active_users")
	}
	if !contains(sql, "INNER JOIN") {
		t.Errorf("Expected SQL to contain INNER JOIN")
	}
}

// TestCTEWithWhere 测试 CTE 与 WHERE 子句
func TestCTEWithWhere(t *testing.T) {
	sql := With("summary",
		Select(Field("id"), Field("total")).
			From(TableName("orders").Ptr()),
	).Select(Star).
		From(TableName("summary").Ptr()).
		Where(Expr("total > ?", 1000)).
		ToSQL()

	t.Logf("CTE with WHERE SQL:\n%s", sql)

	if !contains(sql, "WITH") || !contains(sql, "WHERE") {
		t.Errorf("Expected SQL to contain WITH and WHERE")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
