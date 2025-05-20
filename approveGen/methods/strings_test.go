package methods

import (
	"reflect"
	"testing"
)

func TestParseStringMethod(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *StringMethodInfo
		wantErr bool
	}{
		{
			name:    "with include fields",
			content: `args::string="$key=$value"; include="Name,Age"; exclude="field1,field2"`,
			want: &StringMethodInfo{
				ArgsTemplate:  "$key=$value",
				IncludeFields: []string{"Name", "Age"},
				ExcludeFields: []string{"field1", "field2"},
				Separator:     ", ",
			},
		},
		{
			name:    "only include fields",
			content: `args::string="$key=$value"; include="Name,Age"`,
			want: &StringMethodInfo{
				ArgsTemplate:  "$key=$value",
				IncludeFields: []string{"Name", "Age"},
				ExcludeFields: nil,
				Separator:     ", ",
			},
		},
		{
			name:    "only exclude fields",
			content: `args::string="$key=$value"; exclude="CreatedAt,UpdatedAt"`,
			want: &StringMethodInfo{
				ArgsTemplate:  "$key=$value",
				IncludeFields: nil,
				ExcludeFields: []string{"CreatedAt", "UpdatedAt"},
				Separator:     ", ",
			},
		},
		{
			name:    "complete format",
			content: `args::string="$key: $value"; sep=" && "; include="ID,Name,Age"`,
			want: &StringMethodInfo{
				ArgsTemplate:  "$key: $value",
				IncludeFields: []string{"ID", "Name", "Age"},
				ExcludeFields: nil,
				Separator:     " && ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseStringMethod(tt.content)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseStringMethod() = %v, want %v", got, tt.want)
			}
		})
	}
}
