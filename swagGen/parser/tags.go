package parsers

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// ParseMode 定义了标签内容的解析模式
type ParseMode int

const (
	// ModeNamed 表示内容是 key=value 对, 用分号分隔
	// 例如: @SECURITY(ApiKeyAuth; exclude=A,B; include=C,D)
	// 第一个无名参数会被映射到名为 "Value" 的字段
	ModeNamed ParseMode = iota

	// ModePositional 表示内容是按顺序排列的值, 用分号分隔
	// 例如: @HEADER(X-Api-Key;true;这是一个描述)
	// 值会按顺序填充到结构体的字段中
	ModePositional
)

// Definition 是所有可解析标签结构体必须实现的接口
type Definition interface {
	Name() string
	Mode() ParseMode // 新增方法，返回该定义的解析模式
}

// definitionInfo 存储了注册的结构体类型及其解析模式
type definitionInfo struct {
	Type reflect.Type
	Mode ParseMode
}

type Parser struct {
	definitions map[string]definitionInfo
}

func NewParser() *Parser {
	return &Parser{
		definitions: make(map[string]definitionInfo),
	}
}

// Register 注册时会同时存储类型和解析模式
func (p *Parser) Register(defs ...Definition) error {
	for _, def := range defs {
		t := reflect.TypeOf(def)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return fmt.Errorf("registration failed: %v is not a struct", t)
		}
		p.definitions[def.Name()] = definitionInfo{
			Type: t,
			Mode: def.Mode(),
		}
	}
	return nil
}

// Parse 方法现在根据注册的模式来路由解析逻辑
func (p *Parser) Parse(line string) (any, error) {
	line = strings.TrimSpace(line)
	tagLine := strings.TrimSpace(strings.TrimPrefix(line, "//"))
	if !strings.HasPrefix(tagLine, "@") {
		return nil, fmt.Errorf("tag format error: must start with '@'")
	}

	var (
		tagName string
		content string
		text    string // 新增：存储标签后的自由文本
	)

	// 检查是否存在括号
	firstParen := strings.Index(tagLine, "(")
	if firstParen == -1 {
		// 无括号格式: @TAG-NAME 自由文本
		parts := strings.SplitN(tagLine[1:], " ", 2) // 用第一个空格分隔标签名和自由文本
		tagName = parts[0]
		if len(parts) > 1 {
			text = strings.TrimSpace(parts[1])
		}
	} else {
		// 有括号格式: @TAG-NAME(content) 自由文本
		if !strings.HasSuffix(tagLine, ")") {
			return nil, fmt.Errorf("tag format error: found opening parenthesis '(' but no matching ')' at the end")
		}

		// 提取标签名和括号内容
		tagName = tagLine[1:firstParen]
		content = tagLine[firstParen+1 : strings.LastIndex(tagLine, ")")]

		// 检查括号后是否有自由文本
		rest := strings.TrimSpace(tagLine[strings.LastIndex(tagLine, ")")+1:])
		if rest != "" {
			text = rest
		}
	}

	_ = text

	info, ok := p.definitions[tagName]
	if !ok {
		return nil, fmt.Errorf("unregistered tag: %s", tagName)
	}

	newStructPtr := reflect.New(info.Type)
	newStructElem := newStructPtr.Elem()

	var err error
	switch info.Mode {
	case ModeNamed:
		err = p.fillStructFromNamed(newStructElem, content)
	case ModePositional:
		err = p.fillStructFromPositional(newStructElem, content)
	default:
		err = fmt.Errorf("unknown parsing mode")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fill struct (tag: %s): %w", tagName, err)
	}
	if err := p.validateStruct(newStructElem); err != nil {
		return nil, fmt.Errorf("validation failed (tag: %s): %w", tagName, err)
	}

	return newStructPtr.Interface(), nil
}

// --- 填充逻辑 ---

// fillStructFromNamed 处理具名参数 (key=value)
func (p *Parser) fillStructFromNamed(structElem reflect.Value, content string) error {
	// 1. 解析内容为 map
	args := make(map[string]string)
	parts := strings.Split(content, ";")
	firstPart := strings.TrimSpace(parts[0])
	if firstPart != "" && !strings.Contains(firstPart, "=") {
		args["value"] = firstPart
		parts = parts[1:]
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		keyValue := strings.SplitN(part, "=", 2)
		if len(keyValue) != 2 {
			return fmt.Errorf("invalid named parameter format: '%s'", part)
		}
		key := strings.ToLower(strings.TrimSpace(keyValue[0]))
		value := strings.TrimSpace(keyValue[1])
		args[key] = value
	}

	// 2. 填充结构体
	structType := structElem.Type()
	for i := 0; i < structElem.NumField(); i++ {
		field := structElem.Field(i)
		fieldType := structType.Field(i)
		argKey := strings.ToLower(fieldType.Name)
		if fieldType.Name == "Value" {
			argKey = "value"
		}
		argValue, ok := args[argKey]
		if !ok {
			continue
		}
		if err := setFieldFromString(field, fieldType, argValue); err != nil {
			return fmt.Errorf("error setting field '%s': %w", fieldType.Name, err)
		}
	}
	return nil
}

// fillStructFromPositional 处理位置参数
func (p *Parser) fillStructFromPositional(structElem reflect.Value, content string) error {
	parts := strings.Split(content, ";")
	structType := structElem.Type()

	// 如果内容为空，但结构体有字段，则 parts 会是 [""]
	if len(parts) == 1 && parts[0] == "" {
		parts = []string{} // 视为空参数列表
	}

	if len(parts) > structElem.NumField() {
		return fmt.Errorf("provided %d parameters, but struct has only %d fields", len(parts), structElem.NumField())
	}

	for i, part := range parts {
		field := structElem.Field(i)
		fieldType := structType.Field(i)
		value := strings.TrimSpace(part)
		if err := setFieldFromString(field, fieldType, value); err != nil {
			return fmt.Errorf("error setting field %d '%s': %w", i+1, fieldType.Name, err)
		}
	}
	return nil
}

// setFieldFromString 是一个通用的字段设置函数，支持 string, bool, []string
func setFieldFromString(field reflect.Value, fieldType reflect.StructField, value string) error {
	if !field.CanSet() {
		return fmt.Errorf("field cannot be set")
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			// 将空字符串或 "false" 以外的无效值视为 false
			if value == "" {
				b = false
			} else {
				return fmt.Errorf("'%s' is not a valid boolean value", value)
			}
		}
		field.SetBool(b)
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			sgTag := parseSgTag(fieldType.Tag.Get("sg"))
			delimiter := " " // 默认分隔符是空格
			if d, ok := sgTag["delimiter"]; ok {
				delimiter = d
			}
			var values []string
			if value != "" {
				if delimiter == " " {
					values = strings.Fields(value)
				} else {
					values = strings.Split(value, delimiter)
				}
			}
			field.Set(reflect.ValueOf(values))
		}
	default:
		return fmt.Errorf("unsupported target field type: %s", field.Kind())
	}
	return nil
}

// parseContent, fillStruct, validateStruct, parseSgTag 函数均无变化
func (p *Parser) parseContent(content string) (map[string]string, error) {
	args := make(map[string]string)
	parts := strings.Split(content, ";")
	firstPart := strings.TrimSpace(parts[0])
	if !strings.Contains(firstPart, "=") {
		args["value"] = firstPart
		parts = parts[1:]
	}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		keyValue := strings.SplitN(part, "=", 2)
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("invalid parameter format: '%s'", part)
		}
		key := strings.TrimSpace(keyValue[0])
		value := strings.TrimSpace(keyValue[1])
		args[strings.ToLower(key)] = value
	}
	return args, nil
}

func (p *Parser) fillStruct(structElem reflect.Value, args map[string]string) error {
	structType := structElem.Type()
	for i := 0; i < structElem.NumField(); i++ {
		field := structElem.Field(i)
		fieldType := structType.Field(i)
		argKey := strings.ToLower(fieldType.Name)
		if fieldType.Name == "Value" {
			argKey = "value"
		}
		argValue, ok := args[argKey]
		if !ok {
			continue
		}
		if !field.CanSet() {
			continue
		}
		sgTag := parseSgTag(fieldType.Tag.Get("sg"))
		switch field.Kind() {
		case reflect.String:
			field.SetString(argValue)
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				delimiter := " "
				if d, ok := sgTag["delimiter"]; ok {
					delimiter = d
				}
				var values []string
				if argValue != "" {
					if delimiter == " " {
						values = strings.Fields(argValue)
					} else {
						values = strings.Split(argValue, delimiter)
					}
				}
				field.Set(reflect.ValueOf(values))
			}
		default:
			return fmt.Errorf("unsupported field type: %s (%s)", fieldType.Name, field.Kind())
		}
	}
	return nil
}

func (p *Parser) validateStruct(structElem reflect.Value) error {
	structType := structElem.Type()
	for i := 0; i < structElem.NumField(); i++ {
		field := structElem.Field(i)
		fieldType := structType.Field(i)
		sgTag := parseSgTag(fieldType.Tag.Get("sg"))
		if _, required := sgTag["required"]; required {
			isZero := false
			switch field.Kind() {
			case reflect.String:
				if field.String() == "" {
					isZero = true
				}
			case reflect.Slice:
				if field.Len() == 0 {
					isZero = true
				}
			}
			if isZero {
				return fmt.Errorf("field '%s' is required but value is empty", fieldType.Name)
			}
		}
	}
	return nil
}

func parseSgTag(tag string) map[string]string {
	result := make(map[string]string)
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		key := kv[0]
		value := ""
		if len(kv) > 1 {
			value = kv[1]
		}
		result[key] = value
	}
	return result
}
