package methods

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNoteMethod(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected *NoteMethodInfo
	}{
		{
			name:    "valid note",
			content: `args::note="This is a test note"`,
			expected: &NoteMethodInfo{
				Note: "This is a test note",
			},
		},
		{
			name:     "empty note",
			content:  `args::note=""`,
			expected: nil,
		},
		{
			name:     "invalid format",
			content:  `args::invalid="test"`,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseNoteMethod(tt.content)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected.Note, result.Note)
			}
		})
	}
}

func TestNoteMethod_Generate(t *testing.T) {
	info := &NoteMethodInfo{
		Note: "Test Note",
	}
	method := info.Generator()

	stmt := method.Generate("p", "TestStruct")
	assert.NotNil(t, stmt)

	// Generate the method code
	code, err := method.generate()
	assert.NoError(t, err)
	assert.Contains(t, code, "Test Note")
	assert.Contains(t, code, "func (p *TestStruct) Note() string")
}
