package types

import (
	"fmt"
	"go/ast"
	"strings"

	utils2 "github.com/donutnomad/gotoolkit/internal/utils"
	xast2 "github.com/donutnomad/gotoolkit/internal/xast"
	"github.com/samber/lo"
)

type MyMethodSlice []MyMethod

func (s MyMethodSlice) ToMap() map[string]MyMethod {
	return lo.SliceToMap(s, func(item MyMethod) (string, MyMethod) {
		return item.OutStructName(), item
	})
}

type MyMethod struct {
	ObjName    string // (p *Struct) ==> p
	StructName string // (p *Struct) ==> *Struct

	MethodName    string
	MethodParams  []*ast.Field
	MethodResults []*ast.Field

	Func     *ast.FuncDecl
	Comment  []string
	StartPos int
	EndPos   int
	Recv     *ast.FieldList

	Imports     xast2.ImportInfoSlice
	PkgPath     string
	FilePkgName string
}

func (m *MyMethod) Copy() MyMethod {
	return MyMethod{
		ObjName:       m.ObjName,
		StructName:    m.StructName,
		MethodName:    m.MethodName,
		MethodParams:  m.MethodParams,
		MethodResults: m.MethodResults,
		Func:          m.Func,
		Comment:       m.Comment,
		StartPos:      m.StartPos,
		EndPos:        m.EndPos,
		Recv:          m.Recv,
		Imports:       m.Imports,
		PkgPath:       m.PkgPath,
		FilePkgName:   m.FilePkgName,
	}
}

func (m *MyMethod) ExtractImportPath() []string {
	var newSlice []*ast.Field
	newSlice = append(newSlice, m.MethodParams...)
	newSlice = append(newSlice, m.MethodResults...)

	var out []string
	for _, param := range newSlice {
		xast2.GetFieldType(param.Type, func(expr *ast.SelectorExpr) string {
			x := expr.X.(*ast.Ident).Name // mo
			out = append(out, m.Imports.Find(x).GetPath())
			return ""
		})
	}

	return out
}

func (m *MyMethod) GenMethod() string {
	return fmt.Sprintf("%s_%s", m.StructNameWithoutPtr(), m.MethodName)
}

func (m *MyMethod) StructNameWithoutPtr() string {
	return parseString(m.StructName)
}

func (m *MyMethod) AsParams(getType func(typ ast.Expr) string) []Param {
	var args []Param
	for _, p := range m.MethodParams {
		for _, name := range p.Names {
			args = append(args, Param{
				Name: utils2.EString(name.Name),
				Type: Type(getType(p.Type)),
			})
		}
	}
	return args
}

func (m *MyMethod) IsStructMethod() bool {
	return m.StructName != ""
}

func (m *MyMethod) FindAnnoBody(name string) ([]string, error) {
	var out = make([]string, 0, len(m.Comment))
	for _, comment := range m.Comment {
		comment = strings.TrimSpace(strings.TrimPrefix(comment, "//"))
		if !strings.HasPrefix(comment, "@"+name) {
			continue
		}
		comment = comment[len("@"+name):]
		if len(comment) < 2 {
			continue
		}
		if comment[0] != '(' && comment[len(comment)-1] != ')' {
			return nil, fmt.Errorf("invalid syntax %s", m.Comment)
		}
		comment = strings.TrimSpace(comment[1 : len(comment)-1])
		if len(comment) == 0 {
			continue
		}
		out = append(out, comment)
	}
	return out, nil
}

// OutStructName 最终生成的结构体的名称
func (m *MyMethod) OutStructName() string {
	var structName = m.StructName
	if strings.HasPrefix(structName, "*") {
		structName = structName[1:]
	}
	return fmt.Sprintf("_%sMethod%s", structName, m.MethodName)
}

func parseString(input string) string {
	var out = input
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		out = input[1 : len(input)-1]
	}
	if strings.HasPrefix(out, "*") {
		return out[1:]
	}
	return out
}
