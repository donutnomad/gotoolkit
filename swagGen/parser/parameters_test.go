package parsers

import (
	"fmt"
	"testing"
)

func TestParseParameters(t *testing.T) {
	// 1. 您的原始示例
	input1 := `(		c *gin.Context,		// @PARM(user_id)		id string,		// @FORM		data SendOTPReq,	)`
	fmt.Println("--- 测试用例 1: 原始输入 ---")
	params1, err := ParseParameters(input1)
	if err != nil {
		fmt.Printf("解析错误: %v\n", err)
	} else {
		for _, p := range params1 {
			fmt.Printf("参数名: %-10s | 类型: %-15s | 标签: %-10s\n", p.Name, p.Type, p.Tag)
		}
	}

	// 2. 包含不同空白符和换行符的复杂示例
	input2 := `(
		ctx *gin.Context, // @CTX user User,
		// @HEADER token string,
		// @BODY   reqBody  map[string]any,
	)`
	fmt.Println("\n--- 测试用例 2: 复杂空白符和换行 ---")
	params2, err := ParseParameters(input2)
	if err != nil {
		fmt.Printf("解析错误: %v\n", err)
	} else {
		for _, p := range params2 {
			fmt.Printf("参数名: %-10s | 类型: %-15s | 标签: %-10s\n", p.Name, p.Type, p.Tag)
		}
	}

	// 3. 没有参数的示例
	input3 := `()`
	fmt.Println("\n--- 测试用例 3: 没有参数 ---")
	params3, err := ParseParameters(input3)
	if err != nil {
		fmt.Printf("解析错误: %v\n", err)
	} else {
		fmt.Printf("解析到 %d 个参数\n", len(params3))
	}

	// 4. 格式错误的示例
	input4 := ` c *gin.Context, id string`
	fmt.Println("\n--- 测试用例 4: 格式错误 (缺少括号) ---")
	_, err = ParseParameters(input4)
	if err != nil {
		fmt.Printf("预期内的解析错误: %v\n", err)
	}
}
