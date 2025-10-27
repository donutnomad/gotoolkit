package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/donutnomad/gotoolkit/internal/utils"
	"github.com/samber/lo"
)

var dir = flag.String("dir", ".", "目录路径")
var readable = flag.Bool("readable", false, "生成接口的可读实现类")
var interfaceNames = flag.String("name", "", "接口名称,多个使用逗号分隔")

// MethodInfo 方法信息
type MethodInfo struct {
	Name      string   // 方法名
	Params    []string // 参数列表
	Returns   []string // 返回值列表
	IsQuery   bool     // 是否为查询方法
	Signature string   // 方法签名
}

// InterfaceInfo 接口信息
type InterfaceInfo struct {
	Name     string       // 接口名
	Methods  []MethodInfo // 方法列表
	FilePath string       // 文件路径
	Package  string       // 包名
}

func main() {
	flag.Parse()

	// 必须指定 -name 参数
	if *interfaceNames == "" {
		log.Fatal("[itfgen] 请指定 -name 参数来指定接口名称")
	}

	// 必须指定 -readable 参数
	if !*readable {
		log.Fatal("[itfgen] 请指定 -readable 参数来生成接口的可读实现类")
	}

	// 解析接口名称列表
	args := strings.Split(*interfaceNames, ",")
	for i := range args {
		args[i] = strings.TrimSpace(args[i])
	}

	// 生成接口的可读实现类
	generateReadableImpls(args)
}

// generateReadableImpls 生成接口的可读实现类
func generateReadableImpls(args []string) {
	// 解析接口名称列表
	var structList []string
	for _, arg := range args {
		// 支持逗号分隔的多个接口名称
		names := strings.Split(arg, ",")
		for _, name := range names {
			if trimmed := strings.TrimSpace(name); trimmed != "" {
				structList = append(structList, trimmed)
			}
		}
	}

	// 按照文件分组收集接口信息
	fileInterfacesMap := make(map[string][]*InterfaceInfo)
	var fileOrderList []string // 保持文件顺序

	// 处理每个接口
	for _, structName := range structList {
		if structName == "" {
			continue
		}

		// 查找包含指定接口的文件
		files, err := findGoFiles(*dir)
		if err != nil {
			log.Fatalf("[itfgen] 查找Go文件失败: %v", err)
		}

		var targetFile string
		for _, file := range files {
			if containsInterface(file, structName) {
				targetFile = file
				break
			}
		}

		if targetFile == "" {
			log.Fatalf("[itfgen] 在目录 %s 中未找到包含接口 %s 的文件", *dir, structName)
		}

		fmt.Printf("[itfgen] 找到接口 %s 在文件: %s\n", structName, targetFile)

		// 解析接口
		interfaceInfo, err := parseInterface(targetFile, structName)
		if err != nil {
			log.Fatalf("[itfgen] 解析接口 %s 失败: %v", structName, err)
		}

		// 按文件分组
		if _, exists := fileInterfacesMap[targetFile]; !exists {
			fileOrderList = append(fileOrderList, targetFile)
		}
		fileInterfacesMap[targetFile] = append(fileInterfacesMap[targetFile], interfaceInfo)
	}

	// 按文件分组生成
	for _, targetFile := range fileOrderList {
		interfaces := fileInterfacesMap[targetFile]

		// 确定输出文件路径
		outputFile := strings.TrimSuffix(targetFile, ".go") + "_readable.go"

		err := generateReadableInterfaces(outputFile, interfaces)
		if err != nil {
			log.Fatalf("[itfgen] 生成可读实现类失败: %v", err)
		}

		fmt.Printf("[itfgen] 成功生成可读实现类: %s\n", outputFile)
	}
}

// findGoFiles 查找目录中的所有Go文件
func findGoFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_query.go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// containsInterface 检查文件是否包含指定的接口
func containsInterface(filename, interfaceName string) bool {
	content, err := os.ReadFile(filename)
	if err != nil {
		return false
	}

	// 简单的字符串匹配,检查是否包含 "type InterfaceName interface"
	return strings.Contains(string(content), fmt.Sprintf("type %s interface", interfaceName))
}

// parseInterface 解析接口定义
func parseInterface(filePath, interfaceName string) (*InterfaceInfo, error) {
	fset := token.NewFileSet()

	// 解析文件
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("解析文件失败: %w", err)
	}

	// 查找接口定义
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != interfaceName {
				continue
			}

			ifaceType, ok := typeSpec.Type.(*ast.InterfaceType)
			if !ok {
				continue
			}

			// 解析接口方法
			methods := parseInterfaceMethods(ifaceType)

			return &InterfaceInfo{
				Name:     interfaceName,
				Methods:  methods,
				FilePath: filePath,
				Package:  node.Name.Name,
			}, nil
		}
	}

	return nil, fmt.Errorf("未找到接口 %s", interfaceName)
}

// parseInterfaceMethods 解析接口方法
func parseInterfaceMethods(ifaceType *ast.InterfaceType) []MethodInfo {
	var methods []MethodInfo

	if ifaceType.Methods == nil {
		return methods
	}

	for _, method := range ifaceType.Methods.List {
		if len(method.Names) > 0 {
			methodInfo := parseMethod(method)
			methods = append(methods, methodInfo)
		}
	}

	return methods
}

// parseMethod 解析单个方法
func parseMethod(field *ast.Field) MethodInfo {
	methodName := field.Names[0].Name
	signature := buildMethodSignature(field)

	var params []string
	var returns []string

	// 解析参数
	if funcType, ok := field.Type.(*ast.FuncType); ok {
		params = parseFieldList(funcType.Params)
		returns = parseFieldList(funcType.Results)
	}

	// 判断是否为查询方法
	isQuery := isQueryMethod(methodName)

	return MethodInfo{
		Name:      methodName,
		Params:    params,
		Returns:   returns,
		IsQuery:   isQuery,
		Signature: signature,
	}
}

// parseFieldList 解析字段列表
func parseFieldList(fields *ast.FieldList) []string {
	var result []string

	if fields == nil {
		return result
	}

	for _, field := range fields.List {
		typeStr := getTypeString(field.Type)

		// 处理匿名参数
		if len(field.Names) == 0 {
			result = append(result, typeStr)
			continue
		}

		// 处理命名参数
		for _, name := range field.Names {
			result = append(result, name.Name+" "+typeStr)
		}
	}

	return result
}

// getTypeString 获取类型字符串
func getTypeString(expr ast.Expr) string {
	if expr == nil {
		return "interface{}"
	}

	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + getTypeString(t.X)
	case *ast.ArrayType:
		return "[]" + getTypeString(t.Elt)
	case *ast.SelectorExpr:
		return getTypeString(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map[" + getTypeString(t.Key) + "]" + getTypeString(t.Value)
	case *ast.ChanType:
		if t.Dir == ast.SEND {
			return "chan<- " + getTypeString(t.Value)
		} else if t.Dir == ast.RECV {
			return "<-chan " + getTypeString(t.Value)
		}
		return "chan " + getTypeString(t.Value)
	case *ast.Ellipsis:
		return "..." + getTypeString(t.Elt)
	case *ast.FuncType:
		// 构建完整的函数类型签名
		return buildFuncType(t)
	case *ast.InterfaceType:
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return "interface{}"
		}
		// 构建具体的接口类型
		return buildInterfaceType(t)
	case *ast.StructType:
		if t.Fields == nil || len(t.Fields.List) == 0 {
			return "struct{}"
		}
		// 构建具体的结构体类型
		return buildStructType(t)
	case *ast.ParenExpr:
		return "(" + getTypeString(t.X) + ")"
	case *ast.IndexExpr:
		// 处理泛型类型，如 T[K]
		return getTypeString(t.X) + "[" + getTypeString(t.Index) + "]"
	case *ast.IndexListExpr:
		// 处理多泛型参数，如 T[K, V]
		var indices []string
		for _, idx := range t.Indices {
			indices = append(indices, getTypeString(idx))
		}
		return getTypeString(t.X) + "[" + strings.Join(indices, ", ") + "]"
	default:
		// 对于未知类型，尝试使用 DebugString 或默认处理
		if t != nil {
			return fmt.Sprintf("/* unknown type: %T */", t)
		}
		return "interface{}"
	}
}

// buildFuncType 构建函数类型签名
func buildFuncType(ft *ast.FuncType) string {
	var builder strings.Builder
	builder.WriteString("func")

	// 参数列表
	builder.WriteString("(")
	if ft.Params != nil && len(ft.Params.List) > 0 {
		var params []string
		for _, param := range ft.Params.List {
			typeStr := getTypeString(param.Type)
			if len(param.Names) > 0 {
				for _, name := range param.Names {
					params = append(params, name.Name+" "+typeStr)
				}
			} else {
				params = append(params, typeStr)
			}
		}
		builder.WriteString(strings.Join(params, ", "))
	}
	builder.WriteString(")")

	// 返回值列表
	if ft.Results != nil && len(ft.Results.List) > 0 {
		builder.WriteString(" ")
		if len(ft.Results.List) == 1 && len(ft.Results.List[0].Names) == 0 {
			builder.WriteString(getTypeString(ft.Results.List[0].Type))
		} else {
			builder.WriteString("(")
			var returns []string
			for _, ret := range ft.Results.List {
				typeStr := getTypeString(ret.Type)
				if len(ret.Names) > 0 {
					for _, name := range ret.Names {
						returns = append(returns, name.Name+" "+typeStr)
					}
				} else {
					returns = append(returns, typeStr)
				}
			}
			builder.WriteString(strings.Join(returns, ", "))
			builder.WriteString(")")
		}
	}

	return builder.String()
}

// buildInterfaceType 构建接口类型
func buildInterfaceType(it *ast.InterfaceType) string {
	var builder strings.Builder
	builder.WriteString("interface{")

	if it.Methods != nil && len(it.Methods.List) > 0 {
		for i, method := range it.Methods.List {
			if i > 0 {
				builder.WriteString("; ")
			}

			if len(method.Names) > 0 {
				// 方法
				funcType, ok := method.Type.(*ast.FuncType)
				if ok {
					builder.WriteString(method.Names[0].Name)
					builder.WriteString(buildFuncType(funcType)[4:]) // 去掉 "func" 前缀
				}
			} else {
				// 嵌入的接口类型
				builder.WriteString(getTypeString(method.Type))
			}
		}
	}

	builder.WriteString("}")
	return builder.String()
}

// buildStructType 构建结构体类型
func buildStructType(st *ast.StructType) string {
	var builder strings.Builder
	builder.WriteString("struct{")

	if st.Fields != nil && len(st.Fields.List) > 0 {
		for i, field := range st.Fields.List {
			if i > 0 {
				builder.WriteString("; ")
			}

			typeStr := getTypeString(field.Type)
			if len(field.Names) > 0 {
				for j, name := range field.Names {
					if j > 0 {
						builder.WriteString("; ")
					}
					builder.WriteString(name.Name)
					builder.WriteString(" ")
					builder.WriteString(typeStr)
				}
			} else {
				builder.WriteString(typeStr)
			}
		}
	}

	builder.WriteString("}")
	return builder.String()
}

// buildMethodSignature 构建方法签名
func buildMethodSignature(field *ast.Field) string {
	var parts []string
	parts = append(parts, field.Names[0].Name+"(")

	// 添加参数
	if funcType, ok := field.Type.(*ast.FuncType); ok && funcType.Params != nil {
		var paramParts []string
		for _, param := range funcType.Params.List {
			typeStr := getTypeString(param.Type)
			if len(param.Names) > 0 {
				for _, name := range param.Names {
					paramParts = append(paramParts, name.Name+" "+typeStr)
				}
			} else {
				paramParts = append(paramParts, typeStr)
			}
		}
		parts = append(parts, strings.Join(paramParts, ", "))
	}
	parts = append(parts, ")")

	// 添加返回值
	if funcType, ok := field.Type.(*ast.FuncType); ok && funcType.Results != nil {
		var returnParts []string
		for _, ret := range funcType.Results.List {
			typeStr := getTypeString(ret.Type)
			if len(ret.Names) > 0 {
				for _, name := range ret.Names {
					returnParts = append(returnParts, name.Name+" "+typeStr)
				}
			} else {
				returnParts = append(returnParts, typeStr)
			}
		}

		if len(returnParts) > 0 {
			if len(returnParts) == 1 && !strings.Contains(returnParts[0], " ") {
				parts = append(parts, " "+returnParts[0])
			} else {
				parts = append(parts, " ("+strings.Join(returnParts, ", ")+")")
			}
		}
	}

	return strings.Join(parts, "")
}

// isQueryMethod 判断是否为查询方法
func isQueryMethod(methodName string) bool {
	queryPrefixes := []string{"List", "Get", "Find", "Search", "Query", "Count", "Exists", "Check"}

	return lo.SomeBy(queryPrefixes, func(prefix string) bool {
		return strings.HasPrefix(methodName, prefix)
	})
}

// generateReadableInterfaces 生成可读实现类代码（支持多个接口）
func generateReadableInterfaces(outputFile string, interfaces []*InterfaceInfo) error {
	if len(interfaces) == 0 {
		return fmt.Errorf("没有接口需要生成")
	}

	var sb strings.Builder

	// 写入文件头注释
	sb.WriteString(fmt.Sprintf("// Code generated by itfgen. DO NOT EDIT.\n\n"))

	// 写入包声明 - 使用第一个接口的包名
	sb.WriteString(fmt.Sprintf("package %s\n\n", interfaces[0].Package))

	// 为每个接口生成代码
	for i, interfaceInfo := range interfaces {
		if i > 0 {
			sb.WriteString("\n")
		}

		// 写入接口断言
		implName := "readable" + interfaceInfo.Name
		sb.WriteString(fmt.Sprintf("var _ %s = (*%s)(nil)\n\n", interfaceInfo.Name, implName))

		// 写入结构体定义
		sb.WriteString(fmt.Sprintf("type %s struct {\n", implName))
		sb.WriteString(fmt.Sprintf("\t%s\n", interfaceInfo.Name))
		sb.WriteString("}\n\n")

		// 写入构造函数
		constructorName := "new" + interfaceInfo.Name
		sb.WriteString(fmt.Sprintf("func %s(input %s) %s {\n", constructorName, interfaceInfo.Name, interfaceInfo.Name))
		sb.WriteString(fmt.Sprintf("\treturn &%s{input}\n", implName))
		sb.WriteString("}\n\n")

		// 为每个非查询方法生成空实现
		for _, method := range interfaceInfo.Methods {
			if method.IsQuery {
				// 查询方法不生成实现
				continue
			}

			// 生成方法实现
			sb.WriteString(fmt.Sprintf("func (n *%s) %s {\n", implName, method.Signature))

			// 根据返回值数量生成不同的return语句
			if len(method.Returns) == 0 {
				sb.WriteString("\treturn\n")
			} else if len(method.Returns) == 1 {
				if method.Returns[0] == "error" {
					sb.WriteString("\treturn nil\n")
				} else {
					sb.WriteString("\treturn\n")
				}
			} else {
				// 多个返回值，生成对应的零值
				var returnValues []string
				for _, ret := range method.Returns {
					if strings.Contains(ret, "error") {
						returnValues = append(returnValues, "nil")
					} else if strings.HasPrefix(ret, "*") {
						returnValues = append(returnValues, "nil")
					} else if strings.HasPrefix(ret, "[]") {
						returnValues = append(returnValues, "nil")
					} else if strings.HasPrefix(ret, "map") {
						returnValues = append(returnValues, "nil")
					} else if ret == "string" {
						returnValues = append(returnValues, `""`)
					} else if ret == "int" || ret == "int64" || ret == "uint64" {
						returnValues = append(returnValues, "0")
					} else if ret == "bool" {
						returnValues = append(returnValues, "false")
					} else {
						returnValues = append(returnValues, "nil")
					}
				}
				sb.WriteString(fmt.Sprintf("\treturn %s\n", strings.Join(returnValues, ", ")))
			}

			sb.WriteString("}\n\n")
		}
	}

	// 写入文件
	return utils.WriteFormat(outputFile, []byte(sb.String()))
}
