package methods

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFieldMethod(t *testing.T) {
	tests := []struct {
		name     string
		contents []string
		expected []*FieldInfo
	}{
		{
			name: "single annotation",
			contents: []string{
				`args::field="rawID"; func="格式化的方法"; alias="别名"`,
			},
			expected: []*FieldInfo{
				{
					Field:    "rawID",
					Function: "格式化的方法",
					Alias:    "别名",
				},
			},
		},
		{
			name: "multiple annotations",
			contents: []string{
				`args::field="rawID"; func="格式化的方法"; alias="别名"`,
				`args::field="name"; func="名字格式化"`,
				`args::field="age"; alias="年龄"`,
			},
			expected: []*FieldInfo{
				{
					Field:    "rawID",
					Function: "格式化的方法",
					Alias:    "别名",
				},
				{
					Field:    "name",
					Function: "名字格式化",
				},
				{
					Field: "age",
					Alias: "年龄",
				},
			},
		},
		{
			name: "mixed valid and invalid",
			contents: []string{
				`args::field="rawID"`,
				`func="格式化的方法"; alias="别名"`,
				`invalid="test"`,
			},
			expected: []*FieldInfo{
				{
					Field: "rawID",
				},
			},
		},
		{
			name:     "empty input",
			contents: []string{},
			expected: nil,
		},
		{
			name: "all invalid",
			contents: []string{
				`func="格式化的方法"`,
				`alias="别名"`,
				`invalid="test"`,
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFieldMethod(tt.contents)
			if tt.expected == nil {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, len(tt.expected), len(result))
				for i, expected := range tt.expected {
					assert.Equal(t, expected.Field, result[i].Field)
					assert.Equal(t, expected.Function, result[i].Function)
					assert.Equal(t, expected.Alias, result[i].Alias)
				}
			}
		})
	}
}
