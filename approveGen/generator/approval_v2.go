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

const methodCallApprovalTemplateV2 = `
func {{.GenMethodName}}(targets []any, ctx context.Context, method string, content string, approved bool) (any, error) {
	param, err := UnmarshalMethodArgs(method, content)
	if err != nil {
		return nil, err
	}
	switch p := param.(type) {
{{- range .Methods}}
	case *{{.OutStructName}}:
		type ApprovedInterface interface {
			{{getApprovedMethodName .}}({{formatMethodSignatureWithReturn . $.GetType}})
		}{{if index $.HookRejectedMap .GenMethod}}
		type RejectedInterface interface {
			{{getRejectedMethodName .}}({{formatMethodSignatureWithReturn . $.GetType}})
		}{{end}}
		for _, t := range targets {
			if approved {
				if target, ok := t.(ApprovedInterface); ok {
					{{formatCallLogic . (getApprovedMethodName .) $.GetType}}
				}
			}{{if index $.HookRejectedMap .GenMethod}} else {
				if target, ok := t.(RejectedInterface); ok {
					{{formatCallLogic . (getRejectedMethodName .) $.GetType}}
				}
			}{{end}}
		}
{{- end}}
	}
	return nil, errors.New("CodeUnknownMethod")
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

func GenMethodCallApprovalV2(genMethodName string, addUnmarshalMethodArgs bool, everyMethodSuffix string, methods []MyMethod, getType func(typ ast.Expr, method MyMethod) string, defaultSuccess bool, hookRejectedMap map[string]bool) types.JenStatementSlice {
	data := GenMethodCallApprovalData{
		GenMethodName:          genMethodName,
		AddUnmarshalMethodArgs: addUnmarshalMethodArgs,
		Methods:                methods,
		EveryMethodSuffix:      everyMethodSuffix,
		DefaultSuccess:         defaultSuccess,
		GetType:                getType,
		HookRejectedMap:        hookRejectedMap,
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
		"formatMethodSignature": func(method MyMethod, getType func(typ ast.Expr, method MyMethod) string) string {
			params := method.AsParams(func(typ ast.Expr) string {
				return getType(typ, method)
			})
			var result []string
			for _, param := range params {
				if param.Type == "context.Context" {
					result = append(result, "ctx context.Context")
				} else {
					result = append(result, fmt.Sprintf("%s %s", param.Name.LowerCamelCase(), param.Type))
				}
			}
			return strings.Join(result, ", ")
		},
		"methodWithSuffix": func(method MyMethod, suffix string) string {
			methodName := method.MethodName
			// 如果方法名以 "HookApproved" 结尾，不添加 suffix
			if strings.HasSuffix(methodName, "HookApproved") {
				return strings.TrimSuffix(methodName, "HookApproved") + suffix
			}
			return methodName + suffix
		},
		"getApprovedMethodName": func(method MyMethod) string {
			return method.MethodName
		},
		"getRejectedMethodName": func(method MyMethod) string {
			methodName := method.MethodName
			// 如果方法名以 "HookApproved" 结尾，将后缀替换为 "HookRejected"
			if strings.HasSuffix(methodName, "HookApproved") {
				return strings.TrimSuffix(methodName, "HookApproved") + "HookRejected"
			}
			return methodName + "HookRejected"
		},
		"formatMethodSignatureWithReturn": func(method MyMethod, getType func(typ ast.Expr, method MyMethod) string) string {
			params := method.AsParams(func(typ ast.Expr) string {
				return getType(typ, method)
			})
			var paramResult []string
			for _, param := range params {
				if param.Type == "context.Context" {
					paramResult = append(paramResult, "ctx context.Context")
				} else {
					paramResult = append(paramResult, fmt.Sprintf("%s %s", param.Name.LowerCamelCase(), param.Type))
				}
			}

			// 处理返回值
			var returnTypes []string
			for _, result := range method.MethodResults {
				returnTypes = append(returnTypes, getType(result.Type, method))
			}

			if len(returnTypes) == 0 {
				return strings.Join(paramResult, ", ")
			} else {
				return strings.Join(paramResult, ", ") + ") (" + strings.Join(returnTypes, ", ")
			}
		},
		"formatCallLogic": func(method MyMethod, methodName string, getType func(typ ast.Expr, method MyMethod) string) string {
			params := method.AsParams(func(typ ast.Expr) string {
				return getType(typ, method)
			})
			var paramNames []string
			for _, param := range params {
				if param.Type == "context.Context" {
					paramNames = append(paramNames, "ctx")
				} else {
					paramNames = append(paramNames, fmt.Sprintf("p.%s", param.Name.UpperCamelCase()))
				}
			}

			// 处理返回值
			var returnTypes []string
			for _, result := range method.MethodResults {
				returnTypes = append(returnTypes, getType(result.Type, method))
			}

			callParams := strings.Join(paramNames, ", ")

			if len(returnTypes) == 0 {
				// 没有返回值
				return fmt.Sprintf(`target.%s(%s)
					return nil, nil`, methodName, callParams)
			} else if len(returnTypes) == 1 {
				if returnTypes[0] == "error" {
					// 只有error返回值
					return fmt.Sprintf(`err := target.%s(%s)
					if err != nil {
						return nil, err
					}
					return nil, nil`, methodName, callParams)
				} else {
					// 只有一个非error返回值
					return fmt.Sprintf(`result := target.%s(%s)
					return result, nil`, methodName, callParams)
				}
			} else {
				// 多个返回值
				lastType := returnTypes[len(returnTypes)-1]
				if lastType == "error" {
					// 最后一个是error
					varNames := make([]string, len(returnTypes))
					for i := 0; i < len(returnTypes)-1; i++ {
						varNames[i] = fmt.Sprintf("v%d", i)
					}
					varNames[len(returnTypes)-1] = "err"

					nonErrorVars := varNames[:len(varNames)-1]
					if len(nonErrorVars) == 1 {
						// 只有一个非error返回值，直接返回
						return fmt.Sprintf(`%s := target.%s(%s)
					if err != nil {
						return nil, err
					}
					return %s, nil`, strings.Join(varNames, ", "), methodName, callParams, nonErrorVars[0])
					} else {
						// 多个非error返回值，使用[]any
						return fmt.Sprintf(`%s := target.%s(%s)
					if err != nil {
						return nil, err
					}
					return []any{%s}, nil`, strings.Join(varNames, ", "), methodName, callParams, strings.Join(nonErrorVars, ", "))
					}
				} else {
					// 没有error
					varNames := make([]string, len(returnTypes))
					for i := 0; i < len(returnTypes); i++ {
						varNames[i] = fmt.Sprintf("v%d", i)
					}
					if len(varNames) == 1 {
						// 只有一个返回值，直接返回
						return fmt.Sprintf(`%s := target.%s(%s)
					return %s, nil`, strings.Join(varNames, ", "), methodName, callParams, varNames[0])
					} else {
						// 多个返回值，使用[]any
						return fmt.Sprintf(`%s := target.%s(%s)
					return []any{%s}, nil`, strings.Join(varNames, ", "), methodName, callParams, strings.Join(varNames, ", "))
					}
				}
			}
		},
	}

	tmpl, err := template.New("methodCallApprovalV2").Funcs(funcMap).Parse(methodCallApprovalTemplateV2)
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
