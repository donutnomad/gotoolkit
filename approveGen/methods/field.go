package methods

import (
	"github.com/donutnomad/gotoolkit/internal/utils"
	"regexp"
	"strings"
)

// FieldInfo stores field parsing information
type FieldInfo struct {
	Field     string // 原始字段名
	Function  string // 格式化方法
	Alias     string // 字段别名
	Formatter string // formatter方法名
}

type FieldInfoSlice []FieldInfo

func (s FieldInfoSlice) GetName(name utils.EString) utils.EString {
	for _, field := range s {
		if field.Field == name.String() && field.Alias != "" {
			return utils.EString(field.Alias)
		}
	}
	return name
}

func (s FieldInfoSlice) GetFormatter(name utils.EString) utils.EString {
	for _, field := range s {
		if field.Field == name.String() && field.Formatter != "" {
			return utils.EString(field.Formatter)
		}
	}
	return ""
}

func (s FieldInfoSlice) GetFunction(name utils.EString) utils.EString {
	for _, field := range s {
		if field.Field == name.String() && field.Function != "" {
			return utils.EString(field.Function)
		}
	}
	return ""
}

// ParseFieldMethod parses field method annotation
// Example: args::field="rawID"; func="格式化的方法"; alias="别名"; args::formatter="CreateHookRejected"
func ParseFieldMethod(contents []string) FieldInfoSlice {
	var infos []FieldInfo

	for _, content := range contents {
		info := FieldInfo{}

		// 分割分号分隔的部分
		for _, part := range strings.Split(content, ";") {
			part = strings.TrimSpace(part)

			switch {
			// 解析字段名
			case strings.HasPrefix(part, "args::field="):
				if m := regexp.MustCompile(`"(.*?)"`).FindStringSubmatch(part); len(m) > 1 {
					info.Field = m[1]
				}

			// 解析格式化方法
			case strings.HasPrefix(part, "func="):
				if m := regexp.MustCompile(`"(.*?)"`).FindStringSubmatch(part); len(m) > 1 {
					info.Function = m[1]
				}

			// 解析别名
			case strings.HasPrefix(part, "alias="):
				if m := regexp.MustCompile(`"(.*?)"`).FindStringSubmatch(part); len(m) > 1 {
					info.Alias = m[1]
				}

			// 解析formatter
			case strings.HasPrefix(part, "args::formatter="):
				if m := regexp.MustCompile(`"(.*?)"`).FindStringSubmatch(part); len(m) > 1 {
					info.Formatter = m[1]
				}
			}
		}

		// 如果有有效的字段名，添加到结果中
		if info.Field != "" {
			infos = append(infos, info)
		}
	}

	return infos
}

// ParseFormatterMethod parses formatter method annotation
// Example:
//
//	args::formatter="CreateHookRejected"
//	args::formatter= CreateHookRejected
//	args::formatter=CreateHookRejected
func ParseFormatterMethod(contents []string) string {
	for _, content := range contents {
		if strings.Contains(content, "args::formatter") {
			// 首先尝试匹配带引号的格式: args::formatter="value"
			if m := regexp.MustCompile(`args::formatter\s*=\s*"(.*?)"`).FindStringSubmatch(content); len(m) > 1 {
				return m[1]
			}
			// 然后尝试匹配不带引号的格式: args::formatter= value 或 args::formatter=value
			if m := regexp.MustCompile(`args::formatter\s*=\s*([^;\s)]+)`).FindStringSubmatch(content); len(m) > 1 {
				return strings.TrimSpace(m[1])
			}
			// 如果只有 args::formatter 没有值，返回DEFAULT表示需要使用默认值
			if strings.Contains(content, "args::formatter") && !strings.Contains(content, "=") {
				return "DEFAULT"
			}
		}
	}
	return ""
}
