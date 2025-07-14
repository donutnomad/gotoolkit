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

// ParseParameters 解析函数参数定义字符串
// 输入示例: ( c *gin.Context, // @PARAM id string, // @FORM data SendOTPReq, )
// ParseParameters 解析函数参数定义字符串
// 支持 // TAG name type 和 /* TAG */ name type 两种格式
func ParseParameters(rawInput string) ([]Parameter, error) {
	// 1. 预处理输入字符串
	content := strings.TrimSpace(rawInput)
	if !strings.HasPrefix(content, "(") || !strings.HasSuffix(content, ")") {
		return nil, errors.New("无效的输入格式：必须由一对括号 '()' 包裹")
	}
	content = content[1 : len(content)-1] // 去掉 ( 和 )
	if strings.TrimSpace(content) == "" {
		return []Parameter{}, nil // 处理空参数列表 "()"
	}

	// 2. 按逗号分割成独立的参数定义
	paramDefs := strings.Split(content, ",")

	var result []Parameter

	// 3. 遍历并解析每一个参数定义
	for _, part := range paramDefs {
		part = strings.TrimSpace(part)
		if part == "" {
			continue // 跳过由连续逗号产生的空部分
		}

		var currentParam Parameter
		var definitionPart string // 用来存放 "name type" 这部分

		// 4. 检查注释类型 (if-else if-else 结构)
		if strings.HasPrefix(part, "/*") {
			// --- 处理块注释: /* TAG */ name type ---
			endCommentIndex := strings.Index(part, "*/")
			if endCommentIndex == -1 {
				return nil, fmt.Errorf("解析错误：在 '%s' 中找到未闭合的块注释 '/*'", part)
			}
			// 提取注释标签，并去除前后空格
			currentParam.Tag = strings.TrimSpace(part[2:endCommentIndex])
			// 提取参数定义部分
			definitionPart = strings.TrimSpace(part[endCommentIndex+2:])

		} else if strings.HasPrefix(part, "//") {
			// --- 处理行注释: // TAG name type ---
			commentContent := strings.TrimSpace(strings.TrimPrefix(part, "//"))
			fields := strings.Fields(commentContent) // 使用 Fields 处理多种空白符
			if len(fields) < 3 {
				return nil, fmt.Errorf("解析错误：行注释格式不正确，应为 '// TAG name type'，实际为 '%s'", part)
			}
			currentParam.Tag = fields[0]
			definitionPart = strings.Join(fields[1:], " ")

		} else {
			// --- 处理无注释: name type ---
			currentParam.Tag = "" // 没有标签
			definitionPart = part
		}

		// 5. 从 "name type" 中分离出 name 和 type (公共逻辑)
		if definitionPart == "" {
			// 这种情况可能发生在 "/* @TAG */," 之后，注释后没有参数
			return nil, fmt.Errorf("解析错误：在 '%s' 中注释标签后缺少参数定义", part)
		}

		spaceIndex := strings.Index(definitionPart, " ")
		if spaceIndex == -1 {
			return nil, fmt.Errorf("解析错误：在 '%s' 中无法分离参数名和类型，应为 'name type' 格式", definitionPart)
		}

		currentParam.Name = definitionPart[:spaceIndex]
		currentParam.Type = strings.TrimSpace(definitionPart[spaceIndex:])

		// 避免添加空的参数（虽然前面的检查应该已经覆盖了）
		if currentParam.Name != "" && currentParam.Type != "" {
			result = append(result, currentParam)
		}
	}

	return result, nil
}
