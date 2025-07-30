package methods

import (
	"github.com/dave/jennifer/jen"
	"github.com/donutnomad/gotoolkit/internal/utils"
	"github.com/samber/lo"
	"regexp"
	"strings"
)

// JsonMethodInfo 存储Json方法的定义信息
type JsonMethodInfo struct {
	InterfaceCheck bool // 是否需要生成接口检查
}

// ParseJsonMethod 解析Json方法的注解
// 例子: args::json
func ParseJsonMethod(content string) *JsonMethodInfo {
	info := &JsonMethodInfo{
		InterfaceCheck: false,
	}

	// 检查是否包含args::json
	if !strings.Contains(content, "args::json") {
		return nil
	}

	// 简单解析，如果找到args::json则启用接口检查
	if regexp.MustCompile(`args::json`).MatchString(content) {
		info.InterfaceCheck = true
	}

	return info
}

func (info *JsonMethodInfo) Generator() *JsonMethod {
	return &JsonMethod{
		info: info,
	}
}

type JsonMethod struct {
	info *JsonMethodInfo

	Receiver   string
	StructName string
	MethodName string
}

func (m *JsonMethod) WithMethod(name string) *JsonMethod {
	m.MethodName = name
	return m
}

func (m *JsonMethod) Generate(receiver, structName string) jen.Code {
	m.Receiver = receiver
	m.StructName = structName
	if m.MethodName == "" {
		m.MethodName = "Json"
	}
	return jen.Id(lo.Must1(m.generate())).Line()
}

func (m *JsonMethod) generate() (string, error) {
	var template string
	if m.info.InterfaceCheck {
		// 生成接口检查器
		template = `var _ interface { Json() (any, error) } = (*{{.StructName}})(nil)`
		return utils.ExecuteTemplate(m, template)
	}
	return "", nil
}
