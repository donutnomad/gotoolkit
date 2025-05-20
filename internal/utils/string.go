package utils

import "strings"

func RemoveQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		// 去掉首尾的双引号
		return s[1 : len(s)-1]
	}
	return s
}

func UpperCamelCase(s string) string {
	s = strings.Replace(s, "_", " ", -1)
	s = strings.Title(s)
	return strings.Replace(s, " ", "", -1)
}

type EString string

func (e EString) String() string {
	return string(e)
}

func (e EString) UpperCamelCase() EString {
	return EString(UpperCamelCase(e.String()))
}

func (e EString) LowerCamelCase() EString {
	item := UpperCamelCase(e.String())
	item = strings.ToLower(item[:1]) + item[1:]
	return EString(item)
}

func (e EString) RemoveQuotes() EString {
	return EString(RemoveQuotes(e.String()))
}
