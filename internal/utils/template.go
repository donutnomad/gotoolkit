package utils

import (
	"bytes"
	"github.com/samber/lo"
	"text/template"
)

func MustExecuteTemplate(data any, tpl string) string {
	return lo.Must1(ExecuteTemplate(data, tpl))
}

func ExecuteTemplate(data any, tpl string) (string, error) {
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).Parse(tpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
