package gsql_test

import (
	"testing"
	"time"

	"github.com/donutnomad/gotoolkit/lib/gsql"
	"github.com/donutnomad/gotoolkit/lib/gsql/field"
)

// 演示 CASE WHEN 构建器的实际使用场景

func TestCaseExample_SearchedCase(t *testing.T) {
	// 场景：根据订单金额分级
	amount := field.NewComparable[int64]("", "amount")

	amountLevel := gsql.Case().
		When(amount.Gt(10000), gsql.Primitive("VIP")).
		When(amount.Gt(5000), gsql.Primitive("Premium")).
		When(amount.Gt(1000), gsql.Primitive("Standard")).
		Else(gsql.Primitive("Basic")).
		End().AsF("customer_level")

	sql := gsql.Select(
		gsql.Field("id"),
		gsql.Field("amount"),
		amountLevel,
	).From(gsql.TableName("orders").Ptr()).ToSQL()

	t.Logf("搜索式 CASE SQL:\n%s", sql)
}

func TestCaseExample_SimpleCaseValue(t *testing.T) {
	// 场景：将状态码转换为中文描述
	status := field.NewComparable[int]("", "status")

	statusDesc := gsql.CaseValue(status.ToExpr()).
		When(gsql.Primitive(0), gsql.Primitive("待处理")).
		When(gsql.Primitive(1), gsql.Primitive("处理中")).
		When(gsql.Primitive(2), gsql.Primitive("已完成")).
		Else(gsql.Primitive("未知状态")).
		End().AsF("status_desc")

	sql := gsql.Select(
		gsql.Field("id"),
		gsql.Field("status"),
		statusDesc,
	).From(gsql.TableName("orders").Ptr()).ToSQL()

	t.Logf("简单 CASE SQL:\n%s", sql)
}

func TestCaseExample_ComplexScenario(t *testing.T) {
	// 场景：根据多个条件计算折扣
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
		Else(gsql.Primitive(1.0)).                      // 原价
		End().AsF("discount_rate")

	// 计算最终金额（amount * discount_rate）
	finalAmount := gsql.Field("amount * (" + discount.Name() + ")").As("final_amount")

	sql := gsql.Select(
		gsql.Field("id"),
		gsql.Field("amount"),
		discount,
		finalAmount,
	).From(gsql.TableName("orders").Ptr()).ToSQL()

	t.Logf("复杂条件 CASE SQL:\n%s", sql)
}

func TestCaseExample_InGroupBy(t *testing.T) {
	// 场景：按金额分段统计订单数
	amount := field.NewComparable[int64]("", "amount")

	amountRange := gsql.Case().
		When(amount.Lt(100), gsql.Primitive("0-100")).
		When(amount.Lt(500), gsql.Primitive("100-500")).
		When(amount.Lt(1000), gsql.Primitive("500-1000")).
		Else(gsql.Primitive("1000+")).
		End().AsF("amount_range")

	sql := gsql.
		Select(
			amountRange,
			gsql.COUNT().AsF("order_count"),
			gsql.SUM(amount).AsF("total_amount"),
		).
		From(gsql.TableName("orders").Ptr()).
		GroupBy(amountRange).
		ToSQL()

	t.Logf("GROUP BY with CASE SQL:\n%s", sql)
}

func TestCaseExample_InOrderBy(t *testing.T) {
	// 场景：自定义排序优先级
	status := field.NewPattern[string]("", "status")
	id := field.NewPattern[uint]("", "id")

	priority := gsql.Case().
		When(status.Eq("urgent"), gsql.Primitive(1)).
		When(status.Eq("high"), gsql.Primitive(2)).
		When(status.Eq("normal"), gsql.Primitive(3)).
		Else(gsql.Primitive(4)).
		End().AsF("priority")

	sql := gsql.
		Select(
			id,
			status,
			priority,
		).
		From(gsql.TableName("tasks").Ptr()).
		Order(priority, true). // 按优先级升序
		ToSQL()

	t.Logf("ORDER BY with CASE SQL:\n%s", sql)
}

func TestCaseExample_NestedCase(t *testing.T) {
	// 场景：嵌套 CASE 表达式
	userType := field.NewPattern[string]("", "user_type")
	createdAt := field.NewPattern[time.Time]("", "created_at")
	monthCreatedAt := field.NewComparableWithField[int](gsql.MONTH(createdAt).AsF())

	// 季节性折扣
	seasonDiscount := gsql.Case().
		When(monthCreatedAt.In(11, 12), gsql.Primitive(0.8)). // 双11双12
		When(monthCreatedAt.Eq(6), gsql.Primitive(0.9)).      // 618
		Else(gsql.Primitive(1.0)).
		End()

	// VIP 在季节性折扣基础上再打 95 折
	finalDiscount := gsql.Case().
		When(
			userType.Eq("vip"),
			gsql.Mul(seasonDiscount, gsql.Primitive(0.95)), // VIP在活动基础上再打95折
		).
		Else(seasonDiscount).
		End().AsF("final_discount")

	sql := gsql.Select(
		gsql.Field("`id`"),
		gsql.Field("`user_type`"),
		finalDiscount,
	).From(gsql.TableName("orders").Ptr()).ToSQL()

	t.Logf("嵌套 CASE SQL:\n%s", sql)
}
