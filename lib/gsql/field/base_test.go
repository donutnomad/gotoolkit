package field

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBase(t *testing.T) {
	sub := NewBaseFromSql(Expression{
		Query: "1 + 2",
	})
	ret := sub.As("sub").Column()
	assert.Equal(t, "(1 + 2) AS sub", ret.Query)

	sub2 := NewBaseFromSql(Expression{
		Query: "SELECT ddd FROM aaa WHERE aaa.dd = ?",
		Args:  []any{"123"},
	})
	ret = sub2.As("sub").Column()
	fmt.Println(ret.Query, ret.Args)

	sub3 := NewBase("tt", "name")
	ret = sub3.As("sub").Column()
	fmt.Println(ret.Query, ret.Args)
}

type User string

func TestString(t *testing.T) {
	n := NewPattern[User]("", "name")
	d := n.Like("Tom")

	fmt.Println(d.Query, d.Args)

	d = n.HasPrefix("Tom")
	fmt.Println(d.Query, d.Args)

	d = n.HasSuffix("Tom")
	fmt.Println(d.Query, d.Args)

	d = n.Contains("Tom")
	fmt.Println(d.Query, d.Args)
}
