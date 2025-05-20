package methods

import (
	"github.com/donutnomad/gotoolkit/internal/utils"
	"regexp"
	"strings"
)

// FieldInfo stores field parsing information
type FieldInfo struct {
	Field    string // 原始字段名
	Function string // 格式化方法
	Alias    string // 字段别名
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

func (s FieldInfoSlice) GetFunction(name utils.EString) utils.EString {
	for _, field := range s {
		if field.Field == name.String() && field.Function != "" {
			return utils.EString(field.Function)
		}
	}
	return ""
}

// ParseFieldMethod parses field method annotation
// Example: args::field="rawID"; func="格式化的方法"; alias="别名"
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
			}
		}

		// 如果有有效的字段名，添加到结果中
		if info.Field != "" {
			infos = append(infos, info)
		}
	}

	return infos
}
