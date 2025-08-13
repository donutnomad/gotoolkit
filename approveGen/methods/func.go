package methods

import (
	"github.com/donutnomad/gotoolkit/internal/utils"
	"regexp"
	"strings"

	"github.com/dave/jennifer/jen"
)

// FuncMethodInfo 存储自定义函数方法的定义信息
type FuncMethodInfo struct {
	Name       string            // 函数名称
	Attributes map[string]string // 自定义属性键值对
	Nest       bool              // 是否嵌套
}

// ParseFuncMethod 解析自定义函数方法的注解
// 例子:
// func:name="approveFor"; module="ABC"; event="EVENT"; nest=true
func ParseFuncMethod(content string) *FuncMethodInfo {
	info := &FuncMethodInfo{
		Attributes: make(map[string]string),
	}

	// 检查是否以func:开头
	if !strings.HasPrefix(content, "func:") {
		return nil
	}

	// 分割分号分隔的部分
	parts := strings.Split(content, ";")

	// 处理第一部分，它应该包含name属性
	firstPart := strings.TrimSpace(parts[0])
	if strings.HasPrefix(firstPart, "func:name=") {
		if m := regexp.MustCompile(`"(.*?)"`).FindStringSubmatch(firstPart); len(m) > 1 {
			info.Name = m[1]
		}
	} else {
		// 尝试解析其他格式的第一部分
		nameMatch := regexp.MustCompile(`func:(.*?)="(.*?)"`).FindStringSubmatch(firstPart)
		if len(nameMatch) > 2 {
			key := strings.TrimSpace(nameMatch[1])
			value := nameMatch[2]
			if key == "name" {
				info.Name = value
			} else {
				info.Attributes[key] = value
			}
		}
	}

	// 处理其余部分
	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// 特殊处理nest=true
		if part == "nest=true" {
			info.Nest = true
			continue
		}

		// 解析键值对
		keyValue := regexp.MustCompile(`(.*?)="(.*?)"`).FindStringSubmatch(part)
		if len(keyValue) > 2 {
			key := strings.TrimSpace(keyValue[1])
			value := keyValue[2]

			// 另一种方式处理nest
			if key == "nest" && value == "true" {
				info.Nest = true
			} else {
				info.Attributes[key] = value
			}
		}
	}

	// 如果没有找到函数名，返回nil
	if info.Name == "" {
		return nil
	}

	return info
}

func (info *FuncMethodInfo) Generator() *FuncMethod {
	return &FuncMethod{
		Info: info,
	}
}

type FuncMethod struct {
	Info     *FuncMethodInfo
	Template string

	Receiver   string // (p *Persion) ==> p
	StructName string // (p *Persion) ==> Persion
	MethodArg  string // Struct Method Args ==> &_AAAeMethodBBBB{}
}

func (m *FuncMethod) Generate(template, receiver, structName, methodName string, methodNames []string, methodArgCode string, methodStructArgCode string, returnArgsCode string) jen.Code {
	m.Receiver = receiver
	m.StructName = structName
	methodName = m.Info.Name + "_" + methodName // ApproveFor_DDD
	m.MethodArg = methodStructArgCode
	m.Template = template

	var code = jen.Empty()

	// 解析返回值类型，分析是否需要处理返回值不匹配的情况
	needsErrorHandling := m.needsReturnValueHandling(returnArgsCode)

	code.Func().Params(jen.Id(receiver).Id(structName)).Id(methodName).Id(methodArgCode).Id(returnArgsCode).BlockFunc(func(group *jen.Group) {
		if needsErrorHandling {
			// 生成错误处理逻辑：调用模板，然后返回默认值和error
			group.Id("err := ").Id(utils.MustExecuteTemplate(m, m.Template))
			group.Return().Id(m.generateDefaultReturnValues(returnArgsCode))
		} else {
			// 原有逻辑：直接返回模板结果
			group.Return().Id(utils.MustExecuteTemplate(m, m.Template))
		}
	}).Line()

	if m.Info.Nest {
		code.Func().Params(jen.Id(receiver).Id(structName)).Id(methodName + "Func").Id(methodArgCode).Id("func() " + returnArgsCode).BlockFunc(func(group *jen.Group) {
			group.Return().Func().Id("()").Id(returnArgsCode).Block(jen.Return().Id(receiver).Dot(methodName).CallFunc(func(g *jen.Group) {
				for _, name := range methodNames {
					g.Id(name)
				}
			}))
		}).Line()
	}

	return code
}

// needsReturnValueHandling 检查是否需要处理返回值不匹配的情况
// 如果返回值包含除了error之外的其他类型，则需要特殊处理
func (m *FuncMethod) needsReturnValueHandling(returnArgsCode string) bool {
	// 简单检查：如果返回值不是只有error，则需要处理
	// 例如：(int64, error) -> true, error -> false, (string, int, error) -> true
	if returnArgsCode == "" || returnArgsCode == "error" {
		return false
	}

	// 移除括号并分割返回值
	returnTypes := m.parseReturnTypes(returnArgsCode)

	// 如果有多个返回值，或者单个返回值不是error，则需要处理
	return len(returnTypes) > 1 || (len(returnTypes) == 1 && returnTypes[0] != "error")
}

// parseReturnTypes 解析返回值类型
func (m *FuncMethod) parseReturnTypes(returnArgsCode string) []string {
	if returnArgsCode == "" {
		return []string{}
	}

	// 移除括号
	returnArgsCode = strings.Trim(returnArgsCode, "()")

	// 简单分割，这里假设没有嵌套的复杂类型
	if returnArgsCode == "" {
		return []string{}
	}

	// 按逗号分割
	parts := strings.Split(returnArgsCode, ",")
	var types []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			types = append(types, part)
		}
	}

	return types
}

// generateDefaultReturnValues 生成默认返回值
func (m *FuncMethod) generateDefaultReturnValues(returnArgsCode string) string {
	returnTypes := m.parseReturnTypes(returnArgsCode)

	if len(returnTypes) == 0 {
		return ""
	}

	var defaultValues []string
	for _, returnType := range returnTypes {
		returnType = strings.TrimSpace(returnType)
		if returnType == "error" {
			defaultValues = append(defaultValues, "err")
		} else {
			// 生成类型的默认值
			defaultValues = append(defaultValues, m.getDefaultValue(returnType))
		}
	}

	return strings.Join(defaultValues, ", ")
}

// getDefaultValue 获取类型的默认值
func (m *FuncMethod) getDefaultValue(typeName string) string {
	typeName = strings.TrimSpace(typeName)

	// 处理指针类型
	if strings.HasPrefix(typeName, "*") {
		return "nil"
	}

	// 处理基本类型
	switch typeName {
	case "int", "int8", "int16", "int32", "int64":
		fallthrough
	case "uint", "uint8", "uint16", "uint32", "uint64":
		fallthrough
	case "float32", "float64":
		fallthrough
	case "byte":
		fallthrough
	case "rune":
		return "0"
	case "string":
		return `""`
	case "bool":
		return "false"
	default:
		// 对于其他类型（如接口、结构体、切片、map等），返回nil
		return "*new(" + typeName + ")"
	}
}
