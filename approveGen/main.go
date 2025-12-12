package main

import (
	"bytes"
	"flag"
	"fmt"
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

	"github.com/Xuanwo/gg"
	"github.com/dave/jennifer/jen"
	"github.com/donutnomad/gotoolkit/approveGen/generator"
	"github.com/donutnomad/gotoolkit/approveGen/methods"
	"github.com/donutnomad/gotoolkit/approveGen/types"
	"github.com/donutnomad/gotoolkit/approveGen/utils"
	utils2 "github.com/donutnomad/gotoolkit/internal/utils"
	xast2 "github.com/donutnomad/gotoolkit/internal/xast"
	"github.com/samber/lo"
)

const AnnotationName = "Approve"

// GlobalTemplateInfo 全局模板信息
type GlobalTemplateInfo struct {
	FuncName string
	Template string
	Args     []methods.FuncMethodArg
}

// parseGlobalTemplate 解析全局模板，支持分离的template和args定义
// 支持格式:
// 1. global::template="ApproveFor=模板内容"
// 2. global::template="ApproveFor::args=[参数列表]"
func parseGlobalTemplate(comment string) GlobalTemplateInfo {
	info := GlobalTemplateInfo{}

	// 移除 global::template= 前缀
	content := strings.TrimPrefix(comment, "global::template=")
	content = strings.Trim(content, "\"")

	// 检查是否是args定义
	if strings.Contains(content, "::args=") {
		// args定义格式: "ApproveFor::args=[...]"
		parts := strings.Split(content, "::args=")
		if len(parts) >= 2 {
			info.FuncName = parts[0]

			// 解析args数组
			argsStr := "args=" + parts[1]
			// 移除转义字符
			argsStr = strings.ReplaceAll(argsStr, "\\\"", "\"")
			info.Args = methods.ParseArgsArray(argsStr)
		}
	} else {
		// 模板定义格式: "ApproveFor=模板内容"
		parts := strings.SplitN(content, "=", 2)
		if len(parts) >= 2 {
			info.FuncName = parts[0]
			info.Template = parts[1]
		}
	}

	return info
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
	version2        = flag.Bool("v2", false, "version2")
	version3        = flag.Bool("v3", false, "version3")
	version4        = flag.Bool("v4", false, "version4")
	pkgName         = flag.String("pkgname", "", "package name prefix for MethodName()")
	genMethods      = flag.Bool("methods", true, "generate CallMethodForApproval and CallMethodForApprovalHookRejected methods")
)

func main() {
	flag.Parse()
	if *paths == "" || *outputFileName_ == "" {
		fmt.Println("type parameter is required")
		return
	}

	if *version4 {
		*version3 = true
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
	pkgPath := lo.Must1(utils.GetFullPathWithPackage(files[0]))

	var fSet = token.NewFileSet()
	var importMgr = xast2.NewImportManager(pkgPath)
	var allMethods = types.MyMethodSlice{}

	extractor := NewAnnotationExtractor("@" + AnnotationName)
	var ch = make(chan types.MyMethodSlice)
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
	// 检查是否有formatter方法，如果有就提前添加import
	var formatterMethods types.MyMethodSlice
	for _, method := range utils2.IterSortMap(methodsMap) {
		bodies := lo.Must1(method.FindAnnoBody(AnnotationName))
		formatterName := methods.ParseFormatterMethod(bodies)
		if formatterName != "" {
			formatterMethods = append(formatterMethods, method)
		}
	}

	// 导入import
	importMgr.AddImport("fmt")
	importMgr.AddImport("strings")
	// 如果有formatter方法，添加必要的导入
	if len(formatterMethods) > 0 {
		importMgr.AddImport("context")
		importMgr.AddImport("errors")
	}
	for _, method := range utils2.IterSortMap(methodsMap) {
		for _, item := range method.ExtractImportPath() {
			importMgr.AddImport(item)
		}
	}
	if *genMethods {
		importMgr.AddImport("github.com/bytedance/sonic")
		if *version2 {
			importMgr.AddImport("errors")
		}
	}
	// v3版本的额外imports
	if *version3 && *genMethods {
		importMgr.AddImport("errors")
	}
	// 字段格式化方法
	var formatFunctionBy = func(name string) string {
		if len(name) == 0 {
			return ""
		}
		return "_ApprovedFunc_" + name
	}
	if len(allMethods) == 0 {
		fmt.Println("[approveGen] 未找到方法, skip")
		return
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
	var funcGlobalArgsMapping = make(map[string][]methods.FuncMethodArg) // 新增：全局args映射
	for _, method := range utils2.IterSortMap(notStructMethods.ToMap()) {
		comments := lo.Must1(method.FindAnnoBody(AnnotationName))
		for _, comment := range comments {
			switch {
			case strings.HasPrefix(comment, "global::func"):
				fmt.Println("Global Function:", comments, method.PkgPath, method.MethodName)
				codes.Add(genGlobalFunc(comment, &method, formatFunctionBy, getNameFunc))
			case strings.HasPrefix(comment, "global::template"):
				// 解析全局模板，支持分离的template和args定义
				templateInfo := parseGlobalTemplate(comment)

				// 如果是模板定义，存储到模板映射
				if templateInfo.Template != "" {
					funcTemplateMapping[templateInfo.FuncName] = templateInfo.Template
				}

				// 如果是args定义，存储到args映射
				if len(templateInfo.Args) > 0 {
					funcGlobalArgsMapping[templateInfo.FuncName] = templateInfo.Args
				}
			}
		}
	}

	var hookRejectedMethods types.MyMethodSlice

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
				argsFilter := lo.Filter(args, func(param types.Param, index int) bool {
					return !isIgnoreType(string(param.Type))
				})
				out := info.Generator().WithMethod("String").Generate(receiver, structName, lo.Map(argsFilter, func(p types.Param, idx int) methods.ArgInfo {
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
				tmplStr, ok := funcTemplateMapping[info.Name]
				if !ok {
					panic(fmt.Sprintf("func: %s 's template is not define", info.Name))
				}
				// 如果当前func没有定义args，使用全局args
				if len(info.Args) == 0 {
					if globalArgs, hasGlobalArgs := funcGlobalArgsMapping[info.Name]; hasGlobalArgs {
						info.Args = globalArgs
					}
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
				// 添加一个formatter方法, v4添加
				if *version4 {
					if strings.HasSuffix(methodArgs, ")") {
						methodArgs = methodArgs[:len(methodArgs)-1] + ", formatter IApprovalFormatter" + methodArgs[len(methodArgs)-1:]
					}
					tmplStr = strings.ReplaceAll(tmplStr, "\\n", "\n")
					methodArgNames = append(methodArgNames, "formatter")
				}

				methodCodes = append(methodCodes, info.Generator().Generate(tmplStr, method.ObjName, method.StructName, method.MethodName, methodArgNames, methodArgs, methodStructArgCode.GoString(), returnString, *version2))
			}
			// 生成hookRejected内容
			if strings.HasPrefix(body, "func::hookRejected") {
				hookRejectedMethods = append(hookRejectedMethods, method1.Copy())
			}
		}

		// 生成方法 MethodName()
		var methodNameValue string
		if *pkgName != "" {
			methodNameValue = fmt.Sprintf("%s_%s_%s", *pkgName, method.StructNameWithoutPtr(), method.MethodName)
		} else {
			methodNameValue = fmt.Sprintf("%s_%s", method.StructNameWithoutPtr(), method.MethodName)
		}
		_m := methods.NoteMethod{
			Info: &methods.NoteMethodInfo{Note: methodNameValue},
		}
		methodCodes = append(methodCodes, _m.WithMethod("MethodName").Generate(receiver, structName))

		codes.Add(methodCodes...)
	}

	codes.Line()

	// 根据命令行参数决定是否生成方法调用审批相关方法
	if *genMethods {
		if *version3 {
			// 使用v3版本生成方法，生成ApprovalMethodCaller结构体
			// 创建一个 map 来标记哪些方法支持 HookRejected
			hookRejectedMap := make(map[string]bool)
			for _, method := range hookRejectedMethods {
				hookRejectedMap[method.GenMethod()] = true
			}

			// 将formatterMethods和allMethods合并，确保formatter方法也被包含
			var combinedMethods types.MyMethodSlice
			combinedMethods = append(combinedMethods, allMethods...)
			for _, fMethod := range formatterMethods {
				// 检查是否已经在allMethods中
				found := false
				for _, aMethod := range allMethods {
					if fMethod.GenMethod() == aMethod.GenMethod() {
						found = true
						break
					}
				}
				if !found {
					combinedMethods = append(combinedMethods, fMethod)
				}
			}

			m1 := generator.GenMethodCallApprovalV3("Call", true, *pkgName, "", combinedMethods, func(typ ast.Expr, method MyMethod) string {
				return getNameFunc(typ, method.Imports)
			}, false, hookRejectedMap)
			codes.Add(m1.As()...)
		} else if *version2 {
			// 使用v2版本生成方法，需要传递 hookRejectedMethods 的信息
			// 创建一个 map 来标记哪些方法支持 HookRejected
			hookRejectedMap := make(map[string]bool)
			for _, method := range hookRejectedMethods {
				hookRejectedMap[method.GenMethod()] = true
			}

			m1 := generator.GenMethodCallApprovalV2("CallMethodForApproval", true, "", allMethods, func(typ ast.Expr, method MyMethod) string {
				return getNameFunc(typ, method.Imports)
			}, false, hookRejectedMap)
			codes.Add(m1.As()...)
		} else {
			// 使用v1版本生成方法
			m1 := generator.GenMethodCallApproval("CallMethodForApproval", true, "", allMethods, func(typ ast.Expr, method MyMethod) string {
				return getNameFunc(typ, method.Imports)
			}, false)
			codes.Add(m1.As()...)
			fmt.Println("-===============")
			m2 := generator.GenMethodCallApproval("CallMethodForApprovalHookRejected", false, "HookRejected", hookRejectedMethods, func(typ ast.Expr, method MyMethod) string {
				return getNameFunc(typ, method.Imports)
			}, true)
			codes.Add(m2.As()...)
		}
	}

	// 生成Formatter方法 (v3版本跳过，因为已经在ApprovalMethodCaller中生成)
	if len(formatterMethods) > 0 && !*version3 {
		codes.Line()
		codes.Comment("========================== Formatter Method ==========================").Line()

		formatterGen := methods.NewFormatterMethod()
		for _, method := range formatterMethods {
			bodies := lo.Must1(method.FindAnnoBody(AnnotationName))
			formatterName := methods.ParseFormatterMethod(bodies)

			// 如果没有指定formatter名称或者是DEFAULT，使用方法名
			if formatterName == "" || formatterName == "DEFAULT" {
				formatterName = method.MethodName
			}

			// 收集结构体字段信息
			methodParams := lo.Filter(method.MethodParams, func(item *ast.Field, index int) bool {
				for _, name := range item.Names {
					if name.Name == "_" {
						return false
					}
				}
				return true
			})

			var fields []methods.FormatterFieldInfo
			for _, param := range methodParams {
				tn := getNameFunc(param.Type, method.Imports)
				// 跳过context.Context类型
				if isIgnoreType(tn) {
					continue
				}
				// 添加每个字段
				for _, name := range param.Names {
					fields = append(fields, methods.FormatterFieldInfo{
						Name: utils2.UpperCamelCase(name.Name),
						Type: tn,
					})
				}
			}

			formatterGen.AddMethod(method.OutStructName(), method.MethodName, formatterName, fields)
		}

		codes.Add(formatterGen.Generate())
	}

	// 保存到当前工作目录
	buf := &bytes.Buffer{}
	if err := codes.Render(buf); err != nil {
		panic(err)
	}
	err := utils2.WriteFormat(outputFileName, buf.Bytes())
	if err != nil {
		panic(err)
	}
	pwd, _ := os.Getwd()
	fmt.Println("[approveGen] Success:", filepath.Join(pwd, outputFileName))
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

type MyMethod = types.MyMethod

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
	pkgPath, err := utils.GetFullPathWithPackage(filename)
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
