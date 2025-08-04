package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/Xuanwo/gg"
	"github.com/donutnomad/gotoolkit/approveGen/methods"
	utils2 "github.com/donutnomad/gotoolkit/internal/utils"
	xast2 "github.com/donutnomad/gotoolkit/internal/xast"
	"go/ast"
	"go/parser"
	"go/token"
	"iter"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/samber/lo"
	"golang.org/x/tools/go/packages"
)

const AnnotationName = "Approve"

func nameWithoutPoint(name string) string {
	if strings.HasPrefix(name, "*") {
		return name[1:]
	}
	return name
}

type JenStatementSlice []*jen.Statement

func (a JenStatementSlice) As() []jen.Code {
	return lo.Map(a, func(item *jen.Statement, index int) jen.Code {
		return item
	})
}

func GenMethodCallApproval(genMethodName string, addUnmarshalMethodArgs bool, everyMethodSuffix string, methods []MyMethod, codeUnmarshalFailed, codeUnknownMethod any, defaultSuccess bool, getType func(typ ast.Expr, method MyMethod) string) JenStatementSlice {
	var codeUnmarshalFailedC gg.Node
	var codeUnknownMethodC gg.Node
	if v, ok := codeUnmarshalFailed.(string); ok {
		codeUnmarshalFailedC = gg.String(v)
	} else {
		codeUnmarshalFailedC = gg.Lit(v)
	}
	if v, ok := codeUnknownMethod.(string); ok {
		codeUnknownMethodC = gg.String(v)
	} else {
		codeUnknownMethodC = gg.Lit(v)
	}

	var g1 = gg.NewGroup()

	func1 := g1.NewFunction(genMethodName).
		AddParameter("a", "*AllServices").
		AddParameter("ctx", "context.Context").
		AddParameter("method", "string").
		AddParameter("content", "string").
		AddResult("", "BaseResponse[any]")

	switchOp := gg.Switch("method")
	switchOp2 := gg.Switch("method")

	for _, method := range methods {
		var structName = method.OutStructName()
		var methodName = method.GenMethod()

		switchOp.NewCase(gg.Lit(methodName)).AddBody(
			gg.Var().AddDecl("p", structName),
			gg.If(gg.String("err := sonic.Unmarshal([]byte(content), &p); err != nil")).AddBody(
				gg.Return(gg.Call("Fail[any]").AddParameter(codeUnknownMethodC)),
			),
			gg.Return(gg.Call(method.MethodName+everyMethodSuffix).WithOwner("a."+nameWithoutPoint(method.StructName)).AddParameter(
				lo.Map(method.AsParams(func(typ ast.Expr) string {
					return getType(typ, method)
				}), func(item Param, index int) any {
					if item.Type == "context.Context" {
						return gg.String("ctx")
					} else {
						return gg.String("p.%s", item.Name.UpperCamelCase())
					}
				})...,
			).AddCall("ToAny")),
		)

		switchOp2.NewCase(gg.Lit(methodName)).AddBody(
			gg.Var().AddDecl("p", structName),
			gg.If(gg.String("err := sonic.Unmarshal([]byte(content), &p); err != nil")).AddBody(
				gg.Return(gg.String("nil, err")),
			),
			gg.Return(gg.String("&p, nil")),
		)
	}

	if defaultSuccess {
		switchOp.NewDefault().AddBody(gg.Return(gg.Call("Success[any]").AddParameter(gg.String("struct{}{}"))))
	} else {
		switchOp.NewDefault().AddBody(gg.Return(gg.Call("Fail[any]").AddParameter(codeUnmarshalFailedC)))
	}
	switchOp2.NewDefault().AddBody(gg.Return(gg.String("nil, nil")))

	func1.AddBody(switchOp)

	res := jen.Id(g1.String())
	var output = []*jen.Statement{res}
	if addUnmarshalMethodArgs {
		func2 := gg.NewGroup()
		func2.NewFunction("UnmarshalMethodArgs").
			AddParameter("method", "string").
			AddParameter("content", "string").
			AddResult("", "any").AddResult("", "error").AddBody(
			switchOp2,
		)
		unmarshalMethodArgs := jen.Id(func2.String())

		output = append(output, jen.Line(), unmarshalMethodArgs)
	}
	return output
}

func genGlobalFunc(comment string, method *MyMethod, formatFunctionName func(name string) string, getNameFunc func(typ ast.Expr, imports xast2.ImportInfoSlice) string) jen.Code {
	// Extract function name from comment
	funcName := strings.Trim(strings.Split(comment, "=")[1], "\"")

	g := gg.NewGroup()
	func1 := g.NewFunction(formatFunctionName(funcName))
	// 参数
	for _, param := range method.MethodParams {
		for _, name := range param.Names {
			func1.AddParameter(name.Name, getNameFunc(param.Type, method.Imports))
		}
	}
	// 返回值
	for _, result := range method.MethodResults {
		resultType := getNameFunc(result.Type, method.Imports)
		if len(result.Names) == 0 {
			func1.AddResult("", resultType)
		} else {
			for _, name := range result.Names {
				func1.AddResult(name.Name, resultType)
			}
		}
	}
	func1.AddBody(
		gg.Return(
			gg.Call(method.MethodName).AddParameter(lo.Map(method.MethodParams, func(param *ast.Field, _ int) any {
				return gg.String(param.Names[0].Name)
			})...),
		),
	)

	return jen.Id(g.String())
}

var (
	paths           = flag.String("path", "", "dir paths, separated by comma")
	outputFileName_ = flag.String("out", "", "output filename")
)

func main() {
	flag.Parse()
	if *paths == "" || *outputFileName_ == "" {
		fmt.Println("type parameter is required")
		return
	}

	var pathList = strings.Split(*paths, ",")
	// Trim spaces from each path
	for i, path := range pathList {
		pathList[i] = strings.TrimSpace(path)
	}
	var outputFileName = *outputFileName_

	var files []string
	for _, pwd := range pathList {
		_files := getFiles(pwd)
		files = append(files, _files...)
	}
	if len(files) == 0 {
		return
	}
	pkgPath := lo.Must1(GetFullPathWithPackage(files[0]))

	var fSet = token.NewFileSet()
	var importMgr = xast2.NewImportManager(pkgPath)
	var allMethods = MyMethodSlice{}

	extractor := NewAnnotationExtractor("@" + AnnotationName)
	var ch = make(chan MyMethodSlice)
	for _, file := range files {
		go func() {
			ch <- extractor.ExtractMethods(fSet, file)
		}()
	}
	for i := 0; i < len(files); i++ {
		_methods := <-ch
		allMethods = append(allMethods, _methods...)
	}
	notStructMethods := lo.Filter(allMethods, func(item MyMethod, index int) bool {
		return item.Recv == nil
	})
	allMethods = lo.Filter(allMethods, func(item MyMethod, index int) bool {
		return item.Recv != nil
	})
	// sort methods
	sort.Slice(allMethods, func(i, j int) bool {
		return allMethods[i].GenMethod() < allMethods[j].GenMethod()
	})

	methodsMap := allMethods.ToMap()
	var getNameFunc = func(typ ast.Expr, imports xast2.ImportInfoSlice) string {
		return xast2.GetFieldType(typ, func(expr *ast.SelectorExpr) string {
			x := expr.X.(*ast.Ident).Name // mo
			alias, _ := importMgr.GetAliasAndPath(imports.Find(x).GetPath())
			return alias
		})
	}
	// 导入import
	importMgr.AddImport("fmt")
	importMgr.AddImport("strings")
	for _, method := range utils2.IterSortMap(methodsMap) {
		for _, item := range method.ExtractImportPath() {
			importMgr.AddImport(item)
		}
	}
	importMgr.AddImport("github.com/bytedance/sonic")
	// 字段格式化方法
	var formatFunctionBy = func(name string) string {
		if len(name) == 0 {
			return ""
		}
		return "_ApprovedFunc_" + name
	}
	codes := jen.NewFile(allMethods[0].FilePkgName)

	codes.PackageComment("Code generated by approveGen. DO NOT EDIT.")
	codes.PackageComment("Each method returns a slice of values for the corresponding field.")
	codes.Line()
	// imports
	codes.Id("import").DefsFunc(func(group *jen.Group) {
		for _, info := range importMgr.Iter() {
			if info.HasAlias() {
				group.Id(info.GetAlias()).Lit(info.GetPath())
			} else {
				group.Lit(info.GetPath())
			}
		}
	})
	codes.Line()

	// 代码生成

	// 生成字段格式化方法
	// 获取func的template定义
	var funcTemplateMapping = make(map[string]string)
	for _, method := range utils2.IterSortMap(notStructMethods.ToMap()) {
		comments := lo.Must1(method.FindAnnoBody(AnnotationName))
		for _, comment := range comments {
			switch {
			case strings.HasPrefix(comment, "global::func"):
				fmt.Println("Global Function:", comments, method.PkgPath, method.MethodName)
				codes.Add(genGlobalFunc(comment, &method, formatFunctionBy, getNameFunc))
			case strings.HasPrefix(comment, "global::template"):
				parts := strings.Split(comment, "=")
				funcName := strings.Trim(parts[1], "\"")
				funcTemplate := strings.Trim(parts[2], "\"")
				funcTemplateMapping[funcName] = funcTemplate
			}
		}
	}

	var hookRejectedMethods MyMethodSlice

	var isIgnoreType = func(ty string) bool {
		return ty == "context.Context"
	}

	// 处理每个加了@Approve注释的方法
	for _, method1 := range utils2.IterSortMap(methodsMap) {
		if !method1.IsStructMethod() {
			panic("暂时不能给不是结构体的方法设置")
		}

		var getNameFunc2 = func(typ ast.Expr) string {
			return getNameFunc(typ, method1.Imports)
		}

		methodParams := method1.MethodParams
		methodParams = lo.Filter(methodParams, func(item *ast.Field, index int) bool {
			for _, name := range item.Names {
				if name.Name == "_" {
					return false
				}
			}
			return true
		})
		method := method1.Copy()
		method.MethodParams = methodParams

		///////// 生成方法结构体
		codes.Comment(fmt.Sprintf("========================== %s ==========================", method.OutStructName())).Line()
		codes.Add(jen.Type().Id(method.OutStructName()).StructFunc(func(s *jen.Group) {
			for _, param := range methodParams {
				tn := getNameFunc2(param.Type)
				// 结构体不存储context.Context对象
				if isIgnoreType(tn) {
					continue
				}
				for _, name := range param.Names {
					s.Id(utils2.UpperCamelCase(name.Name)).Qual("", tn)
				}
			}
		}))

		// 获取所有该方法的注释
		var bodies = lo.Must1(method.FindAnnoBody(AnnotationName))

		// 生成方法
		var receiver, structName = "p", method.OutStructName()

		var args = method.AsParams(func(typ ast.Expr) string {
			return getNameFunc(typ, method.Imports)
		})

		var methodCodes []jen.Code
		for _, body := range bodies {
			// 生成方法 String()
			if info := methods.ParseStringMethod(body); info != nil {
				// 解析出所有的args::field的控制语句
				fields := methods.ParseFieldMethod(bodies)
				argsFilter := lo.Filter(args, func(param Param, index int) bool {
					return !isIgnoreType(string(param.Type))
				})
				out := info.Generator().WithMethod("String").Generate(receiver, structName, lo.Map(argsFilter, func(p Param, idx int) methods.ArgInfo {
					key := fields.GetName(p.Name).UpperCamelCase()
					placeholder := p.Type.Placeholder()
					fieldFormatFunc := fields.GetFunction(p.Name)
					if fieldFormatFunc != "" {
						placeholder = "%s"
					}
					var mapping = []string{
						"$key", key.String(),
						"$value", placeholder,
						"$idx", strconv.Itoa(idx),
					}
					return methods.ArgInfo{
						Template:   strings.NewReplacer(mapping...).Replace(info.ArgsTemplate),
						Field:      p.Name.UpperCamelCase().String(),
						FormatFunc: formatFunctionBy(fieldFormatFunc.String()),
						IsPtr:      p.Type.IsPtr(),
					}
				}))
				methodCodes = append(methodCodes, out)
			}
			// 生成方法 Note()
			if info := methods.ParseNoteMethod(body); info != nil {
				methodCodes = append(methodCodes, info.Generator().Generate(receiver, structName))
			}
			// 生成方法 Json()
			if info := methods.ParseJsonMethod(body); info != nil {
				methodCodes = append(methodCodes, info.Generator().Generate(receiver, structName))
			}
			// 为对象结构体生成自定义方法
			if info := methods.ParseFuncMethod(body); info != nil {
				template, ok := funcTemplateMapping[info.Name]
				if !ok {
					panic(fmt.Sprintf("func: %s 's template is not define", info.Name))
				}
				var returnString = genMethodParamsString(method.MethodResults, true, getNameFunc2)

				// 结构体定义, 不要存储context.Context
				methodStructArgCode := jen.Id("&").Id(method.OutStructName()).BlockFunc(func(group *jen.Group) {
					for _, param := range methodParams {
						tn := getNameFunc2(param.Type)
						if isIgnoreType(tn) {
							continue
						}
						for _, name := range param.Names {
							group.Id(utils2.UpperCamelCase(name.Name)).Id(":").Id(name.Name).Op(",")
						}
					}
				})

				var methodArgs = genMethodParamsString(methodParams, false, getNameFunc2)
				var methodArgNames []string
				for _, param := range methodParams {
					for _, name := range param.Names {
						methodArgNames = append(methodArgNames, name.Name)
					}
				}

				methodCodes = append(methodCodes, info.Generator().Generate(template, method.ObjName, method.StructName, method.MethodName, methodArgNames, methodArgs, methodStructArgCode.GoString(), returnString))
			}
			// 生成hookRejected内容
			if strings.HasPrefix(body, "func::hookRejected") {
				hookRejectedMethods = append(hookRejectedMethods, method1.Copy())
			}
		}

		// 生成方法 MethodName()
		_m := methods.NoteMethod{
			Info: &methods.NoteMethodInfo{Note: fmt.Sprintf("%s_%s", method.StructNameWithoutPtr(), method.MethodName)},
		}
		methodCodes = append(methodCodes, _m.WithMethod("MethodName").Generate(receiver, structName))

		codes.Add(methodCodes...)
	}

	codes.Line()
	m1 := GenMethodCallApproval("CallMethodForApproval", true, "", allMethods, "CodeUnmarshalFailed", "CodeUnknownMethod", false, func(typ ast.Expr, method MyMethod) string {
		return getNameFunc(typ, method.Imports)
	})
	codes.Add(m1.As()...)
	fmt.Println("-===============")
	m2 := GenMethodCallApproval("CallMethodForApprovalHookRejected", false, "HookRejected", hookRejectedMethods, "CodeUnmarshalFailed", "CodeUnknownMethod", true, func(typ ast.Expr, method MyMethod) string {
		return getNameFunc(typ, method.Imports)
	})
	codes.Add(m2.As()...)

	// 保存到当前工作目录
	err := codes.Save(outputFileName)
	if err != nil {
		panic(err)
	}
	pwd, _ := os.Getwd()
	fmt.Println("Success:", filepath.Join(pwd, outputFileName))
}

func genMethodParamsString(fields []*ast.Field, isResult bool, nameFor func(ast.Expr) string) string {
	var returnString string
	if isResult && len(fields) == 1 && len(fields[0].Names) == 0 {
		//returnString += "("
		//returnString += "ctx context.Context, "
		returnString = nameFor(fields[0].Type)
		//returnString += ")"
	} else {
		returnString += "("
		//returnString += "ctx context.Context, "
		for idx1, param := range fields {
			tn := nameFor(param.Type)
			if len(param.Names) == 0 {
				returnString += tn
			} else {
				for idx, name := range param.Names {
					returnString += name.Name
					if idx != len(param.Names)-1 {
						returnString += ", "
					} else {
						returnString += " "
					}
				}
				returnString += tn
			}
			if idx1 != len(fields)-1 {
				returnString += ", "
			}
		}
		returnString += ")"
	}
	return returnString
}

type AnnotationExtractor struct {
	AnnotationName string
}

func NewAnnotationExtractor(annotationName string) *AnnotationExtractor {
	return &AnnotationExtractor{AnnotationName: annotationName}
}

func (e *AnnotationExtractor) MethodsIter(file *ast.File) iter.Seq[MyMethod] {
	return func(yield func(MyMethod) bool) {
		ast.Inspect(file, func(n ast.Node) bool {
			if fn, ok := n.(*ast.FuncDecl); ok {
				if hasComment(fn.Doc, e.AnnotationName) {
					var objName, structName string = getObjectName(fn.Recv)
					var sig = MyMethod{
						ObjName:    objName,
						StructName: structName,
						MethodName: fn.Name.Name,
						Func:       fn,
						Recv:       fn.Recv,
						Comment: lo.Map(fn.Doc.List, func(item *ast.Comment, index int) string {
							return item.Text
						}),
						StartPos: int(fn.Pos()),
						EndPos:   int(fn.End()),
					}
					if fn.Type.Params != nil {
						sig.MethodParams = fn.Type.Params.List
					}
					if fn.Type.Results != nil {
						sig.MethodResults = fn.Type.Results.List
					}
					if !yield(sig) {
						return false
					}
				}
			}
			return true
		})
	}
}

func (e *AnnotationExtractor) ExtractMethods(fSet *token.FileSet, filename string) []MyMethod {
	file := lo.Must1(parser.ParseFile(fSet, filename, nil, parser.AllErrors|parser.ParseComments))
	pkgPath, err := GetFullPathWithPackage(filename)
	if err != nil {
		panic(err)
	}
	importInfos := new(xast2.ImportInfoSlice).From(file.Imports)
	methods_ := slices.Collect(e.MethodsIter(file))
	for i, method := range methods_ {
		method.PkgPath = pkgPath
		method.Imports = importInfos
		method.FilePkgName = file.Name.Name
		methods_[i] = method
	}
	return methods_
}

func GetFullPathWithPackage(filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	// Configure package loading
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedModule,
		Dir:  filepath.Dir(absPath),
		Env:  append(os.Environ(), "GO111MODULE=on"),
	}

	// Load the package containing the file
	pkgs, err := packages.Load(cfg, "file="+absPath)
	if err != nil {
		return "", err
	}

	if len(pkgs) == 0 {
		return "", fmt.Errorf("no package found for file: %s", filePath)
	}

	pkg := pkgs[0]

	importPath := pkg.PkgPath
	if importPath == "" {
		if pkg.Module != nil {
			dir := filepath.Dir(absPath)
			relPath, err := filepath.Rel(pkg.Module.Dir, dir)
			if err == nil {
				importPath = filepath.Join(pkg.Module.Path, relPath)
			}
		}
	}

	return importPath, nil
}

type MyMethodSlice []MyMethod

func (s MyMethodSlice) ToMap() map[string]MyMethod {
	return lo.SliceToMap(s, func(item MyMethod) (string, MyMethod) {
		return item.OutStructName(), item
	})
}

type MyMethod struct {
	ObjName    string // (p *Struct) ==> p
	StructName string // (p *Struct) ==> *Struct

	MethodName    string
	MethodParams  []*ast.Field
	MethodResults []*ast.Field

	Func     *ast.FuncDecl
	Comment  []string
	StartPos int
	EndPos   int
	Recv     *ast.FieldList

	Imports     xast2.ImportInfoSlice
	PkgPath     string
	FilePkgName string
}

func (m *MyMethod) Copy() MyMethod {
	return MyMethod{
		ObjName:       m.ObjName,
		StructName:    m.StructName,
		MethodName:    m.MethodName,
		MethodParams:  m.MethodParams,
		MethodResults: m.MethodResults,
		Func:          m.Func,
		Comment:       m.Comment,
		StartPos:      m.StartPos,
		EndPos:        m.EndPos,
		Recv:          m.Recv,
		Imports:       m.Imports,
		PkgPath:       m.PkgPath,
		FilePkgName:   m.FilePkgName,
	}
}

func (m *MyMethod) ExtractImportPath() []string {
	var newSlice []*ast.Field
	newSlice = append(newSlice, m.MethodParams...)
	newSlice = append(newSlice, m.MethodResults...)

	var out []string
	for _, param := range newSlice {
		xast2.GetFieldType(param.Type, func(expr *ast.SelectorExpr) string {
			x := expr.X.(*ast.Ident).Name // mo
			out = append(out, m.Imports.Find(x).GetPath())
			return ""
		})
	}

	return out
}

func (m *MyMethod) GenMethod() string {
	return fmt.Sprintf("%s_%s", m.StructNameWithoutPtr(), m.MethodName)
}

func (m *MyMethod) StructNameWithoutPtr() string {
	return parseString(m.StructName)
}

func (m *MyMethod) AsParams(getType func(typ ast.Expr) string) []Param {
	var args []Param
	for _, p := range m.MethodParams {
		for _, name := range p.Names {
			args = append(args, Param{
				Name: utils2.EString(name.Name),
				Type: Type(getType(p.Type)),
			})
		}
	}
	return args
}

func (m *MyMethod) IsStructMethod() bool {
	return m.StructName != ""
}

func (m *MyMethod) FindAnnoBody(name string) ([]string, error) {
	var out = make([]string, 0, len(m.Comment))
	for _, comment := range m.Comment {
		comment = strings.TrimSpace(strings.TrimPrefix(comment, "//"))
		if !strings.HasPrefix(comment, "@"+name) {
			continue
		}
		comment = comment[len("@"+name):]
		if len(comment) < 2 {
			continue
		}
		if comment[0] != '(' && comment[len(comment)] != ')' {
			return nil, errors.New("invalid syntax")
		}
		comment = strings.TrimSpace(comment[1 : len(comment)-1])
		if len(comment) == 0 {
			continue
		}
		out = append(out, comment)
	}
	return out, nil
}

// OutStructName 最终生成的结构体的名称
func (m *MyMethod) OutStructName() string {
	var structName = m.StructName
	if strings.HasPrefix(structName, "*") {
		structName = structName[1:]
	}
	return fmt.Sprintf("_%sMethod%s", structName, m.MethodName)
}

func getObjectName(list *ast.FieldList) (objName, structName string) {
	var getName = func(input []*ast.Ident) string {
		if len(input) > 0 {
			return input[0].Name
		}
		return ""
	}
	if list == nil {
		return "", ""
	}
	for _, field := range list.List {
		objName = getName(field.Names)
		structName = xast2.GetFieldType(field.Type, nil)
	}
	return
}

func parseString(input string) string {
	var out = input
	if len(input) >= 2 && input[0] == '"' && input[len(input)-1] == '"' {
		out = input[1 : len(input)-1]
	}
	if strings.HasPrefix(out, "*") {
		return out[1:]
	}
	return out
}

func getFiles(pwd string) []string {
	var files []string
	if err := filepath.Walk(pwd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".go" {
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			files = append(files, path)
		}
		return nil
	}); err != nil {
		fmt.Printf("visit dir files failed: %s", err.Error())
		os.Exit(1)
	}
	return files
}

func hasComment(comment *ast.CommentGroup, target string) bool {
	if comment == nil {
		return false
	}
	for _, c := range comment.List {
		var text = c.Text
		if strings.HasPrefix(c.Text, "//") {
			text = text[2:]
		}
		if strings.HasPrefix(strings.TrimSpace(text), target) {
			return true
		}
	}
	return false
}

type Param struct {
	Name utils2.EString
	Type Type // mo.Option[bool]
}
type Type string

func (t Type) IsPtr() bool {
	return strings.HasPrefix(string(t), "*")
}

func (t Type) NoPtr() Type {
	if strings.HasPrefix(string(t), "*") {
		return t[1:]
	}
	return t
}

func (t Type) Placeholder() string {
	typ := string(t.NoPtr())
	if lo.Contains([]string{"int", "int8", "int16", "int32", "int64"}, typ) {
		return "%d"
	}
	if lo.Contains([]string{"uint", "uint8", "uint16", "uint32", "uint64"}, typ) {
		return "%d"
	}
	if typ == "string" {
		return "%s"
	}
	if lo.Contains([]string{"float32", "float64"}, typ) {
		return "%f"
	}
	return "%v"
}
