package main

import (
	"fmt"
	"github.com/donutnomad/gotoolkit/internal/utils"
	"github.com/samber/lo"
	"regexp"
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
func (g *GinGenerator) GenerateGinCode(comments map[string]string) string {
	var parts []string

	// 为每个接口生成包装结构体和绑定方法
	for _, iface := range g.collection.Interfaces {
		// 生成包装结构体
		wrapperCode := g.generateWrapperStruct(iface)
		parts = append(parts, wrapperCode)

		// 生成 bind 通用方法
		bindMethodCode := g.generateBindMethod(iface)
		parts = append(parts, bindMethodCode)
		parts = append(parts, "")

		// 为每个方法生成处理器方法
		for i, method := range iface.Methods {
			// 添加注释
			if v, ok := comments[method.Name]; ok {
				parts = append(parts, v)
			}
			handlerCode := g.generateHandlerMethod(iface, method)
			parts = append(parts, handlerCode)
			parts = append(parts, "")
			_ = i
		}

		// 为每个方法生成绑定方法
		for i, method := range iface.Methods {
			methodCode := g.generateMethodBinding(iface, method)
			parts = append(parts, methodCode)
			// 在方法之间添加空行，但不在最后一个方法后添加
			parts = append(parts, "")
			_ = i
		}

		// BindAll 方法
		template := fmt.Sprintf("func (a *%s) BindAll(router gin.IRoutes, preHandlers ...gin.HandlerFunc) {", iface.GetWrapperName())
		parts = append(parts, template)
		for _, method := range iface.Methods {
			parts = append(parts, fmt.Sprintf("	a.%s(router, preHandlers...)", fmt.Sprintf("Bind%s", method.Name)))
		}
		parts = append(parts, "}")

		parts = append(parts, "") // 接口之间空行分隔
	}

	return strings.Join(parts, "\n")
}

// generateWrapperStruct 生成包装结构体
func (g *GinGenerator) generateWrapperStruct(iface SwaggerInterface) string {
	wrapperName := iface.GetWrapperName()

	template := `
type {{.WrapperName}} struct {
    inner {{.InterfaceName}}
}
`

	data := map[string]interface{}{
		"WrapperName":   wrapperName,
		"InterfaceName": iface.Name,
	}

	result, _ := utils.ExecuteTemplate(data, template)
	return strings.TrimSpace(result)
}

// generateBindMethod 生成通用的 bind 方法
func (g *GinGenerator) generateBindMethod(iface SwaggerInterface) string {
	wrapperName := iface.GetWrapperName()

	template := `
func (a *{{.WrapperName}}) bind(router gin.IRoutes, method, path string, preHandlers []gin.HandlerFunc, f gin.HandlerFunc) {
    var basePath string
    if v, ok := router.(interface {
        BasePath() string
    }); ok {
        basePath = v.BasePath()
    }
    handlers := make([]gin.HandlerFunc, 0, len(preHandlers)+1)
    handlers = append(handlers, preHandlers...)
    handlers = append(handlers, f)
    router.Handle(method, strings.TrimPrefix(path, basePath), handlers...)
}
`

	data := map[string]interface{}{
		"WrapperName": wrapperName,
	}

	result, _ := utils.ExecuteTemplate(data, template)
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

	template := `
func (a *{{.WrapperName}}) {{.HandlerMethodName}}(c *gin.Context) {
{{.ParameterBinding}}
{{.MethodCall}}
}
`

	data := map[string]interface{}{
		"WrapperName":       wrapperName,
		"HandlerMethodName": handlerMethodName,
		"ParameterBinding":  paramBindingCode,
		"MethodCall":        methodCallCode,
	}

	result, _ := utils.ExecuteTemplate(data, template)
	return strings.TrimSpace(result)
}

// generateMethodBinding 生成方法绑定
func (g *GinGenerator) generateMethodBinding(iface SwaggerInterface, method SwaggerMethod) string {
	wrapperName := iface.GetWrapperName()
	bindMethodName := fmt.Sprintf("Bind%s", method.Name)
	handlerMethodName := method.Name

	// 转换路径格式：{param} -> :param
	ginPaths := lo.Map(method.Paths, func(item string, index int) string {
		return convertPathToGinFormat(item)
	})

	template := `
func (a *{{.WrapperName}}) {{.BindMethodName}}(router gin.IRoutes, preHandlers ...gin.HandlerFunc) { {{- range .GinPath}}
    a.bind(router, "{{$.HTTPMethod}}", "{{.}}", preHandlers, a.{{$.HandlerMethodName}}){{end}}
}
`

	data := map[string]interface{}{
		"WrapperName":       wrapperName,
		"BindMethodName":    bindMethodName,
		"HTTPMethod":        method.HTTPMethod,
		"GinPath":           ginPaths,
		"HandlerMethodName": handlerMethodName,
	}

	result, _ := utils.ExecuteTemplate(data, template)
	return strings.TrimSpace(result)
}

// generateParameterBinding 生成参数绑定代码
func (g *GinGenerator) generateParameterBinding(method SwaggerMethod) string {
	var lines []string
	var hasError bool

	// 用于跟踪已处理的 body 参数
	hasBodyParam := false
	hasFormParam := false
	hasQueryParam := false

	// 首先处理路径参数和查询参数
	for _, param := range method.Parameters {
		// 跳过 gin.Context 和 context.Context 参数
		if param.Type.FullName == "*gin.Context" ||
			param.Type.TypeName == "Context" ||
			strings.Contains(param.Type.FullName, "context.Context") {
			continue
		}

		if param.Source == "path" || param.Source == "query" || param.Source == "header" {
			switch param.Source {
			case "path":
				lines = append(lines, g.generatePathParamBinding(param))
			case "query":
				//lines = append(lines, g.generateQueryParamBinding(param))
			case "header":
				lines = append(lines, g.generateHeaderParamBinding(param))
			}
		}
	}

	// 然后处理需要绑定的参数（表单和 body）
	for _, param := range method.Parameters {
		// 跳过 gin.Context 和 context.Context 参数
		if param.Type.FullName == "*gin.Context" ||
			param.Type.TypeName == "Context" ||
			strings.Contains(param.Type.FullName, "context.Context") {
			continue
		}

		switch param.Source {
		case "query":
			if !hasQueryParam {
				lines = append(lines, g.generateQueryParamBinding(param))
				hasQueryParam = true
				hasError = true
			}
		case "formData":
			if !hasFormParam {
				lines = append(lines, g.generateFormParamBinding(param))
				hasFormParam = true
				hasError = true
			}
		case "body":
			if !hasBodyParam {
				// 检查是否应该使用JSON绑定
				if method.ContentType == "application/json" {
					lines = append(lines, g.generateBodyParamBinding(param))
				} else {
					lines = append(lines, g.generateFormParamBinding(param))
				}
				hasBodyParam = true
				hasError = true
			}
		}
	}

	// 添加错误处理
	if hasError {
		lines = append(lines, "            onGinBindErr(c, err)")
		lines = append(lines, "            return")
		lines = append(lines, "        }")
	}

	// 缩进所有行
	for i, line := range lines {
		if line != "" && !strings.HasPrefix(line, "        ") {
			lines[i] = "        " + line
		}
	}

	return strings.Join(lines, "\n")
}

// needsCastImport 检查是否需要 cast 导入
func (g *GinGenerator) needsCastImport() bool {
	for _, iface := range g.collection.Interfaces {
		for _, method := range iface.Methods {
			for _, param := range method.Parameters {
				if param.Source == "path" || param.Source == "query" {
					typeName := param.Type.TypeName
					switch typeName {
					case "int", "int8", "int16", "int32", "int64",
						"uint", "uint8", "uint16", "uint32", "uint64",
						"float32", "float64", "bool":
						return true
					}
				}
			}
		}
	}
	return false
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
	paramValue := fmt.Sprintf(`c.Param("%s")`, paramNameInPath)
	return g.generateTypedParamBinding(param, paramValue)
}

// generateQueryParamBinding 生成query参数绑定
func (g *GinGenerator) generateQueryParamBinding(param Parameter) string {
	varName := param.Name
	typeName := param.Type.FullName

	return fmt.Sprintf(`var %s %s
        if err := c.ShouldBindQuery(&%s); err != nil {`, varName, typeName, varName)
}

// generateFormParamBinding 生成表单参数绑定
func (g *GinGenerator) generateFormParamBinding(param Parameter) string {
	varName := param.Name
	typeName := param.Type.FullName

	return fmt.Sprintf(`var %s %s
        if err := c.ShouldBind(&%s); err != nil {`, varName, typeName, varName)
}

// generateBodyParamBinding 生成 body 参数绑定
func (g *GinGenerator) generateBodyParamBinding(param Parameter) string {
	varName := param.Name
	typeName := param.Type.FullName

	return fmt.Sprintf(`var %s %s
        if err := c.ShouldBindJSON(&%s); err != nil {`, varName, typeName, varName)
}

// generateHeaderParamBinding 生成头部参数绑定
func (g *GinGenerator) generateHeaderParamBinding(param Parameter) string {
	return fmt.Sprintf(`%s := c.GetHeader("%s")`, param.Name, param.Name)
}

// generateMethodCall 生成方法调用代码
func (g *GinGenerator) generateMethodCall(method SwaggerMethod) string {
	var args []string

	// 添加 context 参数（如果方法需要）
	needsContext := g.methodNeedsContext(method)
	if needsContext {
		args = append(args, "c")
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
            onGinBindErr(c, err)
            return
        }
        onGinResponse(c, gin.H{"status": "success"})`, methodCall)
	}

	// 普通返回值 - 使用 result 避免与请求参数 data 冲突
	return fmt.Sprintf(`var result %s = %s
        onGinResponse(c, result)`, method.ResponseType.FullName, methodCall)
}

// isErrorType 检查是否是错误类型
func (g *GinGenerator) isErrorType(typeInfo TypeInfo) bool {
	return typeInfo.TypeName == "error" ||
		strings.Contains(typeInfo.FullName, "error") ||
		strings.HasSuffix(typeInfo.TypeName, "Error")
}

// GenerateConstructors 生成构造函数
func (g *GinGenerator) GenerateConstructors() string {
	var parts []string

	for _, iface := range g.collection.Interfaces {
		constructor := g.generateConstructor(iface)
		parts = append(parts, constructor)
	}

	return strings.Join(parts, "\n\n")
}

// generateConstructor 生成构造函数
func (g *GinGenerator) generateConstructor(iface SwaggerInterface) string {
	wrapperName := iface.GetWrapperName()
	constructorName := fmt.Sprintf("New%s", wrapperName)

	template := `
func {{.ConstructorName}}(inner {{.InterfaceName}}) *{{.WrapperName}} {
    return &{{.WrapperName}}{
        inner: inner,
    }
}
`

	data := map[string]interface{}{
		"ConstructorName": constructorName,
		"WrapperName":     wrapperName,
		"InterfaceName":   iface.Name,
	}

	result, _ := utils.ExecuteTemplate(data, template)
	return strings.TrimSpace(result)
}

// GenerateComplete 生成完整的 Gin 绑定代码
func (g *GinGenerator) GenerateComplete(comments map[string]string) string {
	var parts []string

	// 生成辅助函数
	helperFunctions := g.generateHelperFunctions()
	if helperFunctions != "" {
		parts = append(parts, helperFunctions)
	}

	// 生成构造函数
	constructors := g.GenerateConstructors()
	if constructors != "" {
		parts = append(parts, constructors)
	}

	// 生成包装结构体和绑定方法
	ginCode := g.GenerateGinCode(comments)
	if ginCode != "" {
		parts = append(parts, ginCode)
	}

	return strings.Join(parts, "\n\n")
}

// generateHelperFunctions 生成辅助函数
func (g *GinGenerator) generateHelperFunctions() string {
	return `
// func onGinBindErr(c *gin.Context, err error) {
//     c.JSON(400, gin.H{"error": err.Error()})
// }
// 
// func onGinResponse[T any](c *gin.Context, data T) {
//     c.JSON(200, data)
// }`
}
