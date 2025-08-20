package methods

import (
	"fmt"
	"strings"

	"github.com/dave/jennifer/jen"
)

// FormatterMethodInfo 存储Formatter方法的定义信息
type FormatterMethodInfo struct {
	MethodStructs []FormatterMethodStruct // 需要格式化的方法结构体信息
}

// FormatterMethodStruct 单个方法结构体的信息
type FormatterMethodStruct struct {
	StructName string               // 结构体名称，如 _ListingApprovalUsecaseMethodCreateHookApproved
	MethodName string               // 原始方法名，如 Create
	CustomName string               // 自定义名称，如 CreateHookApproved 或默认为 MethodName
	Fields     []FormatterFieldInfo // 结构体中的所有字段信息
}

// FormatterFieldInfo 结构体字段信息
type FormatterFieldInfo struct {
	Name string // 字段名，如 Name, Age
	Type string // 字段类型，如 string, int
}

// FormatterMethod Formatter方法生成器
type FormatterMethod struct {
	Info *FormatterMethodInfo
}

// NewFormatterMethod 创建新的Formatter方法生成器
func NewFormatterMethod() *FormatterMethod {
	return &FormatterMethod{
		Info: &FormatterMethodInfo{
			MethodStructs: []FormatterMethodStruct{},
		},
	}
}

// AddMethod 添加一个需要格式化的方法
func (f *FormatterMethod) AddMethod(structName, methodName, customName string, fields []FormatterFieldInfo) {
	f.Info.MethodStructs = append(f.Info.MethodStructs, FormatterMethodStruct{
		StructName: structName,
		MethodName: methodName,
		CustomName: customName,
		Fields:     fields,
	})
}

// Generate 生成Formatter方法
func (f *FormatterMethod) Generate() jen.Code {
	if len(f.Info.MethodStructs) == 0 {
		return jen.Empty()
	}

	// 按结构体名称分组方法
	structGroups := make(map[string][]FormatterMethodStruct)
	for _, methodStruct := range f.Info.MethodStructs {
		// 提取结构体的基础名称（去掉Method后缀）
		baseName := extractBaseStructName(methodStruct.StructName)
		structGroups[baseName] = append(structGroups[baseName], methodStruct)
	}

	// 生成代码片段
	var statements []jen.Code

	// 为每个结构体生成接口
	for baseName, methods := range structGroups {
		interfaceName := fmt.Sprintf("%sFormatterInterface", baseName)

		// 收集唯一的方法名
		uniqueMethods := make(map[string]bool)
		var uniqueMethodsList []string

		for _, methodStruct := range methods {
			formatMethodName := methodStruct.CustomName
			// 如果CustomName不是以Format开头，则添加Format前缀
			if !strings.HasPrefix(formatMethodName, "Format") {
				formatMethodName = fmt.Sprintf("Format%s", formatMethodName)
			}

			if !uniqueMethods[formatMethodName] {
				uniqueMethods[formatMethodName] = true
				uniqueMethodsList = append(uniqueMethodsList, formatMethodName)
			}
		}

		// 生成接口定义
		interfaceStmt := jen.Type().Id(interfaceName).InterfaceFunc(func(interfaceGroup *jen.Group) {
			for _, methodName := range uniqueMethodsList {
				// 为每个方法找到对应的字段信息（取第一个匹配的方法）
				var methodFields []FormatterFieldInfo
				for _, methodStruct := range methods {
					currentFormatMethodName := methodStruct.CustomName
					if !strings.HasPrefix(currentFormatMethodName, "Format") {
						currentFormatMethodName = fmt.Sprintf("Format%s", currentFormatMethodName)
					}
					if currentFormatMethodName == methodName {
						methodFields = methodStruct.Fields
						break
					}
				}

				// 生成方法签名：FormatXXX(ctx context.Context, [字段参数...], raw any) (any, error)
				interfaceGroup.Id(methodName).ParamsFunc(func(paramGroup *jen.Group) {
					// 添加context参数
					paramGroup.Id("ctx").Qual("", "context.Context")

					// 添加结构体字段参数
					for _, field := range methodFields {
						paramGroup.Id(strings.ToLower(field.Name)).Id(field.Type)
					}

					// 添加raw参数
					paramGroup.Id("raw").Any()
				}).Params(
					jen.Any(),
					jen.Error(),
				)
			}
		}).Line()

		statements = append(statements, interfaceStmt)
	}

	// 生成Format函数
	formatFunc := jen.Func().Id("Format").Params(
		jen.Id("ctx").Qual("", "context.Context"),
		jen.Id("arg").Qual("", "any"),
		jen.Id("formatter").Any(),
	).Params(
		jen.Any(),
		jen.Error(),
	).BlockFunc(func(g *jen.Group) {
		// 生成 switch 语句
		g.Switch(jen.Id("v").Op(":=").Id("arg").Op(".").Params(jen.Id("type"))).BlockFunc(func(switchGroup *jen.Group) {
			// 为每个方法结构体生成一个 case
			for _, methodStruct := range f.Info.MethodStructs {
				formatMethodName := methodStruct.CustomName
				// 如果CustomName不是以Format开头，则添加Format前缀
				if !strings.HasPrefix(formatMethodName, "Format") {
					formatMethodName = fmt.Sprintf("Format%s", formatMethodName)
				}

				switchGroup.Case(jen.Op("*").Id(methodStruct.StructName)).BlockFunc(func(caseGroup *jen.Group) {
					// 生成接口定义
					interfaceName := fmt.Sprintf("%s_Interface", methodStruct.StructName)
					caseGroup.Type().Id(interfaceName).Interface(
						jen.Id(formatMethodName).ParamsFunc(func(paramGroup *jen.Group) {
							// 添加context参数
							paramGroup.Id("ctx").Qual("", "context.Context")

							// 添加结构体字段参数
							for _, field := range methodStruct.Fields {
								paramGroup.Id(strings.ToLower(field.Name)).Id(field.Type)
							}

							// 添加raw参数
							paramGroup.Id("raw").Any()
						}).Params(
							jen.Any(),
							jen.Error(),
						),
					)

					// 生成类型断言和调用
					caseGroup.If(
						jen.List(jen.Id("target"), jen.Id("ok")).Op(":=").Id("formatter").Op(".").Params(jen.Id(interfaceName)),
						jen.Id("ok"),
					).BlockFunc(func(ifGroup *jen.Group) {
						// 生成方法调用参数
						ifGroup.Return(
							jen.Id("target").Dot(formatMethodName).CallFunc(func(callGroup *jen.Group) {
								// 添加context参数
								callGroup.Id("ctx")

								// 添加结构体字段值
								for _, field := range methodStruct.Fields {
									callGroup.Id("v").Dot(field.Name)
								}

								// 添加raw参数
								callGroup.Id("v")
							}),
						)
					})
				})
			}
		})

		// 在最后统一返回错误
		g.Return(jen.Nil(), jen.Qual("", "errors.New").Call(jen.Lit("NoFormatter")))
	}).Line()

	statements = append(statements, formatFunc)

	// 合并所有代码片段
	var result jen.Code
	for i, stmt := range statements {
		if i == 0 {
			result = stmt
		} else {
			result = jen.Add(result, stmt)
		}
	}

	return result
}

// extractBaseStructName 提取结构体的基础名称
// 例如：_TestStructMethodCreate -> TestStruct
func extractBaseStructName(structName string) string {
	// 移除开头的下划线
	if strings.HasPrefix(structName, "_") {
		structName = structName[1:]
	}

	// 查找Method关键字并截取之前的部分
	if idx := strings.Index(structName, "Method"); idx != -1 {
		return structName[:idx]
	}

	return structName
}
