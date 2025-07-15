package parsers

import (
	"fmt"
	"testing"
)

func TestTag(t *testing.T) {
	parser := NewParser()
	err := parser.Register(Tag{}, Security{}, GET{}, Header{}, FormReq{}, Removed{})
	if err != nil {
		panic(err)
	}

	testCases := []string{
		// --- 合法用例 ---
		"// @TAG(Company,A,B,C)",
		"//    @SECURITY(ApiKeyAuth; exclude=A,B ;  include= C D )   ", // 末尾有空格，可以被trim
		// --- 新增的非法用例 ---
		"// @TAG(no close paren",
		"// @TAG(has trailing chars) abc",
		"// @TAG[mismatched parens)",
		// --- 原有的非法用例 ---
		"// @SECURITY(MyValue)",
		"// @TAGS()",
		"// @UNKNOWN(some_value)",
		"// Not a tag",
		"// @GET(/api/v1/abc/{userId})",

		// --- 具名模式用例 ---
		"// @SECURITY(ApiKeyAuth; include=A B C)",
		"// @SECURITY(exclude=X,Y; include=Z)", // 无名主值 'Value' 为空，会报错

		// --- 错误用例 ---
		"// @HEADER(a;b;c;d)",  // 参数过多
		"// @HEADER(a;true;c)", // 参数过多

		"// @FORM-REQ", // 成功: 类型=*parsers.FormReq, 值=&parsers.FormReq{}
		"// @SECURITY", // 验证失败 (标签: SECURITY): 字段 'Value' 是必须的，但值为空
		"@Removed",
	}

	for _, tc := range testCases {
		fmt.Printf("\n解析: %s\n", tc)
		result, err := parser.Parse(tc)
		if err != nil {
			fmt.Printf("  -> 错误: %v\n", err)
		} else {
			fmt.Printf("  -> 成功: 类型=%T, 值=%#v\n", result, result)
		}
	}
}
