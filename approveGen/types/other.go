package types

import (
	"github.com/dave/jennifer/jen"
	utils2 "github.com/donutnomad/gotoolkit/internal/utils"
	"github.com/samber/lo"
	"strings"
)

type Param struct {
	Name utils2.EString
	Type Type // mo.Option[bool]
}
type Type string

func (t Type) IsPtr() bool {
	return strings.HasPrefix(string(t), "*")
}

func (t Type) NoPtr() Type {
	if strings.HasPrefix(string(t), "*") {
		return t[1:]
	}
	return t
}

func (t Type) Placeholder() string {
	typ := string(t.NoPtr())
	if lo.Contains([]string{"int", "int8", "int16", "int32", "int64"}, typ) {
		return "%d"
	}
	if lo.Contains([]string{"uint", "uint8", "uint16", "uint32", "uint64"}, typ) {
		return "%d"
	}
	if typ == "string" {
		return "%s"
	}
	if lo.Contains([]string{"float32", "float64"}, typ) {
		return "%f"
	}
	return "%v"
}

type JenStatementSlice []*jen.Statement

func (a JenStatementSlice) As() []jen.Code {
	return lo.Map(a, func(item *jen.Statement, index int) jen.Code {
		return item
	})
}
