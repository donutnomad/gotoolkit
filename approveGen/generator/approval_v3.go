package generator

import (
	"bytes"
	"fmt"
	"go/ast"
	"regexp"
	"strings"
	"text/template"

	"github.com/dave/jennifer/jen"
	"github.com/donutnomad/gotoolkit/approveGen/types"
	"github.com/donutnomad/gotoolkit/approveGen/utils"
)

const methodCallApprovalTemplateV3 = `
type ApprovalCaller struct {
	targets []any
	formatter IApprovalFormatter
}

func newApprovalCaller(formatter IApprovalFormatter, targets ...any) *ApprovalCaller {
    return &ApprovalCaller{targets: targets, formatter: formatter}
}

func (amc *ApprovalCaller) {{.GenMethodName}}(ctx context.Context, arg any, approved bool) (any, error) {
	switch p := arg.(type) {
{{- range .Methods}}
	case *{{.OutStructName}}:
		type ApprovedInterface interface {
			{{getApprovedMethodName .}}({{formatMethodSignatureWithReturn . $.GetType}})
		}{{if index $.HookRejectedMap .GenMethod}}
		type RejectedInterface interface {
			{{getRejectedMethodName .}}({{formatMethodSignatureWithReturn . $.GetType}})
		}{{end}}
		for _, t := range amc.targets {
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

func (amc *ApprovalCaller) UnmarshalMethodArgs(method string, content string) (any, error) {
	switch method {
{{- range .Methods}}
	case "{{genMethodName .}}":
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

func (amc *ApprovalCaller) Format(ctx context.Context, arg any) (any, error) {
{{- $hasAnyFormatter := false -}}
{{- range .Methods -}}
{{- if hasFormatterMethod . -}}
{{- $hasAnyFormatter = true -}}
{{- end -}}
{{- end -}}
{{- if not $hasAnyFormatter}}
	return "", nil
{{- else}}
	switch v := arg.(type) {
{{- range .Methods}}
{{- $method := . -}}
{{- if hasFormatterMethod $method}}
	case *{{.OutStructName}}:
		return amc.formatter.{{getFormatterMethodName .}}({{formatFormatterCallParams . $.GetType}})
{{- end}}
{{- end}}
	}
	return nil, errors.New("NoFormatter")
{{- end}}
}

type IApprovalFormatter interface {
{{- $structFormatterMethods := groupFormatterMethodsByStruct .Methods $.GetType -}}
{{range $structName, $methods := $structFormatterMethods}}
{{- range $methods}}
	{{.MethodName}}({{.Signature}})
{{- end}}
{{- end}}
}

`

func GenMethodCallApprovalV3(genMethodName string, addUnmarshalMethodArgs bool, pkgName string, everyMethodSuffix string, methods []MyMethod, getType func(typ ast.Expr, method MyMethod) string, defaultSuccess bool, hookRejectedMap map[string]bool) types.JenStatementSlice {
	// 计算 FormatterInterfaces
	formatterInterfaces := make(map[string]bool)
	formatterInterfaces["IApprovalFormatter"] = true

	data := GenMethodCallApprovalDataV3{
		GenMethodName:          genMethodName,
		AddUnmarshalMethodArgs: addUnmarshalMethodArgs,
		PkgName:                pkgName,
		Methods:                methods,
		EveryMethodSuffix:      everyMethodSuffix,
		DefaultSuccess:         defaultSuccess,
		GetType:                getType,
		HookRejectedMap:        hookRejectedMap,
		FormatterInterfaces:    formatterInterfaces,
	}

	// 创建模板函数
	funcMap := template.FuncMap{
		"nameWithoutPoint": utils.NameWithoutPoint,
		"groupFormatterMethodsByStruct": func(methods []MyMethod, getType func(typ ast.Expr, method MyMethod) string) map[string][]FormatterMethod {
			structMethods := make(map[string][]FormatterMethod)

			for _, method := range methods {
				bodies, err := method.FindAnnoBody("Approve")
				if err != nil {
					continue
				}

				// 检查是否有formatter参数
				hasFormatter := false
				for _, body := range bodies {
					if strings.Contains(body, "formatter") {
						hasFormatter = true
						break
					}
				}
				if !hasFormatter {
					continue
				}

				// 获取formatter方法名
				var formatterName string
				for _, content := range bodies {
					if strings.Contains(content, "args::formatter") {
						// 首先尝试匹配带引号的格式: args::formatter="value"
						if m := regexp.MustCompile(`args::formatter\s*=\s*"(.*?)"`).FindStringSubmatch(content); len(m) > 1 {
							formatterName = m[1]
							break
						}
						// 然后尝试匹配不带引号的格式: args::formatter= value 或 args::formatter=value
						if m := regexp.MustCompile(`args::formatter\s*=\s*([^;\s)]+)`).FindStringSubmatch(content); len(m) > 1 {
							formatterName = strings.TrimSpace(m[1])
							break
						}
						// 如果只有 args::formatter 没有值，使用默认值
						if strings.Contains(content, "args::formatter") && !strings.Contains(content, "=") {
							formatterName = "Format" + method.MethodName
							break
						}
					}
				}
				if formatterName == "" {
					formatterName = "Format" + method.MethodName
				}

				// 生成方法签名
				params := method.AsParams(func(typ ast.Expr) string {
					return getType(typ, method)
				})
				var paramResult []string
				paramResult = append(paramResult, "ctx context.Context")

				// 添加除context.Context之外的所有参数
				for _, param := range params {
					if param.Type != "context.Context" {
						paramResult = append(paramResult, fmt.Sprintf("%s %s", param.Name.LowerCamelCase(), param.Type))
					}
				}

				// 添加raw any参数
				paramResult = append(paramResult, "raw any")
				signature := strings.Join(paramResult, ", ") + ") (any, error"

				structName := method.StructNameWithoutPtr()
				formatterMethod := FormatterMethod{
					MethodName: formatterName,
					Signature:  signature,
				}

				// 检查是否已存在相同的方法名（不检查签名，因为Go不允许同名方法）
				found := false
				for _, existing := range structMethods[structName] {
					if existing.MethodName == formatterMethod.MethodName {
						found = true
						break
					}
				}

				if !found {
					structMethods[structName] = append(structMethods[structName], formatterMethod)
				}
			}

			return structMethods
		},
		"genMethodName": func(method MyMethod) string {
			if pkgName != "" {
				return fmt.Sprintf("%s_%s_%s", pkgName, method.StructNameWithoutPtr(), method.MethodName)
			}
			return fmt.Sprintf("%s_%s", method.StructNameWithoutPtr(), method.MethodName)
		},
		"hasFormatterMethod": func(method MyMethod) bool {
			// 检查方法的注释中是否有formatter相关的配置
			bodies, err := method.FindAnnoBody("Approve")
			if err != nil {
				return false
			}
			for _, body := range bodies {
				if strings.Contains(body, "formatter") {
					return true
				}
			}
			return false
		},
		"getFormatterMethodName": func(method MyMethod) string {
			// 解析注释中指定的formatter方法名
			bodies, err := method.FindAnnoBody("Approve")
			if err != nil {
				return "Format" + method.MethodName
			}

			// 需要导入methods包来使用ParseFormatterMethod
			// 这里我们需要直接实现解析逻辑
			for _, content := range bodies {
				if strings.Contains(content, "args::formatter") {
					// 首先尝试匹配带引号的格式: args::formatter="value"
					if m := regexp.MustCompile(`args::formatter\s*=\s*"(.*?)"`).FindStringSubmatch(content); len(m) > 1 {
						return m[1]
					}
					// 然后尝试匹配不带引号的格式: args::formatter= value 或 args::formatter=value
					if m := regexp.MustCompile(`args::formatter\s*=\s*([^;\s)]+)`).FindStringSubmatch(content); len(m) > 1 {
						return strings.TrimSpace(m[1])
					}
					// 如果只有 args::formatter 没有值，使用默认值
					if strings.Contains(content, "args::formatter") && !strings.Contains(content, "=") {
						return "Format" + method.MethodName
					}
				}
			}
			return "Format" + method.MethodName
		},
		"formatFormatterSignature": func(method MyMethod, getType func(typ ast.Expr, method MyMethod) string) string {
			params := method.AsParams(func(typ ast.Expr) string {
				return getType(typ, method)
			})
			var paramResult []string
			paramResult = append(paramResult, "ctx context.Context")

			// 添加除context.Context之外的所有参数
			for _, param := range params {
				if param.Type != "context.Context" {
					paramResult = append(paramResult, fmt.Sprintf("%s %s", param.Name.LowerCamelCase(), param.Type))
				}
			}

			// 添加raw any参数
			paramResult = append(paramResult, "raw any")

			return strings.Join(paramResult, ", ") + ") (any, error"
		},
		"formatFormatterCallParams": func(method MyMethod, getType func(typ ast.Expr, method MyMethod) string) string {
			params := method.AsParams(func(typ ast.Expr) string {
				return getType(typ, method)
			})
			var paramNames []string
			paramNames = append(paramNames, "ctx")

			// 添加除context.Context之外的所有参数
			for _, param := range params {
				if param.Type != "context.Context" {
					paramNames = append(paramNames, fmt.Sprintf("v.%s", param.Name.UpperCamelCase()))
				}
			}

			// 添加raw参数（传入v本身）
			paramNames = append(paramNames, "v")

			return strings.Join(paramNames, ", ")
		},
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

	tmpl, err := template.New("methodCallApprovalV3").Funcs(funcMap).Parse(methodCallApprovalTemplateV3)
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

type FormatterMethod struct {
	MethodName string
	Signature  string
}

type GenMethodCallApprovalDataV3 struct {
	GenMethodName          string
	AddUnmarshalMethodArgs bool
	PkgName                string
	Methods                []MyMethod
	EveryMethodSuffix      string
	DefaultSuccess         bool
	GetType                func(typ ast.Expr, method MyMethod) string
	HookRejectedMap        map[string]bool
	FormatterInterfaces    map[string]bool
}
