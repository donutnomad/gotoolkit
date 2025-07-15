package parsers

import (
	"fmt"
	"testing"
)

func TestWord(t *testing.T) {
	fmt.Println("--- ToCamel 函数测试 ---")
	testCases := []string{
		"request_id",
		"reqeust_book",
		"HTTPRequest",
		"__API_Key-for-user__",
		"version-1-2",
		"alreadyCamel",
		" leading-space",
		"trailing-space ",
		"A",
		"b",
		"",
		"__",
	}

	for _, tc := range testCases {
		fmt.Printf("输入: %-25s  输出: %s\n", `"`+tc+`"`, ToCamel(tc))
	}

	fmt.Println("\n--- Equal 方法测试 ---")

	// 创建一个 CamelString 实例
	varName := NewCamelString("http_request_id")
	fmt.Printf("创建的 CamelString: \"%s\"\n", varName)

	equivalents := []string{
		"httpRequestID",
		"http-request-id",
		"HTTPRequestID",
		" http_request_id ",
		"httpRequestId", // 这个也会被规范化
	}

	nonEquivalents := []string{
		"httpRequest",
		"http_request",
		"anotherVar",
	}

	fmt.Println("\n测试等价的名称:")
	for _, eq := range equivalents {
		fmt.Printf("`%s`.Equal(`%s`) => %v\n", varName, eq, varName.Equal(eq))
	}

	fmt.Println("\n测试不等价的名称:")
	for _, neq := range nonEquivalents {
		fmt.Printf("`%s`.Equal(`%s`) => %v\n", varName, neq, varName.Equal(neq))
	}

	// 另一个例子: request_id => requestID or requestId
	// 我们的实现会将 request_id 规范化为 requestid
	// 而 requestID 和 requestId 都会被规范化为 requestid
	fmt.Println("\n--- 特殊等价性测试 (request_id vs requestID/requestId) ---")
	specialVar := NewCamelString("request_id")
	fmt.Printf("NewCamelString(\"request_id\") => \"%s\"\n", specialVar)
	fmt.Printf("`%s`.Equal(\"requestID\") => %v\n", specialVar, specialVar.Equal("requestID"))
	fmt.Printf("`%s`.Equal(\"requestId\") => %v\n", specialVar, specialVar.Equal("requestId"))

}
