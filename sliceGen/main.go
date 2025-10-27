package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/donutnomad/gotoolkit/internal/utils"
	xast2 "github.com/donutnomad/gotoolkit/internal/xast"

	"github.com/samber/lo"
	"golang.org/x/exp/maps"
)

var (
	typeName_     = flag.String("type", "", "struct name to process, format: [package_path/]struct_name1,struct_name2...")
	ignoreFields  = flag.String("ignoreFields", "", "fields to ignore, comma separated")
	includeFields = flag.String("includeFields", "", "fields to include (takes precedence over ignoreFields), comma separated")
	extraMethods  = flag.String("methods", "", "extra methods to generate: filter,map,reduce,sort,groupby (comma separated)")
)

type StructType = xast2.StructType

// ImportInfo tracks package imports and their aliases
type ImportInfo = xast2.ImportInfo

type IExecute interface {
	Execute(w io.Writer, data any, addImport func(path string)) error
	Comment() string
}

type TypeSpecWithFile struct {
	*ast.TypeSpec
	File *ast.File
}

// Generator struct with version compatibility
type Generator struct {
	typeNames     []string
	packagePath   string
	ignoreFields  map[string]bool
	includeFields map[string]bool
	extraMethods  map[string]IExecute
	importMg      *xast2.ImportManager         // key: full import path, value: import info
	typeSpecs     map[string]*TypeSpecWithFile // 缓存所有类型定义
}

// NewGenerator creates a new generator instance with version checks
func NewGenerator(typeNames []string, packagePath string, ignoreFields, includeFields map[string]bool, extraMethods map[string]IExecute) *Generator {
	return &Generator{
		typeNames:     typeNames,
		packagePath:   packagePath,
		ignoreFields:  ignoreFields,
		includeFields: includeFields,
		extraMethods:  extraMethods,
		importMg:      xast2.NewImportManager(packagePath),
		typeSpecs:     make(map[string]*TypeSpecWithFile),
	}
}

func main() {
	flag.Parse()
	if err := run(*typeName_, *ignoreFields, *includeFields, *extraMethods); err != nil {
		panic(err)
	}
}

func run(typeName, ignoreFields, includeFields, methods string) error {
	if typeName == "" {
		return fmt.Errorf("type parameter is required")
	}

	// Parse type name and package path
	var packagePath string
	var structNames []string

	parts := strings.Split(typeName, "/")
	if len(parts) == 1 {
		structNames = strings.Split(parts[0], ",")
		packagePath = "."
	} else {
		structNames = strings.Split(parts[len(parts)-1], ",")
		packagePath = strings.Join(parts[:len(parts)-1], "/")
	}

	// Clean struct names
	for i, name := range structNames {
		structNames[i] = strings.TrimSpace(name)
	}

	// Parse include fields (takes precedence over ignoreFields)
	includeFieldsMap := make(map[string]bool)
	if includeFields != "" {
		for _, f := range strings.Split(includeFields, ",") {
			includeFieldsMap[strings.TrimSpace(f)] = true
		}
	}

	// Parse ignore fields (only used if includeFields is empty)
	ignoreFieldsMap := make(map[string]bool)
	if includeFields == "" && ignoreFields != "" {
		for _, f := range strings.Split(ignoreFields, ",") {
			ignoreFieldsMap[strings.TrimSpace(f)] = true
		}
	}

	var allExtraMethods = utils.DefSlice(MethodFilter, MethodMap, MethodGroupBy, MethodReduce, MethodSort)

	// Parse extra methods
	var methodsList []string
	if methods != "" {
		methodsList = strings.Split(methods, ",")
		for _, method := range methodsList {
			if !lo.ContainsBy(allExtraMethods, func(item MyGenerator[MethodTemplateData]) bool {
				return strings.EqualFold(item.Name, method)
			}) {
				return fmt.Errorf("unknown extra method: %s", method)
			}
		}
	}
	extraMethods := lo.SliceToMap(methodsList, func(method string) (string, IExecute) {
		v, _ := lo.Find(allExtraMethods, func(item MyGenerator[MethodTemplateData]) bool {
			return strings.EqualFold(item.Name, method)
		})
		return method, &v
	})

	g := NewGenerator(structNames, packagePath, ignoreFieldsMap, includeFieldsMap, extraMethods)

	// Parse all .go files in the specified directory
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, g.packagePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// Traverse packages
	foundTypes := make(map[string]*xast2.StructType)
	var packageName string

	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for name, structType := range g.findStructTypes(file) {
				foundTypes[name] = structType
				packageName = file.Name.Name
			}
		}
	}

	// Check if all requested types are found
	for _, typeName := range g.typeNames {
		if _, ok := foundTypes[typeName]; !ok {
			return fmt.Errorf("struct %s not found in directory %s", typeName, g.packagePath)
		}
	}

	// Generate code
	return g.generateCode(foundTypes, packageName, "slice_generated.go")
}

func (g *Generator) findStructTypes(file *ast.File) map[string]*xast2.StructType {
	foundTypes := make(map[string]*xast2.StructType)

	// 首先收集所有类型定义
	ast.Inspect(file, func(n ast.Node) bool {
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			g.typeSpecs[typeSpec.Name.Name] = &TypeSpecWithFile{typeSpec, file}
		}
		return true
	})

	// 然后找到我们要处理的结构体
	ast.Inspect(file, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		for _, name := range g.typeNames {
			if typeSpec.Name.Name == name {
				foundTypes[name] = &StructType{StructType: structType, Imports: file.Imports}
				break
			}
		}

		return true
	})

	return foundTypes
}

// StructTypeFields 一个结构体中所有的字段
type StructTypeFields struct {
	*xast2.StructType
	Fields map[string]*xast2.MyField
}

func (g *Generator) generateCode(types map[string]*xast2.StructType, packageName, outputFileName string) error {
	// Generate output file
	outputDir := lo.Ternary(g.packagePath == ".", "", g.packagePath)
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}

	outputName := filepath.Join(outputDir, outputFileName)

	// Use bytes.Buffer to store generated code
	var buf strings.Builder

	// Add default imports
	g.importMg.AddImport("github.com/samber/lo")

	var types2 = make(map[string]StructTypeFields)

	// First pass: collect all imports from all types and fields
	// 收集所有的类型和里面的所有字段
	for key, structType := range utils.IterSortMap(types) {
		var fields = make(map[string]*xast2.MyField)
		if err := g.collectFields(structType, fields); err != nil {
			return err
		}
		fields = utils.FilterMapEntries(fields, func(k string, v *xast2.MyField) (string, *xast2.MyField, bool) {
			return k, v, ast.IsExported(k) && g.shouldIncludeField(k)
		})
		types2[key] = StructTypeFields{structType, fields}
		g.addImports(lo.Flatten(lo.Map(maps.Values(utils.CollectMap(utils.IterSortMap(fields))), func(item *xast2.MyField, index int) []string {
			return item.CollectImports()
		})))
	}

	// Generate code for each type, 为每个结构体生成新的Slice结构体和方法
	for typeName, structType := range utils.IterSortMap(types2) {
		sliceTypeName := typeName + "Slice"
		// Write slice type definition
		Fprintf(&buf, "type %s []%s\n\n", sliceTypeName, typeName)

		// Generate field methods
		for fieldName, field := range utils.IterSortMap(structType.Fields) {
			data := FieldTemplateData{
				TypeName:     sliceTypeName,
				TypeItemName: typeName,
				FieldName:    fieldName,
				FieldType:    g.getFieldType(structType.StructType, field.Type),
			}
			for _, impl := range utils.DefSlice(MethodMapField, MethodField) {
				if err := impl.Generate(&buf, data, g.importMg.AddImport); err != nil {
					return err
				}
				Fprintf(&buf, "\n")
			}
		}

		// Generate extra helper methods
		if err := g.generateExtraMethodsToBuffer(&buf, sliceTypeName, typeName); err != nil {
			return err
		}
	}

	bodyString := buf.String()
	buf = strings.Builder{}

	// Write file header comments
	Fprintf(&buf, "// Code generated by sliceGen. DO NOT EDIT.\n")
	Fprintf(&buf, "//\n")
	Fprintf(&buf, "// This file contains slice helper methods for types: %s\n", strings.Join(g.typeNames, ", "))
	Fprintf(&buf, "// Each method returns a slice of values for the corresponding field.\n")
	Fprintf(&buf, "//\n")
	Fprintf(&buf, "// Example usage:\n")
	Fprintf(&buf, "//   var slice TypeSlice = []Type{...}\n")
	Fprintf(&buf, "//   values := slice.FieldName()\n\n")
	// Write package declaration
	Fprintf(&buf, "package %s\n\n", packageName)

	// Write imports
	Fprintf(&buf, "import (\n")
	for _, info := range g.importMg.Iter() {
		Fprintf(&buf, "\t%s\n", info.String())
	}
	Fprintf(&buf, ")\n\n")
	Fprintf(&buf, bodyString)

	// Format the generated code
	formattedCode, err := format.Source([]byte(buf.String()))
	if err != nil {
		return fmt.Errorf("failed to format generated code: %v", err)
	}

	// Write the formatted code to file
	if err := os.WriteFile(outputName, formattedCode, 0644); err != nil {
		return fmt.Errorf("failed to write generated code: %v", err)
	}

	return nil
}

// 收集结构体内部的字段到map中
// 如果结构体内部有嵌套结构体，那么会递归调用
func (g *Generator) collectFields(structType *StructType, fields map[string]*xast2.MyField) error {
	var collectFieldsFromType func(fieldType ast.Expr) error
	collectFieldsFromType = func(fieldType ast.Expr) error {
		switch t := fieldType.(type) {
		case *ast.Ident:
			// 如果是标识符，可能是本地定义的类型，需要解析
			obj, ok := g.typeSpecs[t.Name]
			if !ok {
				return nil
			}
			if structType, ok := obj.Type.(*ast.StructType); ok {
				for _, field := range structType.Fields.List {
					if field.Names == nil { // 递归处理嵌入字段
						if err := collectFieldsFromType(field.Type); err != nil {
							return err
						}
					} else {
						// 只有在字段不存在时才添加，这样保证外层字段优先
						if _, exists := fields[field.Names[0].Name]; !exists {
							fields[field.Names[0].Name] = &xast2.MyField{
								Field: field,
								StructType: &StructType{
									StructType: structType, Imports: obj.File.Imports,
								},
							}
						}
					}
				}
			}
		case *ast.SelectorExpr:
			// 处理来自其他包的类型
			// TODO: 实现对外部包类型的解析
			return nil
		}
		return nil
	}

	// 处理当前结构体的字段
	for _, field := range structType.Fields.List {
		if field.Names == nil {
			// 处理嵌入字段
			if err := collectFieldsFromType(field.Type); err != nil {
				return err
			}
		} else {
			// 普通字段直接添加，会覆盖同名的嵌入字段
			fields[field.Names[0].Name] = &xast2.MyField{field, structType}
		}
	}

	return nil
}

// 将生成的数据写入到buf中
func (g *Generator) generateExtraMethodsToBuffer(buf *strings.Builder, sliceTypeName, typeName string) error {
	for methodName := range utils.IterSortMap(g.extraMethods) {
		if tmpl, ok := g.extraMethods[methodName]; ok {
			data := MethodTemplateData{
				TypeName:     sliceTypeName,
				TypeItemName: typeName,
				Description:  tmpl.Comment(),
			}
			if err := tmpl.Execute(buf, data, g.importMg.AddImport); err != nil {
				return err
			}
			Fprintf(buf, "\n")
		}
	}
	return nil
}

func (g *Generator) addImports(path []string) {
	for _, item := range path {
		g.importMg.AddImport(item)
	}
}

func (g *Generator) getFieldType(structType *StructType, expr ast.Expr) string {
	return xast2.GetFieldType(expr, func(expr *ast.SelectorExpr) string {
		if pkgPath := structType.GetPkgPathBySelector(expr); pkgPath != "" {
			alias, _ := g.importMg.GetAliasAndPath(pkgPath)
			return alias
		}
		return ""
	})
}

// 修改字段过滤逻辑
func (g *Generator) shouldIncludeField(fieldName string) bool {
	// 如果指定了 includeFields，只处理包含的字段
	if len(g.includeFields) > 0 {
		return g.includeFields[fieldName]
	}
	// 否则使用 ignoreFields 逻辑
	return !g.ignoreFields[fieldName]
}

// MethodTemplateData 定义模板数据结构
type MethodTemplateData struct {
	TypeName     string // 切片类型名称 (例如: UserSlice)
	TypeItemName string // 元素类型名称 (例如: User)
	Description  string // 方法描述
}

// FieldTemplateData 定义字段模板数据结构
type FieldTemplateData struct {
	TypeName     string // 切片类型名称 (例如: UserSlice)
	TypeItemName string // 元素类型名称 (例如: User)
	FieldName    string // 字段名称
	FieldType    string // 字段类型
}

type MyGenerator[T any] struct {
	Name        string
	Description string
	Template    string
	Imports     []string
	_template   *template.Template
}

func (g *MyGenerator[T]) Comment() string {
	return g.Description
}

func (g *MyGenerator[T]) Generate(w io.Writer, data T, addImport func(path string)) error {
	return g.Execute(w, data, addImport)
}

func (g *MyGenerator[T]) Execute(w io.Writer, data any, addImport func(path string)) error {
	if g._template == nil {
		mapperTmpl, err := template.New("").Parse(g.Template)
		if err != nil {
			return err
		}
		g._template = mapperTmpl
	}
	for _, imp := range g.Imports {
		addImport(imp)
	}
	return g._template.Execute(w, data)
}

var MethodMapField = MyGenerator[FieldTemplateData]{
	Template: `
// Map{{.FieldName}} is a mapper function for field {{.FieldName}}
func (s {{.TypeItemName}}) Map{{.FieldName}}(item {{.TypeItemName}}, index int) {{.FieldType}} {
	return item.{{.FieldName}}
}
`,
}

var MethodField = MyGenerator[FieldTemplateData]{
	Template: `
// {{.FieldName}} returns a slice of {{.FieldName}} field values
func (s {{.TypeName}}) {{.FieldName}}() []{{.FieldType}} {
	return lo.Map(s, {{.TypeItemName}}{}.Map{{.FieldName}})
}
`,
}

var MethodFilter = MyGenerator[MethodTemplateData]{
	Name:        "filter",
	Description: "returns a new slice containing only the elements that satisfy the predicate fn",
	Template: `
// Filter {{.Description}}
func (s {{.TypeName}}) Filter(fn func({{.TypeItemName}}) bool) {{.TypeName}} {
	return lo.Filter(s, func(item {{.TypeItemName}}, _ int) bool {
		return fn(item)
	})
}`,
}

var MethodMap = MyGenerator[MethodTemplateData]{
	Name:        "map",
	Description: "transforms each element using the provided function fn",
	Template: `
// Map {{.Description}}
func (s {{.TypeName}}) Map(fn func({{.TypeItemName}}) any) []any {
	return lo.Map(s, func(item {{.TypeItemName}}, _ int) any {
		return fn(item)
	})
}`,
}

var MethodReduce = MyGenerator[MethodTemplateData]{
	Name:        "reduce",
	Description: "reduces the slice to a single value using the provided function fn",
	Template: `
// Reduce {{.Description}}
func (s {{.TypeName}}) Reduce(fn func(acc, curr {{.TypeItemName}}) {{.TypeItemName}}, initial {{.TypeItemName}}) {{.TypeItemName}} {
	return lo.Reduce(s, func(acc {{.TypeItemName}}, item {{.TypeItemName}}, _ int) {{.TypeItemName}} {
		return fn(acc, item)
	}, initial)
}`,
}

var MethodSort = MyGenerator[MethodTemplateData]{
	Name:        "sort",
	Description: "returns a new sorted slice using the provided less function",
	Imports:     []string{"sort"},
	Template: `
// Sort {{.Description}}
func (s {{.TypeName}}) Sort(less func({{.TypeItemName}}, {{.TypeItemName}}) bool) {{.TypeName}} {
	result := append({{.TypeName}}{}, s...)
	sort.Slice(result, func(i, j int) bool {
		return less(result[i], result[j])
	})
	return result
}`,
}

var MethodGroupBy = MyGenerator[MethodTemplateData]{
	Name:        "groupBy",
	Description: "groups elements by the key returned by the fn function",
	Template: `
// GroupBy {{.Description}}
func (s {{.TypeName}}) GroupBy(fn func({{.TypeItemName}}) string) map[string]{{.TypeName}} {
	return lo.GroupBy(s, func(item {{.TypeItemName}}) string {
		return fn(item)
	})
}`,
}

func Fprintf(w io.Writer, format string, a ...any) {
	_, _ = fmt.Fprintf(w, format, a...)
}
