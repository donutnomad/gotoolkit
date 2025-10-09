package gsql

import (
	"fmt"
	"testing"

	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func buildStmt() *gorm.Statement {
	tx := &gorm.DB{
		Config: &gorm.Config{
			ClauseBuilders: map[string]clause.ClauseBuilder{},
			Dialector:      dialector,
		},
		Statement: &gorm.Statement{
			Clauses:      map[string]clause.Clause{},
			BuildClauses: queryClauses,
		},
	}
	tx.Statement.DB = tx
	return tx.Statement
}

// WriteQuoted的作用
// dd => `dd`
// table.name => `table`.`name`
// table`name => `table“name`
func TestWriteQuoted(t *testing.T) {
	stmt := buildStmt()

	stmt.WriteQuoted("dd")
	fmt.Println(stmt.SQL.String())
	stmt.SQL.Reset()

	stmt.WriteQuoted("table.name")
	fmt.Println(stmt.SQL.String())
	stmt.SQL.Reset()

	stmt.WriteQuoted("table`name")
	fmt.Println(stmt.SQL.String())
	stmt.SQL.Reset()
}

func Test2(t *testing.T) {
	stmt := buildStmt()

	f1 := field.NewComparable[int]("", "id")
	f2 := field.NewComparable[int]("", "id2")

	LeftJoin(TableName2("DDD")).On(f1.EqF(f2)).Build(stmt)
	fmt.Println(stmt.SQL.String())
	stmt.SQL.Reset()

	LeftJoin(&compactFromImpl{
		tableName: "DDD",
		expr:      Expr("SELECT aaa FROM Name"),
	}).On(f1.EqF(f2)).Build(stmt)
	fmt.Println(stmt.SQL.String())
	stmt.SQL.Reset()

	InnerJoin(JsonTable(
		field.NewBase("alt", "exchange_rules"),
		"$[*]",
	).
		AddColumn("symbol", "VARCHAR(255)", "$.token_symbol").
		AddColumn("symbol2", "VARCHAR(255)", "$.token_symbol", "ERROR").
		As("t"),
	).OnEmpty().Build(stmt)
	fmt.Println(stmt.SQL.String())
	//Joins("JOIN JSON_TABLE(alt.exchange_rules, '$[*]' COLUMNS(symbol VARCHAR(255) PATH '$.token_symbol')) AS t").

	//e := clause.Expr{
	//	SQL:  "SELECT aaa ?",
	//	Vars: []any{clause.Expr{SQL: "FROM DDD"}},
	//}
	//e.Build(stmt)

	//stmt.AddVar()
}

type compactFromImpl struct {
	tableName string
	expr      clause.Expression
}

func (c *compactFromImpl) ToExpr() clause.Expression {
	return c.expr
}

func (c *compactFromImpl) TableName() string {
	return c.tableName
}
