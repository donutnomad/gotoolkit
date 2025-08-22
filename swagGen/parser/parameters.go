package parsers

import (
	"errors"
	"fmt"
	"strings"
)

// Parameter 用于存储解析后的参数信息
type Parameter struct {
	Name string // 参数名
	Type string // 参数类型
	Tag  string // 注释标签 (如 @PARAM, @FORM, 或 /* */ 中的内容)
}

func (p Parameter) FullName() string {
	return fmt.Sprintf("%s %s", p.Name, p.Type)
}

// splitTopLevel 在顶层按逗号分割参数字符串。
// 它能正确处理嵌套结构，如 "m map[string, int]"，不会在内部逗号处错误分割。
func splitTopLevel(input string) []string {
	var result []string
	var parenLevel int // 用于跟踪括号、方括号等的嵌套层级
	lastSplit := 0

	for i, r := range input {
		switch r {
		case '(', '[', '{':
			parenLevel++
		case ')', ']', '}':
			parenLevel--
		case ',':
			// 只在非嵌套的顶层进行分割
			if parenLevel == 0 {
				result = append(result, strings.TrimSpace(input[lastSplit:i]))
				lastSplit = i + 1
			}
		}
	}
	// 添加最后一个参数
	result = append(result, strings.TrimSpace(input[lastSplit:]))
	return result
}

// ParseParameters 解析函数参数定义字符串
// 支持 // TAG name type, /* TAG */ name type, 以及 name, anotherName type 等格式
// 输入示例: (c *gin.Context, date, filename string)
func ParseParameters(rawInput string) ([]Parameter, error) {
	// 1. 预处理输入字符串
	content := strings.TrimSpace(rawInput)
	if !strings.HasPrefix(content, "(") || !strings.HasSuffix(content, ")") {
		return nil, errors.New("invalid input format: must be wrapped by parentheses '()'")
	}
	content = content[1 : len(content)-1] // 去掉 ( 和 )
	if strings.TrimSpace(content) == "" {
		return []Parameter{}, nil // 处理空参数列表 "()"
	}

	// 2. 使用智能方式按逗号分割
	paramDefs := splitTopLevel(content)

	var result []Parameter
	var nameAccumulator []string // 【核心改动】用于累积等待类型的参数名
	var accumulatedTag string    // 【核心改动】用于存储累积参数组的标签

	// 3. 遍历并解析每一个参数片段
	for _, part := range paramDefs {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		var currentTag string
		var definitionPart string

		// 4. 检查并分离注释
		if strings.HasPrefix(part, "/*") {
			endCommentIndex := strings.Index(part, "*/")
			if endCommentIndex == -1 {
				return nil, fmt.Errorf("parse error: found unclosed block comment '/*' in '%s'", part)
			}
			currentTag = strings.TrimSpace(part[2:endCommentIndex])
			definitionPart = strings.TrimSpace(part[endCommentIndex+2:])
		} else if strings.HasPrefix(part, "//") {
			commentContent := strings.TrimSpace(strings.TrimPrefix(part, "//"))
			fields := strings.Fields(commentContent)
			if len(fields) >= 2 { // 至少要有 TAG 和 name/type
				currentTag = fields[0]
				definitionPart = strings.Join(fields[1:], " ")
			} else {
				definitionPart = part // 格式不符的注释，视为普通定义
			}
		} else {
			currentTag = ""
			definitionPart = part
		}

		// 如果只有注释，没有定义，则将标签暂存，可能会用于下一个参数
		if definitionPart == "" {
			if len(nameAccumulator) == 0 {
				accumulatedTag = currentTag
			}
			continue
		}

		// 5. 【核心逻辑】判断当前片段是否包含类型
		if !strings.Contains(definitionPart, " ") {
			// 情况A: 当前片段只是一个名字 (e.g., "a" from "a, b, c string")
			nameAccumulator = append(nameAccumulator, definitionPart)
			// 如果这是累积的第一个名字，它的标签就是整个组的标签
			if len(nameAccumulator) == 1 {
				accumulatedTag = currentTag
			}
		} else {
			// 情况B: 当前片段包含类型 (e.g., "c string" or "ctx *gin.Context")
			lastSpaceIndex := strings.LastIndex(definitionPart, " ")

			paramType := strings.TrimSpace(definitionPart[lastSpaceIndex+1:])
			namesPart := strings.TrimSpace(definitionPart[:lastSpaceIndex])

			// 将当前片段的名字也加入累加器
			if namesPart != "" {
				currentNames := strings.Split(namesPart, ",")
				for _, n := range currentNames {
					nameAccumulator = append(nameAccumulator, strings.TrimSpace(n))
				}
			}

			if len(nameAccumulator) == 0 {
				return nil, fmt.Errorf("parse error: found type '%s' but no corresponding parameter name in '%s'", paramType, part)
			}

			// 优先使用当前片段的标签，否则使用累积的标签
			finalTag := currentTag
			if finalTag == "" {
				finalTag = accumulatedTag
			}

			// 为所有累积的名字创建 Parameter 对象
			for _, name := range nameAccumulator {
				if name == "" {
					continue
				}
				p := Parameter{
					Name: name,
					Type: paramType,
					Tag:  finalTag,
				}
				result = append(result, p)
			}

			// 清空累加器，为下一组参数做准备
			nameAccumulator = []string{}
			accumulatedTag = ""
		}
	}

	// 6. 检查是否有悬空的参数名（只有名字没有类型）
	if len(nameAccumulator) > 0 {
		return nil, fmt.Errorf("parse error: parameters '%s' missing type definition", strings.Join(nameAccumulator, ", "))
	}

	return result, nil
}
