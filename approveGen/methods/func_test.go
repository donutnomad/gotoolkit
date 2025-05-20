package methods

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFuncMethod(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected *FuncMethodInfo
		isNil    bool
	}{
		{
			name:    "基本函数定义",
			content: `func:name="approveFor"; module="ABC"; event="EVENT"`,
			expected: &FuncMethodInfo{
				Name: "approveFor",
				Attributes: map[string]string{
					"module": "ABC",
					"event":  "EVENT",
				},
			},
		},
		{
			name:    "只有函数名",
			content: `func:name="validateData"`,
			expected: &FuncMethodInfo{
				Name:       "validateData",
				Attributes: map[string]string{},
			},
		},
		{
			name:    "多个属性",
			content: `func:name="processEvent"; module="USER"; event="CREATED"; priority="HIGH"; async="true"`,
			expected: &FuncMethodInfo{
				Name: "processEvent",
				Attributes: map[string]string{
					"module":   "USER",
					"event":    "CREATED",
					"priority": "HIGH",
					"async":    "true",
				},
			},
		},
		{
			name:    "不同格式的第一部分",
			content: `func:name="handleRequest"; path="/api/v1"; method="POST"`,
			expected: &FuncMethodInfo{
				Name: "handleRequest",
				Attributes: map[string]string{
					"path":   "/api/v1",
					"method": "POST",
				},
			},
		},
		{
			name:    "无效格式 - 不以func:开头",
			content: `name="approveFor"; module="ABC"`,
			isNil:   true,
		},
		{
			name:    "无效格式 - 没有name属性",
			content: `func:module="ABC"; event="EVENT"`,
			isNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFuncMethod(tt.content)

			if tt.isNil {
				assert.Nil(t, result)
				return
			}

			assert.NotNil(t, result)
			assert.Equal(t, tt.expected.Name, result.Name)

			// 验证属性
			assert.Equal(t, len(tt.expected.Attributes), len(result.Attributes))
			for key, expectedValue := range tt.expected.Attributes {
				actualValue, exists := result.Attributes[key]
				assert.True(t, exists, "属性 %s 应该存在", key)
				assert.Equal(t, expectedValue, actualValue, "属性 %s 的值应该是 %s，但得到了 %s", key, expectedValue, actualValue)
			}
		})
	}
}

func TestFuncMethodGenerator(t *testing.T) {
	info := &FuncMethodInfo{
		Name: "approveFor",
		Attributes: map[string]string{
			"module": "ABC",
			"event":  "EVENT",
		},
	}

	generator := info.Generator()
	assert.NotNil(t, generator)

	// 测试生成的代码
	//code := generator.Generate("", "r", "TestStruct", "ApproveFor_"+"DDD", "", "&_AAAeMethodBBBB{}", "fmt.Stringer")
	//assert.NotNil(t, code)
	//fmt.Println(jen.Add(code).GoString())
}
