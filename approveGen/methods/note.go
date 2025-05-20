package methods

import (
	"github.com/donutnomad/gotoolkit/internal/utils"
	"regexp"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/samber/lo"
)

// NoteMethodInfo stores the Note method definition information
type NoteMethodInfo struct {
	Note string // The note string to be returned
}

// ParseNoteMethod parses the Note method annotation
// Example: args::note="This is a note"
func ParseNoteMethod(content string) *NoteMethodInfo {
	info := &NoteMethodInfo{}

	// Parse the note content
	if m := regexp.MustCompile(`args::note="(.*?)"`).FindStringSubmatch(content); len(m) > 1 {
		info.Note = strings.ReplaceAll(m[1], "\t", " ")
	}

	if info.Note == "" {
		return nil
	}
	return info
}

func (info *NoteMethodInfo) Generator() *NoteMethod {
	return &NoteMethod{
		Info:       info,
		MethodName: "Note",
	}
}

type NoteMethod struct {
	Info       *NoteMethodInfo
	Receiver   string
	StructName string
	MethodName string
}

func (m *NoteMethod) WithMethod(name string) *NoteMethod {
	m.MethodName = name
	return m
}

func (m *NoteMethod) Generate(receiver, structName string) jen.Code {
	m.Receiver = receiver
	m.StructName = structName
	return jen.Id(lo.Must1(m.generate())).Line()
}

func (m *NoteMethod) generate() (string, error) {
	return utils.ExecuteTemplate(m,
		`
func ({{.Receiver}} *{{.StructName}}) {{.MethodName}}() string {
    return "{{.Info.Note}}"
}
`)
}
