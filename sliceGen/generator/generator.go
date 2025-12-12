package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/donutnomad/gotoolkit/internal/utils"
	xast2 "github.com/donutnomad/gotoolkit/internal/xast"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"
)

type StructType = xast2.StructType

// TypeSpecWithFile 结构体类型规范和对应的文件
type TypeSpecWithFile struct {
	*ast.TypeSpec
	File *ast.File
}

// GeneratedCode 表示生成的代码，分为三部分
type GeneratedCode struct {
	FileComments string   // 文件头部注释
	PackageName  string   // 包名
	Imports      []string // import 语句列表
	Body         string   // 实际代码体
}

// String 返回完整的生成代码
func (g *GeneratedCode) String() string {
	var buf strings.Builder

	// 文件注释
	if g.FileComments != "" {
		buf.WriteString(g.FileComments)
		buf.WriteString("\n")
	}

	// 包声明
	fmt.Fprintf(&buf, "package %s\n\n", g.PackageName)

	// imports
	if len(g.Imports) > 0 {
		buf.WriteString("import (\n")
		for _, imp := range g.Imports {
			fmt.Fprintf(&buf, "\t%s\n", imp)
		}
		buf.WriteString(")\n\n")
	}

	// 代码体
	buf.WriteString(g.Body)

	return buf.String()
}

// Generator 代码生成器
type Generator struct {
	typeNames     []string
	packagePath   string
	ignoreFields  map[string]bool
	includeFields map[string]bool
	extraMethods  map[string]IExecute
	importMg      *xast2.ImportManager         // key: full import path, value: import info
	typeSpecs     map[string]*TypeSpecWithFile // 缓存所有类型定义
	usePointer    bool                         // 是否生成指针类型
}

// NewGenerator 创建生成器实例
func NewGenerator(typeNames []string, packagePath string, ignoreFields, includeFields map[string]bool, extraMethods map[string]IExecute, usePointer bool) *Generator {
	return &Generator{
		typeNames:     typeNames,
		packagePath:   packagePath,
		ignoreFields:  ignoreFields,
		includeFields: includeFields,
		extraMethods:  extraMethods,
		importMg:      xast2.NewImportManager(packagePath),
		typeSpecs:     make(map[string]*TypeSpecWithFile),
		usePointer:    usePointer,
	}
}

// Generate 生成代码
func (g *Generator) Generate() (*GeneratedCode, error) {
	// Parse all .go files in the specified directory
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, g.packagePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
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
			return nil, fmt.Errorf("struct %s not found in directory %s", typeName, g.packagePath)
		}
	}

	// Generate code
	return g.generateCode(foundTypes, packageName)
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

func (g *Generator) generateCode(types map[string]*xast2.StructType, packageName string) (*GeneratedCode, error) {
	// Add default imports
	g.importMg.AddImport("github.com/samber/lo")

	var types2 = make(map[string]StructTypeFields)

	// First pass: collect all imports from all types and fields
	// 收集所有的类型和里面的所有字段
	for key, structType := range utils.IterSortMap(types) {
		var fields = make(map[string]*xast2.MyField)
		if err := g.collectFields(structType, fields); err != nil {
			return nil, err
		}
		fields = utils.FilterMapEntries(fields, func(k string, v *xast2.MyField) (string, *xast2.MyField, bool) {
			return k, v, ast.IsExported(k) && g.shouldIncludeField(k)
		})
		types2[key] = StructTypeFields{structType, fields}
		g.addImports(lo.Flatten(lo.Map(maps.Values(utils.CollectMap(utils.IterSortMap(fields))), func(item *xast2.MyField, index int) []string {
			return item.CollectImports()
		})))
	}

	// Generate code body
	var bodyBuf strings.Builder

	// Generate code for each type, 为每个结构体生成新的Slice结构体和方法
	for typeName, structType := range utils.IterSortMap(types2) {
		sliceTypeName := typeName + "Slice"
		ptrPrefix := lo.Ternary(g.usePointer, "*", "")
		// Write slice type definition
		fmt.Fprintf(&bodyBuf, "type %s []%s%s\n\n", sliceTypeName, ptrPrefix, typeName)

		// Generate field methods
		for fieldName, field := range utils.IterSortMap(structType.Fields) {
			data := FieldTemplateData{
				TypeName:     sliceTypeName,
				TypeItemName: typeName,
				FieldName:    fieldName,
				FieldType:    g.getFieldType(structType.StructType, field.Type),
				UsePointer:   g.usePointer,
				PtrPrefix:    ptrPrefix,
			}
			for _, impl := range utils.DefSlice(MethodMapField, MethodField) {
				if err := impl.Generate(&bodyBuf, data, g.importMg.AddImport); err != nil {
					return nil, err
				}
				fmt.Fprintf(&bodyBuf, "\n")
			}
		}

		// Generate extra helper methods
		if err := g.generateExtraMethodsToBuffer(&bodyBuf, sliceTypeName, typeName); err != nil {
			return nil, err
		}
	}

	// Generate file comments
	var comments strings.Builder
	fmt.Fprintf(&comments, "// Code generated by sliceGen. DO NOT EDIT.\n")
	fmt.Fprintf(&comments, "//\n")
	fmt.Fprintf(&comments, "// This file contains slice helper methods for types: %s\n", strings.Join(g.typeNames, ", "))
	fmt.Fprintf(&comments, "// Each method returns a slice of values for the corresponding field.\n")
	fmt.Fprintf(&comments, "//\n")
	fmt.Fprintf(&comments, "// Example usage:\n")
	fmt.Fprintf(&comments, "//   var slice TypeSlice = []Type{...}\n")
	fmt.Fprintf(&comments, "//   values := slice.FieldName()\n")

	// Collect imports
	var imports []string
	for _, info := range g.importMg.Iter() {
		imports = append(imports, info.String())
	}

	return &GeneratedCode{
		FileComments: comments.String(),
		PackageName:  packageName,
		Imports:      imports,
		Body:         bodyBuf.String(),
	}, nil
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
			fields[field.Names[0].Name] = &xast2.MyField{Field: field, StructType: structType}
		}
	}

	return nil
}

// 将生成的数据写入到buf中
func (g *Generator) generateExtraMethodsToBuffer(buf *strings.Builder, sliceTypeName, typeName string) error {
	ptrPrefix := lo.Ternary(g.usePointer, "*", "")
	for methodName := range utils.IterSortMap(g.extraMethods) {
		if tmpl, ok := g.extraMethods[methodName]; ok {
			data := MethodTemplateData{
				TypeName:     sliceTypeName,
				TypeItemName: typeName,
				Description:  tmpl.Comment(),
				UsePointer:   g.usePointer,
				PtrPrefix:    ptrPrefix,
			}
			if err := tmpl.Execute(buf, data, g.importMg.AddImport); err != nil {
				return err
			}
			fmt.Fprintf(buf, "\n")
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
