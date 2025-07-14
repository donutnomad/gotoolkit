package parsers

import (
	"fmt"
	"testing"
)

func TestPackages(t *testing.T) {
	// 示例 1
	input1 := "service.BaseResponse[service_resp.DD]"
	extracted1 := ExtractPackages(input1)
	fmt.Printf("输入: %s\n", input1)
	fmt.Printf("提取结果: %v\n\n", extracted1) // 期望输出: [service service_resp]

	// 示例 2
	input2 := "service.BaseResponse[dd2.A[dd3.B]]"
	extracted2 := ExtractPackages(input2)
	fmt.Printf("输入: %s\n", input2)
	fmt.Printf("提取结果: %v\n\n", extracted2) // 期望输出: [service dd2 dd3]

	// 更多测试用例
	input3 := "a.B[c.D[e.F[g.H]]]"
	extracted3 := ExtractPackages(input3)
	fmt.Printf("输入: %s\n", input3)
	fmt.Printf("提取结果: %v\n\n", extracted3) // 期望输出: [a c e g]

	// 测试有重复包名的情况
	input4 := "service.A[service.B]"
	extracted4 := ExtractPackages(input4)
	fmt.Printf("输入: %s\n", input4)
	fmt.Printf("提取结果: %v\n\n", extracted4) // 期望输出: [service]
}
