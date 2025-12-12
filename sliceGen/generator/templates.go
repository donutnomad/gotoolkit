package generator

import (
	"io"
	"text/template"
)

// IExecute 模板执行接口
type IExecute interface {
	Execute(w io.Writer, data any, addImport func(path string)) error
	Comment() string
}

// MethodTemplateData 定义模板数据结构
type MethodTemplateData struct {
	TypeName     string // 切片类型名称 (例如: UserSlice)
	TypeItemName string // 元素类型名称 (例如: User)
	Description  string // 方法描述
	UsePointer   bool   // 是否使用指针
	PtrPrefix    string // 指针前缀 ("*" 或 "")
}

// FieldTemplateData 定义字段模板数据结构
type FieldTemplateData struct {
	TypeName     string // 切片类型名称 (例如: UserSlice)
	TypeItemName string // 元素类型名称 (例如: User)
	FieldName    string // 字段名称
	FieldType    string // 字段类型
	UsePointer   bool   // 是否使用指针
	PtrPrefix    string // 指针前缀 ("*" 或 "")
}

// MyGenerator 通用模板生成器
type MyGenerator[T any] struct {
	Name        string
	Description string
	Template    string
	Imports     []string
	_template   *template.Template
}

func (g *MyGenerator[T]) Comment() string {
	return g.Description
}

func (g *MyGenerator[T]) Generate(w io.Writer, data T, addImport func(path string)) error {
	return g.Execute(w, data, addImport)
}

func (g *MyGenerator[T]) Execute(w io.Writer, data any, addImport func(path string)) error {
	if g._template == nil {
		mapperTmpl, err := template.New("").Parse(g.Template)
		if err != nil {
			return err
		}
		g._template = mapperTmpl
	}
	for _, imp := range g.Imports {
		addImport(imp)
	}
	return g._template.Execute(w, data)
}

// MethodMapField 字段映射方法模板
var MethodMapField = MyGenerator[FieldTemplateData]{
	Template: `
// Map{{.FieldName}} is a mapper function for field {{.FieldName}}
func (s {{.PtrPrefix}}{{.TypeItemName}}) Map{{.FieldName}}(item {{.PtrPrefix}}{{.TypeItemName}}, index int) {{.FieldType}} {
	return item.{{.FieldName}}
}
`,
}

// MethodField 字段提取方法模板
var MethodField = MyGenerator[FieldTemplateData]{
	Template: `
// {{.FieldName}} returns a slice of {{.FieldName}} field values
func (s {{.TypeName}}) {{.FieldName}}() []{{.FieldType}} {
{{- if .UsePointer}}
	return lo.Map(s, func(item {{.PtrPrefix}}{{.TypeItemName}}, index int) {{.FieldType}} {
		return item.{{.FieldName}}
	})
{{- else}}
	return lo.Map(s, {{.TypeItemName}}{}.Map{{.FieldName}})
{{- end}}
}
`,
}

// MethodFilter 过滤方法模板
var MethodFilter = MyGenerator[MethodTemplateData]{
	Name:        "filter",
	Description: "returns a new slice containing only the elements that satisfy the predicate fn",
	Template: `
// Filter {{.Description}}
func (s {{.TypeName}}) Filter(fn func({{.PtrPrefix}}{{.TypeItemName}}) bool) {{.TypeName}} {
	return lo.Filter(s, func(item {{.PtrPrefix}}{{.TypeItemName}}, _ int) bool {
		return fn(item)
	})
}`,
}

// MethodMap 映射转换方法模板
var MethodMap = MyGenerator[MethodTemplateData]{
	Name:        "map",
	Description: "transforms each element using the provided function fn",
	Template: `
// Map {{.Description}}
func (s {{.TypeName}}) Map(fn func({{.PtrPrefix}}{{.TypeItemName}}) any) []any {
	return lo.Map(s, func(item {{.PtrPrefix}}{{.TypeItemName}}, _ int) any {
		return fn(item)
	})
}`,
}

// MethodReduce 归约方法模板
var MethodReduce = MyGenerator[MethodTemplateData]{
	Name:        "reduce",
	Description: "reduces the slice to a single value using the provided function fn",
	Template: `
// Reduce {{.Description}}
func (s {{.TypeName}}) Reduce(fn func(acc, curr {{.PtrPrefix}}{{.TypeItemName}}) {{.PtrPrefix}}{{.TypeItemName}}, initial {{.PtrPrefix}}{{.TypeItemName}}) {{.PtrPrefix}}{{.TypeItemName}} {
	return lo.Reduce(s, func(acc {{.PtrPrefix}}{{.TypeItemName}}, item {{.PtrPrefix}}{{.TypeItemName}}, _ int) {{.PtrPrefix}}{{.TypeItemName}} {
		return fn(acc, item)
	}, initial)
}`,
}

// MethodSort 排序方法模板
var MethodSort = MyGenerator[MethodTemplateData]{
	Name:        "sort",
	Description: "returns a new sorted slice using the provided less function",
	Imports:     []string{"sort"},
	Template: `
// Sort {{.Description}}
func (s {{.TypeName}}) Sort(less func({{.PtrPrefix}}{{.TypeItemName}}, {{.PtrPrefix}}{{.TypeItemName}}) bool) {{.TypeName}} {
	result := append({{.TypeName}}{}, s...)
	sort.Slice(result, func(i, j int) bool {
		return less(result[i], result[j])
	})
	return result
}`,
}

// MethodGroupBy 分组方法模板
var MethodGroupBy = MyGenerator[MethodTemplateData]{
	Name:        "groupBy",
	Description: "groups elements by the key returned by the fn function",
	Template: `
// GroupBy {{.Description}}
func (s {{.TypeName}}) GroupBy(fn func({{.PtrPrefix}}{{.TypeItemName}}) string) map[string]{{.TypeName}} {
	return lo.GroupBy(s, func(item {{.PtrPrefix}}{{.TypeItemName}}) string {
		return fn(item)
	})
}`,
}
