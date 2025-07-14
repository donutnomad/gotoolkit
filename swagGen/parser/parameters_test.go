package parsers

import (
	"fmt"
	"testing"
)

func Test2(t *testing.T) {
	input := `(
		ctx *gin.Context,
		// @PARAM(token_id)
		tokenId uint,
		// @QUERY
		req GetTokenHistoryReq,
	)`
	params1, err := ParseParameters(input)
	if err != nil {
		fmt.Printf("解析错误: %v\n", err)
	} else {
		for _, p := range params1 {
			fmt.Printf("参数名: %-15s | 类型: %-15s | 标签: %-10s\n", p.Name, p.Type, p.Tag)
		}
	}
}

func TestParseParameters(t *testing.T) {
	fmt.Println("--- 测试用例 1: 混合注释类型 ---")
	input1 := `(
		/* @HEADER */   Authorization string,
		c *gin.Context,
		// @PATH id int,
		/*   @BODY   */	data  MyRequest,
	)`
	params1, err := ParseParameters(input1)
	if err != nil {
		fmt.Printf("解析错误: %v\n", err)
	} else {
		for _, p := range params1 {
			fmt.Printf("参数名: %-15s | 类型: %-15s | 标签: %-10s\n", p.Name, p.Type, p.Tag)
		}
	}

	fmt.Println("\n--- 测试用例 2: 原始行注释输入 ---")
	input2 := `(c *gin.Context, // @PARAM id string, // @FORM data SendOTPReq,)`
	params2, err := ParseParameters(input2)
	if err != nil {
		fmt.Printf("解析错误: %v\n", err)
	} else {
		for _, p := range params2 {
			fmt.Printf("参数名: %-15s | 类型: %-15s | 标签: %-10s\n", p.Name, p.Type, p.Tag)
		}
	}

	fmt.Println("\n--- 测试用例 3: 包含多行内容的块注释 ---")
	input3 := `(
		/* 
		  @BODY
		  @Description("用户提交的数据")
		*/
		payload   map[string]any
	)`
	params3, err := ParseParameters(input3)
	if err != nil {
		fmt.Printf("解析错误: %v\n", err)
	} else {
		// 注意：Tag 会包含换行符和空格，这符合预期
		// 如果需要进一步解析Tag内部，需要额外的逻辑
		for _, p := range params3 {
			fmt.Printf("参数名: %-15s | 类型: %-15s\n", p.Name, p.Type)
			fmt.Printf("原始标签内容:\n---\n%s\n---\n", p.Tag)
		}
	}

	fmt.Println("\n--- 测试用例 4: 格式错误 (未闭合块注释) ---")
	input4 := `(/* @HEADER */ token string, /* @OOPS `
	_, err4 := ParseParameters(input4)
	if err4 != nil {
		fmt.Printf("预期内的解析错误: %v\n", err4)
	}

	fmt.Println("\n--- 测试用例 5: 格式错误 (注释后无参数) ---")
	input5 := `(id int, /* @TAG */ )`
	_, err5 := ParseParameters(input5)
	if err5 != nil {
		fmt.Printf("预期内的解析错误: %v\n", err5)
	}
}
