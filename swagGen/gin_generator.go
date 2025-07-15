package main

import (
	"fmt"
	"github.com/donutnomad/gotoolkit/internal/utils"
	parsers "github.com/donutnomad/gotoolkit/swagGen/parser"
	"github.com/samber/lo"
	"golang.org/x/exp/maps"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// convertPathToGinFormat 将 Swagger 路径格式 {param} 转换为 Gin 路径格式 :param
func convertPathToGinFormat(path string) string {
	// 使用正则表达式将 {param} 替换为 :param
	re := regexp.MustCompile(`\{([^}]+)\}`)
	return re.ReplaceAllString(path, ":$1")
}

// NewGinGenerator 创建 Gin 生成器
func NewGinGenerator(collection *InterfaceCollection) *GinGenerator {
	return &GinGenerator{
		collection: collection,
	}
}

// GenerateGinCode 生成 Gin 绑定代码
func (g *GinGenerator) GenerateGinCode(comments map[string]string) (constructCode, code string) {
	var parts []string
	var constructorParts []string

	// 生成Gin.Handler的interface
	var handlerInterface []string

	// 为每个接口生成包装结构体和绑定方法
	for _, iface := range g.collection.Interfaces {

		var handlerItf = make(map[string]struct{})
		var middlewareCount int
		var middlewareMap = make(map[string][]*parsers.MiddleWare)
		var handlerItfName = fmt.Sprintf("%sHandler", iface.Name)

		for _, method := range iface.Methods {
			for _, item := range method.Def {
				if v, ok := item.(*parsers.MiddleWare); ok {
					middlewareMap[method.Name] = append(middlewareMap[method.Name], v)
					middlewareCount++
					for _, val := range v.Value {
						if _, exists := handlerItf[val]; !exists {
							handlerItf[val] = struct{}{}
						}
					}
				}
			}
		}

		// 生成包装结构体
		constructor, wrapperCode := g.generateWrapperStruct(iface, lo.Ternary(middlewareCount > 0, handlerItfName, ""))
		parts = append(parts, wrapperCode)

		// 生成 bind 通用方法
		bindMethodCode := g.generateBindMethod(iface)
		parts = append(parts, bindMethodCode)
		parts = append(parts, "")

		// 为每个方法生成处理器方法
		for _, method := range iface.Methods {
			// 添加注释
			if v, ok := comments[method.Name]; ok {
				parts = append(parts, v)
			}
			handlerCode := g.generateHandlerMethod(iface, method)
			parts = append(parts, handlerCode)
			parts = append(parts, "")
		}

		// 为每个方法生成绑定方法
		for _, method := range iface.Methods {
			methodCode := g.generateMethodBinding(iface, method, middlewareMap[method.Name])
			parts = append(parts, methodCode)
			// 在方法之间添加空行，但不在最后一个方法后添加
			parts = append(parts, "")
		}

		// BindAll 方法
		template := fmt.Sprintf("func (a *%s) BindAll(router gin.IRoutes, preHandlers ...gin.HandlerFunc) {", iface.GetWrapperName())
		parts = append(parts, template)
		for _, method := range iface.Methods {
			parts = append(parts, fmt.Sprintf("	a.%s(router, preHandlers...)", fmt.Sprintf("Bind%s", method.Name)))
		}
		parts = append(parts, "}")
		parts = append(parts, "") // 接口之间空行分隔

		if len(handlerItf) > 0 {
			handlerInterface = append(handlerInterface, fmt.Sprintf("type %s interface {", handlerItfName))
			items := maps.Keys(handlerItf)
			sort.Strings(items)
			for _, key := range items {
				handlerInterface = append(handlerInterface, fmt.Sprintf("%s() gin.HandlerFunc", key))
			}
			handlerInterface = append(handlerInterface, "}")
			handlerInterface = append(handlerInterface, "\n")
		}

		constructorParts = append(constructorParts, constructor)
	}

	return strings.Join(constructorParts, "\n\n"), strings.Join(slices.Concat(handlerInterface, parts), "\n")
}

// generateWrapperStruct 生成包装结构体l
func (g *GinGenerator) generateWrapperStruct(iface SwaggerInterface, handlerItfName string) (string, string) {
	wrapperName := iface.GetWrapperName()
	constructorName := fmt.Sprintf("New%s", wrapperName)

	if len(handlerItfName) == 0 {
		template1 := `
func {{.ConstructorName}}(inner {{.InterfaceName}}) *{{.WrapperName}} {
    return &{{.WrapperName}}{
        inner: inner,
    }
}
`

		data1 := map[string]interface{}{
			"ConstructorName": constructorName,
			"WrapperName":     wrapperName,
			"InterfaceName":   iface.Name,
		}
		constructorResult := utils.MustExecuteTemplate(data1, template1)

		template := `
type {{.WrapperName}} struct {
    inner {{.InterfaceName}}
}
`
		data := map[string]interface{}{
			"WrapperName":   wrapperName,
			"InterfaceName": iface.Name,
		}

		result := utils.MustExecuteTemplate(data, template)
		return strings.TrimSpace(constructorResult), strings.TrimSpace(result)
	}

	template1 := `
func {{.ConstructorName}}(inner {{.InterfaceName}}, handler {{.HandlerName}}) *{{.WrapperName}} {
    return &{{.WrapperName}}{
        inner: inner,
        handler: handler,
    }
}
`

	data1 := map[string]interface{}{
		"ConstructorName": constructorName,
		"WrapperName":     wrapperName,
		"InterfaceName":   iface.Name,
		"HandlerName":     handlerItfName,
	}

	constructorResult := utils.MustExecuteTemplate(data1, template1)

	template := `
type {{.WrapperName}} struct {
    inner {{.InterfaceName}}
    handler {{.HandlerName}}
}
`
	data := map[string]interface{}{
		"WrapperName":   wrapperName,
		"InterfaceName": iface.Name,
		"HandlerName":   handlerItfName,
	}

	result := utils.MustExecuteTemplate(data, template)
	return strings.TrimSpace(constructorResult), strings.TrimSpace(result)
}

// generateBindMethod 生成通用的 bind 方法
func (g *GinGenerator) generateBindMethod(iface SwaggerInterface) string {
	wrapperName := iface.GetWrapperName()

	template := `
func (a *{{.WrapperName}}) bind(router gin.IRoutes, method, path string, preHandlers, innerHandlers []gin.HandlerFunc, f gin.HandlerFunc) {
    var basePath string
    if v, ok := router.(interface {
        BasePath() string
    }); ok {
        basePath = v.BasePath()
    }
    handlers := make([]gin.HandlerFunc, 0, len(preHandlers)+len(innerHandlers)+1)
    handlers = append(handlers, preHandlers...)
    handlers = append(handlers, innerHandlers...)
    handlers = append(handlers, f)
    router.Handle(method, strings.TrimPrefix(path, basePath), handlers...)
}
`

	data := map[string]interface{}{
		"WrapperName": wrapperName,
	}

	result := utils.MustExecuteTemplate(data, template)
	return strings.TrimSpace(result)
}

// generateHandlerMethod 生成处理器方法
func (g *GinGenerator) generateHandlerMethod(iface SwaggerInterface, method SwaggerMethod) string {
	wrapperName := iface.GetWrapperName()
	handlerMethodName := method.Name

	// 生成参数绑定代码
	paramBindingCode := g.generateParameterBinding(method)

	// 生成方法调用代码
	methodCallCode := g.generateMethodCall(method)

	var template string
	if paramBindingCode == "" {
		template = `
func (a *{{.WrapperName}}) {{.HandlerMethodName}}(ctx *gin.Context) {
{{.MethodCall}}
}
`
	} else {
		template = `
func (a *{{.WrapperName}}) {{.HandlerMethodName}}(ctx *gin.Context) {
{{.ParameterBinding}}
{{.MethodCall}}
}
`
	}
	data := map[string]interface{}{
		"WrapperName":       wrapperName,
		"HandlerMethodName": handlerMethodName,
		"ParameterBinding":  paramBindingCode,
		"MethodCall":        methodCallCode,
	}
	return strings.TrimSpace(utils.MustExecuteTemplate(data, template))
}

// generateMethodBinding 生成方法绑定
func (g *GinGenerator) generateMethodBinding(iface SwaggerInterface, method SwaggerMethod, middlewares []*parsers.MiddleWare) string {
	wrapperName := iface.GetWrapperName()
	bindMethodName := fmt.Sprintf("Bind%s", method.Name)
	handlerMethodName := method.Name

	if method.Def.IsRemoved() {
		template := `
func (a *{{.WrapperName}}) {{.BindMethodName}}(router gin.IRoutes, preHandlers ...gin.HandlerFunc) {
}
`
		data := map[string]interface{}{
			"WrapperName":    wrapperName,
			"BindMethodName": bindMethodName,
		}
		return strings.TrimSpace(utils.MustExecuteTemplate(data, template))
	}

	// 转换路径格式：{param} -> :param
	ginPaths := lo.Map(method.GetPaths(), func(item string, index int) string {
		return convertPathToGinFormat(item)
	})

	if len(middlewares) > 0 {
		template := `
func (a *{{.WrapperName}}) {{.BindMethodName}}(router gin.IRoutes, preHandlers ...gin.HandlerFunc) { {{- range .GinPath}}
	var handlers []gin.HandlerFunc
	if a.handler != nil {
		handlers = []gin.HandlerFunc{
			{{range $.Handlers}}a.handler.{{.}}(),
			{{end}}
		}
	}
	a.bind(router, "{{$.HTTPMethod}}", "{{.}}", preHandlers, handlers, a.{{$.HandlerMethodName}}){{end}}
}
`

		data := map[string]interface{}{
			"WrapperName":    wrapperName,
			"BindMethodName": bindMethodName,
			"Handlers": lo.Flatten(lo.Map(middlewares, func(item *parsers.MiddleWare, index int) []string {
				return item.Value
			})),
			"HTTPMethod":        method.GetHTTPMethod(),
			"GinPath":           ginPaths,
			"HandlerMethodName": handlerMethodName,
		}
		return strings.TrimSpace(utils.MustExecuteTemplate(data, template))
	} else {
		template := `
func (a *{{.WrapperName}}) {{.BindMethodName}}(router gin.IRoutes, preHandlers ...gin.HandlerFunc) { {{- range .GinPath}}
	a.bind(router, "{{$.HTTPMethod}}", "{{.}}", preHandlers, nil, a.{{$.HandlerMethodName}}){{end}}
}
`

		data := map[string]interface{}{
			"WrapperName":       wrapperName,
			"BindMethodName":    bindMethodName,
			"HTTPMethod":        method.GetHTTPMethod(),
			"GinPath":           ginPaths,
			"HandlerMethodName": handlerMethodName,
		}
		return strings.TrimSpace(utils.MustExecuteTemplate(data, template))
	}
}

// generateParameterBinding 生成参数绑定代码
func (g *GinGenerator) generateParameterBinding(method SwaggerMethod) string {
	var lines []string

	for i, param := range method.Parameters {
		if param.Type.FullName == "*gin.Context" ||
			param.Type.TypeName == "Context" ||
			strings.Contains(param.Type.FullName, "context.Context") {
			continue
		}
		if param.Source == "path" {
			// 来源于路径的参数
			lines = append(lines, g.generatePathParamBinding(param))
			continue
		} else if param.Source == "header" {
			// 来源于header的参数
			lines = append(lines, g.generateHeaderParamBinding(param))
			continue
		}
		if i == len(method.Parameters)-1 {
			// 默认的
			if method.GetHTTPMethod() == "GET" {
				lines = append(lines, g.generateQueryParamBinding(param))
			} else if v, _ := method.Def.GetAcceptType(); v == "json" {
				lines = append(lines, g.generateBodyParamBinding(param))
			} else {
				lines = append(lines, g.generateFormParamBinding(param))
			}
		}
	}

	// 缩进所有行
	for i, line := range lines {
		if line != "" && !strings.HasPrefix(line, "        ") {
			lines[i] = "        " + line
		}
	}

	return strings.Join(lines, "\n")
}

// generateTypedParamBinding 生成带类型转换的参数绑定
func (g *GinGenerator) generateTypedParamBinding(param Parameter, paramValue string) string {
	typeName := param.Type.TypeName

	// 检查是否需要类型转换
	switch typeName {
	case "int":
		return fmt.Sprintf(`%s := cast.ToInt(%s)`, param.Name, paramValue)
	case "int8":
		return fmt.Sprintf(`%s := cast.ToInt8(%s)`, param.Name, paramValue)
	case "int16":
		return fmt.Sprintf(`%s := cast.ToInt16(%s)`, param.Name, paramValue)
	case "int32":
		return fmt.Sprintf(`%s := cast.ToInt32(%s)`, param.Name, paramValue)
	case "int64":
		return fmt.Sprintf(`%s := cast.ToInt64(%s)`, param.Name, paramValue)
	case "uint":
		return fmt.Sprintf(`%s := cast.ToUint(%s)`, param.Name, paramValue)
	case "uint8":
		return fmt.Sprintf(`%s := cast.ToUint8(%s)`, param.Name, paramValue)
	case "uint16":
		return fmt.Sprintf(`%s := cast.ToUint16(%s)`, param.Name, paramValue)
	case "uint32":
		return fmt.Sprintf(`%s := cast.ToUint32(%s)`, param.Name, paramValue)
	case "uint64":
		return fmt.Sprintf(`%s := cast.ToUint64(%s)`, param.Name, paramValue)
	case "float32":
		return fmt.Sprintf(`%s := cast.ToFloat32(%s)`, param.Name, paramValue)
	case "float64":
		return fmt.Sprintf(`%s := cast.ToFloat64(%s)`, param.Name, paramValue)
	case "bool":
		return fmt.Sprintf(`%s := cast.ToBool(%s)`, param.Name, paramValue)
	case "string":
		// 字符串类型不需要转换
		return fmt.Sprintf(`%s := %s`, param.Name, paramValue)
	default:
		// 对于其他类型，假设是字符串类型
		return fmt.Sprintf(`%s := %s`, param.Name, paramValue)
	}
}

// generatePathParamBinding 生成路径参数绑定
func (g *GinGenerator) generatePathParamBinding(param Parameter) string {
	paramNameInPath := param.Name
	if param.Alias != "" {
		paramNameInPath = param.Alias
	}
	paramValue := fmt.Sprintf(`ctx.Param("%s")`, paramNameInPath)
	return g.generateTypedParamBinding(param, paramValue)
}

// generateQueryParamBinding 生成query参数绑定
func (g *GinGenerator) generateQueryParamBinding(param Parameter) string {
	varName := param.Name
	typeName := param.Type.FullName

	s := fmt.Sprintf(`var %s %s
        if !onGinBind(ctx, &%s, "QUERY") {
			return
		}`, varName, typeName, varName)
	return s
}

// generateFormParamBinding 生成表单参数绑定
func (g *GinGenerator) generateFormParamBinding(param Parameter) string {
	varName := param.Name
	typeName := param.Type.FullName

	s := fmt.Sprintf(`var %s %s
        if !onGinBind(ctx, &%s, "FORM") {
			return
		}`, varName, typeName, varName)
	return s
}

// generateBodyParamBinding 生成 body 参数绑定
func (g *GinGenerator) generateBodyParamBinding(param Parameter) string {
	varName := param.Name
	typeName := param.Type.FullName

	s := fmt.Sprintf(`var %s %s
        if !onGinBind(ctx, &%s, "JSON") {
			return
		}`, varName, typeName, varName)
	return s
}

// generateHeaderParamBinding 生成头部参数绑定
func (g *GinGenerator) generateHeaderParamBinding(param Parameter) string {
	return fmt.Sprintf(`%s := ctx.GetHeader("%s")`, param.Name, param.Name)
}

// generateMethodCall 生成方法调用代码
func (g *GinGenerator) generateMethodCall(method SwaggerMethod) string {
	var args []string

	// 添加 context 参数（如果方法需要）
	needsContext := g.methodNeedsContext(method)
	if needsContext {
		args = append(args, "ctx")
	}

	// 按照接口定义的顺序添加参数
	for _, param := range method.Parameters {
		// 跳过 gin.Context 和 context.Context 参数
		if param.Type.FullName == "*gin.Context" ||
			param.Type.TypeName == "Context" ||
			strings.Contains(param.Type.FullName, "context.Context") {
			continue
		}
		args = append(args, param.Name)
	}

	// 构建方法调用
	methodCall := fmt.Sprintf("a.inner.%s(%s)", method.Name, strings.Join(args, ", "))

	// 生成响应处理
	responseCode := g.generateResponseHandling(method, methodCall)

	return "        " + responseCode
}

// methodNeedsContext 检查方法是否需要 context 参数
func (g *GinGenerator) methodNeedsContext(method SwaggerMethod) bool {
	// 检查原始接口方法是否有 context 参数
	for _, param := range method.Parameters {
		if param.Type.FullName == "*gin.Context" ||
			param.Type.TypeName == "Context" ||
			strings.Contains(param.Type.FullName, "context.Context") {
			return true
		}
	}
	return false
}

// generateResponseHandling 生成响应处理代码
func (g *GinGenerator) generateResponseHandling(method SwaggerMethod, methodCall string) string {
	// 检查返回类型
	if method.ResponseType.FullName == "" {
		// 无返回值
		return fmt.Sprintf(`%s
        onGinResponse(c, gin.H{"status": "success"})`, methodCall)
	}

	// 检查是否是错误类型
	if g.isErrorType(method.ResponseType) {
		return fmt.Sprintf(`if err := %s; err != nil {
            onGinBindErr(ctx, err)
            return
        }
        onGinResponse(c, gin.H{"status": "success"})`, methodCall)
	}

	// 普通返回值 - 使用 result 避免与请求参数 data 冲突
	return fmt.Sprintf(`var result %s = %s
        onGinResponse(ctx, result)`, method.ResponseType.FullName, methodCall)
}

// isErrorType 检查是否是错误类型
func (g *GinGenerator) isErrorType(typeInfo TypeInfo) bool {
	return typeInfo.TypeName == "error" ||
		strings.Contains(typeInfo.FullName, "error") ||
		strings.HasSuffix(typeInfo.TypeName, "Error")
}

// GenerateComplete 生成完整的 Gin 绑定代码
func (g *GinGenerator) GenerateComplete(comments map[string]string) string {
	var parts []string

	// 生成包装结构体和绑定方法
	constructorCode, ginCode := g.GenerateGinCode(comments)
	if constructorCode != "" {
		parts = append(parts, constructorCode)
	}
	if ginCode != "" {
		parts = append(parts, ginCode)
	}

	// 生成辅助函数
	helperFunctions := g.generateHelperFunctions()
	if helperFunctions != "" {
		parts = append(parts, helperFunctions)
	}

	return strings.Join(parts, "\n\n")
}

// generateHelperFunctions 生成辅助函数
func (g *GinGenerator) generateHelperFunctions() string {
	return `
// func onGinBind(c *gin.Context, val any, typ string) bool {
//     if err := c.ShouldBind(&val); err != nil {
// 			c.JSON(400, gin.H{"error": err.Error()})
// 			return false
// 		}
// 		return true
// }
// 
// func onGinResponse[T any](c *gin.Context, data T) {
//     c.JSON(200, data)
// }`
}
