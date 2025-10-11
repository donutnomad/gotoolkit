package gsql

import (
	"strings"
	"testing"

	"gorm.io/gorm/clause"
)

func TestCase_SearchedCase(t *testing.T) {
	// 搜索式 CASE: CASE WHEN condition THEN result ... END
	tests := []struct {
		name     string
		builder  *CaseBuilder
		wantSQL  string
		wantVars []interface{}
	}{
		{
			name: "simple when-then",
			builder: Case().
				When(Expr("status = ?", "active"), Primitive("Active")),
			wantSQL:  "CASE WHEN status = ? THEN ? END",
			wantVars: []interface{}{"active", "Active"},
		},
		{
			name: "multiple when-then",
			builder: Case().
				When(Expr("status = ?", "active"), Primitive("Active")).
				When(Expr("status = ?", "pending"), Primitive("Pending")),
			wantSQL:  "CASE WHEN status = ? THEN ? WHEN status = ? THEN ? END",
			wantVars: []interface{}{"active", "Active", "pending", "Pending"},
		},
		{
			name: "with else",
			builder: Case().
				When(Expr("age >= ?", 18), Primitive("Adult")).
				Else(Primitive("Minor")),
			wantSQL:  "CASE WHEN age >= ? THEN ? ELSE ? END",
			wantVars: []interface{}{18, "Adult", "Minor"},
		},
		{
			name: "multiple conditions with else",
			builder: Case().
				When(Expr("score >= ?", 90), Primitive("A")).
				When(Expr("score >= ?", 80), Primitive("B")).
				When(Expr("score >= ?", 70), Primitive("C")).
				Else(Primitive("F")),
			wantSQL:  "CASE WHEN score >= ? THEN ? WHEN score >= ? THEN ? WHEN score >= ? THEN ? ELSE ? END",
			wantVars: []interface{}{90, "A", 80, "B", 70, "C", "F"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := buildSQL(tt.builder)
			if !strings.Contains(sql, tt.wantSQL) {
				t.Errorf("SQL mismatch:\ngot:  %s\nwant: %s", sql, tt.wantSQL)
			}
		})
	}
}

func TestCase_SimpleCaseValue(t *testing.T) {
	// 简单 CASE: CASE value WHEN compare THEN result ... END
	tests := []struct {
		name     string
		builder  *CaseBuilder
		wantSQL  string
		wantVars []interface{}
	}{
		{
			name: "simple case with value",
			builder: CaseValue(Expr("status")).
				When(Primitive("active"), Primitive(1)).
				When(Primitive("inactive"), Primitive(0)),
			wantSQL:  "CASE status WHEN ? THEN ? WHEN ? THEN ? END",
			wantVars: []interface{}{"active", 1, "inactive", 0},
		},
		{
			name: "simple case with else",
			builder: CaseValue(Expr("level")).
				When(Primitive("gold"), Primitive(0.9)).
				When(Primitive("silver"), Primitive(0.95)).
				Else(Primitive(1.0)),
			wantSQL:  "CASE level WHEN ? THEN ? WHEN ? THEN ? ELSE ? END",
			wantVars: []interface{}{"gold", 0.9, "silver", 0.95, 1.0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql := buildSQL(tt.builder)
			if !strings.Contains(sql, tt.wantSQL) {
				t.Errorf("SQL mismatch:\ngot:  %s\nwant: %s", sql, tt.wantSQL)
			}
		})
	}
}

func TestCase_AsField(t *testing.T) {
	// 测试作为字段使用
	caseExpr := Case().
		When(Expr("amount > ?", 1000), Primitive("High")).
		When(Expr("amount > ?", 500), Primitive("Medium")).
		Else(Primitive("Low")).
		End().AsF("amount_level")

	if caseExpr.Name() != "amount_level" {
		t.Errorf("Field name mismatch: got %s, want amount_level", caseExpr.Name())
	}

	// 验证可以在 SELECT 中使用
	sql := Select(caseExpr).
		From(TableName("orders").Ptr()).
		ToSQL()

	if !strings.Contains(sql, "CASE") || !strings.Contains(sql, "amount_level") {
		t.Errorf("CASE expression not properly used in SELECT:\n%s", sql)
	}
	t.Log(sql)
}

// buildSQL 辅助函数：构建 SQL 字符串用于测试
func buildSQL(expr interface{ Build(clause.Builder) }) string {
	var sql strings.Builder
	var vars []interface{}

	stmt := &testStatement{
		sql:  &sql,
		vars: &vars,
	}

	expr.Build(stmt)
	return sql.String()
}

// testStatement 实现 clause.Builder 接口用于测试
type testStatement struct {
	sql  *strings.Builder
	vars *[]interface{}
}

func (s *testStatement) WriteByte(b byte) error {
	return s.sql.WriteByte(b)
}

func (s *testStatement) WriteString(str string) (int, error) {
	return s.sql.WriteString(str)
}

func (s *testStatement) WriteQuoted(field interface{}) {
	s.sql.WriteString("`")
	s.sql.WriteString(field.(string))
	s.sql.WriteString("`")
}

func (s *testStatement) AddVar(writer clause.Writer, values ...interface{}) {
	for i, v := range values {
		if i > 0 {
			writer.WriteByte(',')
		}

		// 处理 Expression 类型
		if expr, ok := v.(clause.Expression); ok {
			expr.Build(s)
			continue
		}

		writer.WriteByte('?')
		*s.vars = append(*s.vars, v)
	}
}

func (s *testStatement) AddError(err error) error {
	return err
}
