package generator

import (
	"bytes"
	"fmt"
	"github.com/dave/jennifer/jen"
	"github.com/donutnomad/gotoolkit/approveGen/types"
	"github.com/donutnomad/gotoolkit/approveGen/utils"
	"go/ast"
	"strings"
	"text/template"
)

// 方法调用审批生成器的模板
const methodCallApprovalTemplate = `
func {{.GenMethodName}}(a *AllServices, ctx context.Context, method string, content string) BaseResponse[any] {
	switch method {
{{- range .Methods}}
	case "{{.GenMethod}}":
		var p {{.OutStructName}}
		if err := sonic.Unmarshal([]byte(content), &p); err != nil {
			return Fail[any]("CodeUnmarshalFailed")
		}
		return a.{{nameWithoutPoint .StructName}}.{{.MethodName}}{{$.EveryMethodSuffix}}({{formatParams . $.GetType}}).ToAny()
{{- end}}
	default:
{{- if .DefaultSuccess}}
		return Success[any](struct{}{})
{{- else}}
		return Fail[any]("CodeUnknownMethod")
{{- end}}
	}
}

{{- if .AddUnmarshalMethodArgs}}

func UnmarshalMethodArgs(method string, content string) (any, error) {
	switch method {
{{- range .Methods}}
	case "{{.GenMethod}}":
		var p {{.OutStructName}}
		if err := sonic.Unmarshal([]byte(content), &p); err != nil {
			return nil, err
		}
		return &p, nil
{{- end}}
	default:
		return nil, nil
	}
}
{{- end}}
`

func GenMethodCallApproval(genMethodName string, addUnmarshalMethodArgs bool, everyMethodSuffix string, methods []MyMethod, getType func(typ ast.Expr, method MyMethod) string, defaultSuccess bool) types.JenStatementSlice {
	data := GenMethodCallApprovalData{
		GenMethodName:          genMethodName,
		AddUnmarshalMethodArgs: addUnmarshalMethodArgs,
		Methods:                methods,
		EveryMethodSuffix:      everyMethodSuffix,
		DefaultSuccess:         defaultSuccess,
		GetType:                getType,
	}

	// 创建模板函数
	funcMap := template.FuncMap{
		"nameWithoutPoint": utils.NameWithoutPoint,
		"formatParams": func(method MyMethod, getType func(typ ast.Expr, method MyMethod) string) string {
			params := method.AsParams(func(typ ast.Expr) string {
				return getType(typ, method)
			})
			var result []string
			for _, param := range params {
				if param.Type == "context.Context" {
					result = append(result, "ctx")
				} else {
					result = append(result, fmt.Sprintf("p.%s", param.Name.UpperCamelCase()))
				}
			}
			return strings.Join(result, ", ")
		},
	}

	tmpl, err := template.New("methodCallApproval").Funcs(funcMap).Parse(methodCallApprovalTemplate)
	if err != nil {
		panic(fmt.Sprintf("解析模板失败: %v", err))
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic(fmt.Sprintf("执行模板失败: %v", err))
	}

	// 将生成的代码转换为 jen.Statement
	code := jen.Id(buf.String())
	return types.JenStatementSlice{code}
}
