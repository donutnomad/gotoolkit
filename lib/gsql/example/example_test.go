package example

import (
	"testing"

	"github.com/donutnomad/gotoolkit/lib/gsql"
)

// 说明：
// 该用例演示本次新增 API 基本用法（基于生成的 Schema），
// 通过生成 SQL 字符串进行断言，便于快速检查 DSL 是否正确构建。

func Test_BasicUsage_GroupBy_Having_For_IndexHint_Partition(t *testing.T) {
	u := UserSchema.As("u")
	o := OrderSchema.As("o")

	// 基础查询：GroupBy/Having + 锁子句 + 分区 + 索引提示
	sql := gsql.Select(u.OrgID, gsql.COUNT(u.ID).AsF("cnt")).
		From(u).
		Partition("p2025_10").
		UseIndex("idx_users_status").
		UseIndexForOrderBy("idx_users_created_at").
		Where(u.Status.Eq("active")).
		GroupBy(u.OrgID).
		Having(gsql.Expr("COUNT(?) > ?", u.ID.ToExpr(), 10)).
		Order(u.CreatedAt, false).
		ForUpdate().
		Nowait().
		ToSQL()

	if len(sql) == 0 {
		t.Fatal("expected non-empty SQL")
	}
	t.Log(sql)

	// Join 示例：
	jsql := gsql.Select(gsql.Star).
		From(u).
		Join(
			gsql.LeftJoin(o).On(u.ID.EqF(o.UserID)),
		).
		Where(u.Status.Eq("active")).
		ForShare().
		SkipLocked().
		ToSQL()

	if len(jsql) == 0 {
		t.Fatal("expected non-empty join SQL")
	}
	t.Log(jsql)
}
