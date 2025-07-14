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
	Tag  string // 注释标签 (如 @PARM, @FORM)
}

func (p Parameter) FullName() string {
	return fmt.Sprintf("%s %s", p.Name, p.Type)
}

// ParseParameters 解析函数参数定义字符串
// 输入示例: ( c *gin.Context, // @PARM id string, // @FORM data SendOTPReq, )
func ParseParameters(rawInput string) ([]Parameter, error) {
	// 1. 预处理输入字符串
	// 去除首尾的空白符
	content := strings.TrimSpace(rawInput)

	// 检查并去除首尾的括号
	if !strings.HasPrefix(content, "(") || !strings.HasSuffix(content, ")") {
		return nil, errors.New("无效的输入格式：缺少括号")
	}
	content = content[1 : len(content)-1] // 去掉 ( 和 )

	// 2. 按逗号分割成独立的参数定义
	paramDefs := strings.Split(content, ",")

	var result []Parameter

	// 3. 遍历并解析每一个参数定义
	for _, part := range paramDefs {
		// 去除每个部分前后的空白符
		part = strings.TrimSpace(part)
		if part == "" {
			continue // 如果是空字符串（比如由 "id string, ," 这种连续逗号产生），则跳过
		}

		var currentParam Parameter
		var definitionPart string // 用来存放 "name type" 这部分

		// 检查是否存在注释标签
		if strings.HasPrefix(part, "//") {
			// 清理注释标记，保留后面的内容
			commentContent := strings.TrimSpace(strings.TrimPrefix(part, "//"))

			// 使用 Fields 可以轻松处理多个空格或制表符
			fields := strings.Fields(commentContent)
			if len(fields) < 3 {
				// 格式如 "// @TAG name type" 至少需要3个部分
				continue // 或者返回错误，取决于你的严格程度
			}

			currentParam.Tag = fields[0]
			// 将 "name type" 部分重新组合起来
			definitionPart = strings.Join(fields[1:], " ")

		} else {
			// 没有注释标签，整个部分都是 "name type"
			definitionPart = part
		}

		// 4. 从 "name type" 中分离出 name 和 type
		// 寻找第一个空白符的位置
		spaceIndex := strings.Index(definitionPart, " ")
		if spaceIndex == -1 {
			// 没有找到空格，格式不正确（例如只有一个 "name"）
			continue // 或者返回错误
		}

		currentParam.Name = definitionPart[:spaceIndex]
		// 类型是第一个空格之后的所有内容
		currentParam.Type = strings.TrimSpace(definitionPart[spaceIndex:])

		result = append(result, currentParam)
	}

	return result, nil
}
