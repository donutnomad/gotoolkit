package automap

import (
	"time"

	"github.com/samber/mo"
	"gorm.io/datatypes"
)

// 支持的效果:

type ID uint64
type Address [20]byte

type Base struct {
	ID        uint64    `gorm:"column:id999;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at2"`
	DeletedAt *time.Time
}

type A struct {
	ID               ID
	CreatedAt        time.Time
	DeletedAt        *time.Time
	Book             A_Book
	Book2            A_Book
	Address          Address
	TokenName        string
	TokenSymbol      string
	TokenDecimals    uint8
	TokenTotalSupply uint64
	Others           []string
	patch            APatch
}

func (a *A) ExportPatch() *APatch {
	return &a.patch
}

type APatch struct {
	ID               mo.Option[ID]
	CreatedAt        mo.Option[time.Time]
	DeletedAt        mo.Option[*time.Time]
	Book             mo.Option[A_Book]
	Address          mo.Option[Address]
	TokenName        mo.Option[string]
	TokenSymbol      mo.Option[string]
	TokenDecimals    mo.Option[uint8]
	TokenTotalSupply mo.Option[uint64]
	Others           mo.Option[[]string]
}

type A_Book struct {
	Name   string
	Author string
	Year   int
}

type B struct {
	Base
	Token      datatypes.JSONType[B_Token] `gorm:"column:token21"`
	Token2     datatypes.JSONType[B_Token] `gorm:"column:token22"`
	BookName   string                      `gorm:"column:book_name1"`
	BookAuthor string                      `gorm:"column:book_author2"`
	BookYear   int                         `gorm:"column:book_year3"`

	BookName2   string `gorm:"column:book_name12"`
	BookAuthor2 string `gorm:"column:book_author22"`
	BookYear2   int    `gorm:"column:book_year32"`

	Address Address                     `gorm:"column:address4"`
	Others  datatypes.JSONSlice[string] `gorm:"column:others5"`
}

type B_Token struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Decimals    uint8  `json:"decimals"`
	TotalSupply uint64 `json:"total_supply"`
}

type Gen struct {
}

func (g *Gen) MapAToB2(a *A) *B {
	tokenName := a.TokenName
	if tokenName == "" {
		tokenName = "xxxx"
	}
	ret := &B{
		Base: Base{
			ID:        uint64(a.ID),
			CreatedAt: a.CreatedAt,
			DeletedAt: a.DeletedAt,
		},
		Token: datatypes.NewJSONType(B_Token{
			Name:        tokenName,
			Symbol:      a.TokenSymbol,
			Decimals:    a.TokenDecimals,
			TotalSupply: a.TokenTotalSupply,
		}),
		Token2: datatypes.NewJSONType(B_Token{
			Name:   tokenName,
			Symbol: a.TokenSymbol,
		}),
		BookName:   a.Book.Name,
		BookAuthor: a.Book.Author,
		BookYear:   a.Book.Year,

		BookName2:   a.Book2.Name,
		BookAuthor2: a.Book2.Author,

		Address: a.Address,
		Others:  datatypes.NewJSONSlice(a.Others),
	}
	return ret
}

//
//// 期望的输出
//func Do(input *A) map[string]any {
//	b := MapAToB(input)
//	fields := input.ExportPatch()
//	var ret = make(map[string]any)
//	if fields.ID.IsPresent() {
//		ret["id"] = b.ID
//	}
//	// A的一个字段，对应B的多字段
//	if fields.Book.IsPresent() {
//		ret["book_name"] = b.BookName
//		ret["book_author"] = b.BookAuthor
//		ret["book_year"] = b.BookYear
//	}
//	if fields.Address.IsPresent() {
//		ret["address"] = b.Address
//	}
//	if fields.Others.IsPresent() {
//		ret["others"] = b.Others
//	}
//	// B的一个字段，对应A的多字段, 如果判断此时B是JSONType类型，则这么处理，否则直接赋值全部
//	// B.token1
//	{
//		set := datatypes.JSONSet("token1")
//		if fields.TokenName.IsPresent() {
//			ret["token1"] = set.Set("name", fields.TokenName.MustGet())
//		}
//		if fields.TokenSymbol.IsPresent() {
//			ret["token1"] = set.Set("symbol", fields.TokenSymbol.MustGet())
//		}
//		if fields.TokenDecimals.IsPresent() {
//			ret["token1"] = set.Set("decimals", fields.TokenDecimals.MustGet())
//		}
//		if fields.TokenTotalSupply.IsPresent() {
//			ret["token1"] = set.Set("total_supply", fields.TokenTotalSupply.MustGet())
//		}
//	}
//	return ret
//}

// mapAToBTest 从mod.go文件调用AutoMap的测试函数
func mapAToBTest() (*ParseResult, error) {
	return Parse("MapAToB")
}
