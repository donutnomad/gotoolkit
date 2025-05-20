package methods

import (
	"github.com/dave/jennifer/jen"
	"github.com/donutnomad/gotoolkit/internal/utils"
	"github.com/samber/lo"
	"regexp"
	"strings"
)

// StringMethodInfo 存储String方法的定义信息
type StringMethodInfo struct {
	ArgsTemplate  string   // 参数格式化模板，如 "$key=$value"
	Separator     string   // 分隔符
	IncludeFields []string // 需要包含的字段（优先级高于exclude）
	ExcludeFields []string // 需要排除的字段
}

// ParseStringMethod 解析String方法的注解
// 例子:
// args::string="$key==>$key, 索引$idx==>$value"; sep="|| "; exclude="email,roles"; include="rawID"
func ParseStringMethod(content string) *StringMethodInfo {
	info := &StringMethodInfo{
		Separator: ", ",
	}

	// 分割分号分隔的部分
	for idx, part := range strings.Split(content, ";") {
		part = strings.TrimSpace(part)

		if idx == 0 && !strings.HasPrefix(part, "args::string=") {
			return nil
		}

		switch {
		// 解析参数模板
		case strings.HasPrefix(part, "args::string="):
			t := part[len("args::string="):]
			if strings.Count(t, "\"") != 2 {
				panic("unknown field: " + part + ", use ;")
			}
			t = t[1 : len(t)-1]
			info.ArgsTemplate = t
		case strings.HasPrefix(part, "include="):
			value := strings.TrimPrefix(part, "include=")
			value = strings.Trim(value, `"`)
			if value != "" {
				info.IncludeFields = strings.Split(value, ",")
				for i, field := range info.IncludeFields {
					info.IncludeFields[i] = strings.TrimSpace(field)
				}
				// 当include存在时，清空exclude
				info.ExcludeFields = nil
			}

		// 解析排除字段
		case strings.HasPrefix(part, "exclude="):
			value := strings.TrimPrefix(part, "exclude=")
			value = strings.Trim(value, `"`)
			if value != "" {
				info.ExcludeFields = strings.Split(value, ",")
				// 清理每个字段的空白
				for i, field := range info.ExcludeFields {
					info.ExcludeFields[i] = strings.TrimSpace(field)
				}
			}
		// 解析分隔符
		case strings.HasPrefix(part, "sep="):
			if m := regexp.MustCompile(`"(.*?)"`).FindStringSubmatch(part); len(m) > 1 {
				info.Separator = m[1]
			}
		default:
			panic("unknown field: " + part)
		}
	}
	if info.ArgsTemplate == "" {
		return nil
	}
	return info
}

func (info *StringMethodInfo) Generator() *StringMethod {
	return &StringMethod{
		info: info,
	}
}

type StringMethod struct {
	info *StringMethodInfo

	Receiver   string
	StructName string
	MethodName string
	ArgsSep    string
	Args       []ArgInfo
}

type ArgInfo struct {
	Template   string // 格式化模板
	Field      string // 字段名称
	FormatFunc string // 为Field格式化使用, 例如将uid ==> 转换为email调用方法uinfo(uid)
	IsPtr      bool   // 字段类型是否是指针
	IsMoOption bool   // 字段类型是否是mo.Option[x]
}

func (m *StringMethod) WithMethod(name string) *StringMethod {
	m.MethodName = name
	return m
}

func (m *StringMethod) Generate(receiver, structName string, args []ArgInfo) jen.Code {
	m.Receiver = receiver
	m.StructName = structName
	m.ArgsSep = m.info.Separator
	m.Args = lo.Filter(args, func(item ArgInfo, index int) bool {
		name_ := strings.ToLower(item.Field[:1]) + item.Field[1:]
		if len(m.info.IncludeFields) > 0 {
			return lo.Contains(m.info.IncludeFields, name_)
		}
		return !lo.Contains(m.info.ExcludeFields, name_)
	})
	if m.MethodName == "" {
		m.MethodName = "String"
	}
	return jen.Id(lo.Must1(m.generate())).Line()
}

func (m *StringMethod) generate() (string, error) {
	return utils.ExecuteTemplate(m,
		`
func ({{.Receiver}} *{{.StructName}}) {{.MethodName}}() string {
    ss := make([]string, 0, {{len .Args}})
    {{- range $idx, $arg := .Args}}
		{{- if eq $arg.FormatFunc "" -}}  
			{{- if $arg.IsPtr}}
				if {{$.Receiver}}.{{$arg.Field}} != nil {
					ss = append(ss, fmt.Sprintf("{{$arg.Template}}", *{{$.Receiver}}.{{$arg.Field}}))
				}
			{{- else}}
				ss = append(ss, fmt.Sprintf("{{$arg.Template}}", {{$.Receiver}}.{{$arg.Field}}))
			{{- end}}
		{{- else -}}  
			{{- if $arg.IsPtr}}
				if {{$.Receiver}}.{{$arg.Field}} != nil {
					ss = append(ss, fmt.Sprintf("{{$arg.Template}}", {{$arg.FormatFunc}}(*{{$.Receiver}}.{{$arg.Field}})))
				}
			{{- else}}
				ss = append(ss, fmt.Sprintf("{{$arg.Template}}", {{$arg.FormatFunc}}({{$.Receiver}}.{{$arg.Field}})))
			{{- end}}
		{{- end -}}
    {{- end}}
    return strings.Join(ss, "{{.ArgsSep}}")
}
`)
}
