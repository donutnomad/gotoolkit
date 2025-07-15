package parsers

import (
	"strings"
	"unicode"
)

// ToCamel 将字符串转换为小驼峰（lowerCamelCase）格式。
// 这个函数是健壮的，可以处理 snake_case, kebab-case, 混合大小写等多种格式。
//
// 示例:
//
//	request_id        => requestID
//	HTTPRequestState  => httpRequestState
//	_my_variable-name => myVariableName
func ToCamel(s string) string {
	if s == "" {
		return ""
	}

	// 1. 按分隔符和大小写变化切分单词
	words := splitIntoWords(s)
	if len(words) == 0 {
		return ""
	}

	var builder strings.Builder

	// 2. 处理第一个单词：全小写
	firstWord := strings.ToLower(words[0])
	builder.WriteString(firstWord)

	// 3. 处理后续单词：首字母大写，其余小写
	for _, word := range words[1:] {
		// Title-case the word (e.g., "world" -> "World", "ID" -> "Id")
		if len(word) > 0 {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			for i := 1; i < len(runes); i++ {
				runes[i] = unicode.ToLower(runes[i])
			}
			builder.WriteString(string(runes))
		}
	}

	return builder.String()
}

// splitIntoWords 是一个辅助函数，用于将各种格式的字符串切分为单词列表。
// 它能处理下划线、中划线、空格作为分隔符，并且能识别驼峰命名中的边界。
// 例如 "MyVariable_name-for-HTTP" -> ["My", "Variable", "name", "for", "HTTP"]
func splitIntoWords(s string) []string {
	if s == "" {
		return nil
	}
	var words []string
	var currentWord strings.Builder

	runes := []rune(s)

	for i, r := range runes {
		// 如果是分隔符，则结束当前单词
		if r == '_' || r == '-' || r == ' ' {
			if currentWord.Len() > 0 {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
			continue
		}

		// 如果当前字符是大写
		if unicode.IsUpper(r) {
			// 如果当前单词不为空，并且前一个字符不是大写或分隔符，则说明新单词开始了
			// 例: "myVariable", 'V' 是新单词的开始
			if currentWord.Len() > 0 && !unicode.IsUpper(runes[i-1]) {
				words = append(words, currentWord.String())
				currentWord.Reset()
			}
		}

		currentWord.WriteRune(r)
	}

	// 添加最后一个单词
	if currentWord.Len() > 0 {
		words = append(words, currentWord.String())
	}

	return words
}

// CamelString 是一个自定义类型，用于表示驼峰格式的字符串。
type CamelString string

// NewCamelString 通过 ToCamel 函数创建一个新的 CamelString。
func NewCamelString(input string) CamelString {
	return CamelString(ToCamel(input))
}

// Equal 检查一个 CamelString 是否等价于另一个任意格式的变量名字符串。
// 等价性是通过将两个字符串都转换为规范的小驼峰格式后进行比较来确定的。
//
// 示例:
//
//		cs := NewCamelString("request_id") // cs 的值是 "requestID"
//		cs.Equal("requestID")             // true
//		cs.Equal("requestId")             // true (因为 ToCamel("requestId") => "requestid"，而 ToCamel("request_id") => "requestid" )
//	 cs.Equal("RequestID")             // true
//	 cs.Equal("request-id")            // true
func (cs CamelString) Equal(other string) bool {
	// 将自身和对方都转换为规范的小驼峰格式进行比较
	// 这是判断等价性的最可靠方法
	canonicalSelf := ToCamel(string(cs))
	canonicalOther := ToCamel(other)
	return canonicalSelf == canonicalOther
}
