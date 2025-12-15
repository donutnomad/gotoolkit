package structparse

import "testing"

func TestEncodeModulePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "all lowercase",
			input:    "gorm.io/gorm",
			expected: "gorm.io/gorm",
		},
		{
			name:     "single capital in username",
			input:    "github.com/Xuanwo/gg",
			expected: "github.com/!xuanwo/gg",
		},
		{
			name:     "single capital S in Samber",
			input:    "github.com/Samber/lo",
			expected: "github.com/!samber/lo",
		},
		{
			name:     "multiple capitals - BurntSushi",
			input:    "github.com/BurntSushi/toml",
			expected: "github.com/!burnt!sushi/toml",
		},
		{
			name:     "multiple capitals - DataDog",
			input:    "github.com/DataDog/datadog-go",
			expected: "github.com/!data!dog/datadog-go",
		},
		{
			name:     "all capitals - ABC",
			input:    "github.com/ABC/xyz",
			expected: "github.com/!a!b!c/xyz",
		},
		{
			name:     "mixed case in path",
			input:    "github.com/user/MyRepo",
			expected: "github.com/user/!my!repo",
		},
		{
			name:     "no change needed",
			input:    "github.com/donutnomad/gotoolkit",
			expected: "github.com/donutnomad/gotoolkit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeModulePath(tt.input)
			if got != tt.expected {
				t.Errorf("encodeModulePath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
