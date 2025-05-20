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

	code.Func().Params(jen.Id(receiver).Id(structName)).Id(methodName).Id(methodArgCode).Id(returnArgsCode).BlockFunc(func(group *jen.Group) {
		group.Return().Id(utils.MustExecuteTemplate(m, m.Template))
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
