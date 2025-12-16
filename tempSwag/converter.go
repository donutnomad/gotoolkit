package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi2conv"
	"github.com/getkin/kin-openapi/openapi3"
)

// APIItemResponse API 响应项
type APIItemResponse struct {
	Name             string      `json:"name"`
	Description      string      `json:"description"`
	Tags             []string    `json:"tags"`
	Path             string      `json:"path"`
	Method           string      `json:"method"`
	Parameters       Parameters  `json:"parameters"`
	CommonParameters struct{}    `json:"commonParameters"`
	Auth             struct{}    `json:"auth"`
	Responses        []Response  `json:"responses"`
	ResponseExamples []any       `json:"responseExamples"`
	RequestBody      RequestBody `json:"requestBody"`
	Cases            []Case      `json:"cases"`
	CustomAPIFields  string      `json:"customApiFields"`
	CodeSamples      []any       `json:"codeSamples"`
	OasExtensions    string      `json:"oasExtensions"`
	SecurityScheme   struct{}    `json:"securityScheme"`
	Callbacks        string      `json:"callbacks"`
}

// ChildrenItem 子项结构
type ChildrenItem struct {
	Name     string            `json:"name"`
	Children []any             `json:"children"`
	Items    []APIItemResponse `json:"items"`
}

// HttpCollection HTTP API集合
type HttpCollection struct {
	Name     string         `json:"name"`
	ModuleId string         `json:"moduleId"`
	Children []ChildrenItem `json:"children"`
	Items    []any          `json:"items"`
	Auth     struct{}       `json:"auth"`
}

// JsonScheme JSON Schema定义
//
//	type JsonScheme struct {
//		Type       string                       `json:"type"`
//		Required   []string                     `json:"required,omitempty"`
//		Properties map[string]*openapi3.Schema  `json:"properties"`
//	}
type JsonScheme = openapi3.Schema

// SchemaItem Schema项
type SchemaItem struct {
	Id          string      `json:"id"`
	Name        string      `json:"name"`
	JsonSchema  *JsonScheme `json:"jsonSchema"`
	Description string      `json:"description"`
}

// SchemeCollection Schema集合
type SchemeCollection[T any] struct {
	Name     string `json:"name"`
	Children []any  `json:"children"`
	Items    []T    `json:"items"`
	ModuleId string `json:"moduleId"`
	TempId   int    `json:"__tempId,omitempty"`
}

// Collection 通用集合
type Collection struct {
	Name     string `json:"name"`
	Children []any  `json:"children"`
	Items    []any  `json:"items"`
}

// ModuleSetting 模块设置
type ModuleSetting struct {
	Id          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	ApiFolderId int         `json:"apiFolderId,omitempty"`
	ModuleId    int         `json:"moduleId"`
	OpenApiInfo OpenAPIInfo `json:"openApiInfo"`
	ImportTo    string      `json:"importTo,omitempty"`
}

// OpenAPIConverter OpenAPI转换器
type OpenAPIConverter struct {
	data                   []byte
	format                 string
	projectLanguage        string
	projectCustomAPIFields []any
	isExistedProject       bool
	apiFolderId            string
	moduleId               string
	apiNameExtractMethod   string

	// 转换结果
	refMap                         map[string]string
	combinationSecuritySchemeNames map[string]bool
	dataSchemaFolders              []SchemeCollection[SchemaItem]
	securitySchemeFolders          []Collection
	httpFolders                    []HttpCollection
	environments                   []Environment
	basePath                       string
	title                          string
	extra                          Extra
	contentVersion                 string
}

// Extra 额外信息
type Extra struct {
	ModuleSettings []ModuleSetting `json:"moduleSettings"`
}

// Environment 环境配置
type Environment struct {
	BaseURLs   map[string]string `json:"baseUrls"`
	Name       string            `json:"name"`
	Variables  []any             `json:"variables"`
	Parameters map[string]any    `json:"parameters"`
}

// NewOpenAPIConverter 创建转换器实例
func NewOpenAPIConverter(data []byte, format string) *OpenAPIConverter {
	return &OpenAPIConverter{
		data:                           data,
		format:                         format,
		refMap:                         make(map[string]string),
		combinationSecuritySchemeNames: make(map[string]bool),
		projectCustomAPIFields:         []any{},
	}
}

// Convert 执行转换
func (c *OpenAPIConverter) Convert() error {
	// 获取自定义API字段
	if err := c.getCustomAPIFields(); err != nil {
		return err
	}

	// 获取项目设置
	if err := c.getProjectSettingData(); err != nil {
		return err
	}

	// 检测文档类型
	var rawDoc map[string]any
	if err := json.Unmarshal(c.data, &rawDoc); err != nil {
		return fmt.Errorf("解析JSON失败: %w", err)
	}

	// 解析OpenAPI文档
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	var doc *openapi3.T
	var err error

	// 检查是否是 Swagger 2.0
	if swagger, ok := rawDoc["swagger"].(string); ok && strings.HasPrefix(swagger, "2.") {
		// Swagger 2.0 格式，需要转换
		log.Println("检测到 Swagger 2.0 格式，正在转换为 OpenAPI 3.0...")
		doc, err = convertSwagger2ToOpenAPI3(c.data)
		if err != nil {
			return fmt.Errorf("Swagger 2.0 转换失败: %w", err)
		}
	} else {
		// OpenAPI 3.0+ 格式
		doc, err = loader.LoadFromData(c.data)
		if err != nil {
			return fmt.Errorf("解析OpenAPI文档失败: %w", err)
		}
	}

	// 验证文档
	if err := doc.Validate(loader.Context); err != nil {
		log.Printf("警告: OpenAPI文档验证出现问题: %v", err)
		// 不返回错误，继续处理
	}

	c.contentVersion = doc.OpenAPI

	// 生成模块ID
	moduleId := generateID()

	// 转换数据模式
	c.dataSchemaFolders = c.convertDataSchema(doc)
	for i := range c.dataSchemaFolders {
		c.dataSchemaFolders[i].ModuleId = moduleId
	}

	// 转换安全方案
	c.securitySchemeFolders = c.convertSecurityScheme(doc)

	// 转换HTTP API
	c.httpFolders = c.convertHttpAPI(doc)
	for i := range c.httpFolders {
		c.httpFolders[i].ModuleId = moduleId
	}

	// 转换环境
	c.environments = c.convertEnvironment(doc)

	// 设置额外信息
	c.extra = Extra{
		ModuleSettings: []ModuleSetting{
			{
				Id:          moduleId,
				Name:        getInfoTitle(doc),
				Description: getInfoDescription(doc),
				ApiFolderId: 0,
				ModuleId:    0,
				OpenApiInfo: OpenAPIInfo{
					Title:   doc.Info.Title,
					Version: doc.Info.Version,
					Contact: struct{}{},
				},
			},
		},
	}

	// 设置basePath
	if basePath, ok := doc.Extensions["x-basePath"].(string); ok {
		c.basePath = basePath
	}

	// 设置标题
	if len(c.extra.ModuleSettings) == 1 {
		c.title = c.extra.ModuleSettings[0].Name
	}

	return nil
}

// convertSwagger2ToOpenAPI3 将 Swagger 2.0 转换为 OpenAPI 3.0
func convertSwagger2ToOpenAPI3(data []byte) (*openapi3.T, error) {
	// 解析 Swagger 2.0 文档
	var swagger2Doc openapi2.T
	if err := json.Unmarshal(data, &swagger2Doc); err != nil {
		return nil, fmt.Errorf("解析 Swagger 2.0 文档失败: %w", err)
	}

	// 转换为 OpenAPI 3.0
	doc, err := openapi2conv.ToV3(&swagger2Doc)
	if err != nil {
		return nil, fmt.Errorf("转换为 OpenAPI 3.0 失败: %w", err)
	}

	return doc, nil
}

// convertHttpAPI 转换HTTP API
func (c *OpenAPIConverter) convertHttpAPI(doc *openapi3.T) []HttpCollection {
	folders := []FolderItem{}
	componentParameters := make(map[string]*openapi3.ParameterRef)

	// 处理组件参数
	if doc.Components != nil && doc.Components.Parameters != nil {
		for name, param := range doc.Components.Parameters {
			componentParameters[fmt.Sprintf("#/components/parameters/%s", name)] = param
		}
	}

	// 获取可用的安全方案
	availableSecuritySchemes := []string{}
	if doc.Components != nil && doc.Components.SecuritySchemes != nil {
		for name := range doc.Components.SecuritySchemes {
			availableSecuritySchemes = append(availableSecuritySchemes, name)
		}
	}

	// 创建默认文件夹
	defaultFolder := &FolderItem{
		Name:     "默认文件夹",
		Children: []*FolderItem{},
		Items:    []APIItem{},
	}

	// 处理路径
	processPathItem := func(paths map[string]*openapi3.PathItem, isWebhook bool) {
		for path, pathItem := range paths {
			if pathItem == nil {
				continue
			}

			operations := map[string]*openapi3.Operation{
				"get":     pathItem.Get,
				"post":    pathItem.Post,
				"put":     pathItem.Put,
				"delete":  pathItem.Delete,
				"patch":   pathItem.Patch,
				"options": pathItem.Options,
				"head":    pathItem.Head,
				"trace":   pathItem.Trace,
			}

			for method, operation := range operations {
				if operation == nil {
					continue
				}

				// 获取文件夹名称
				folderName := c.getFolderName(operation)

				// 创建API项
				apiItem := c.convertAPIItem(path, method, pathItem, operation, doc, componentParameters, availableSecuritySchemes, isWebhook)

				if folderName != "" {
					// 创建或查找文件夹
					c.ensureFolderPath(&folders, folderName)
					folder := findFolder(folders, folderName)
					if folder != nil {
						folder.Items = append(folder.Items, apiItem)
					}
				} else {
					defaultFolder.Items = append(defaultFolder.Items, apiItem)
				}
			}
		}
	}

	// 处理paths
	if doc.Paths != nil {
		pathsMap := make(map[string]*openapi3.PathItem)
		for path, pathItem := range doc.Paths.Map() {
			pathsMap[path] = pathItem
		}
		processPathItem(pathsMap, false)
	}

	// 处理webhooks (OpenAPI 3.1)
	if webhooks, ok := doc.Extensions["webhooks"].(map[string]any); ok {
		webhookPaths := make(map[string]*openapi3.PathItem)
		for name, webhook := range webhooks {
			if webhookData, err := json.Marshal(webhook); err == nil {
				var pathItem openapi3.PathItem
				if json.Unmarshal(webhookData, &pathItem) == nil {
					webhookPaths[name] = &pathItem
				}
			}
		}
		processPathItem(webhookPaths, true)
	}

	// 构建树形结构
	tree := c.buildTreeList(folders)
	defaultFolder.Children = tree

	return []HttpCollection{
		{
			Name:     defaultFolder.Name,
			ModuleId: "",
			Children: convertFolderToChildren(defaultFolder.Children),
			Items:    []any{},
			Auth:     struct{}{},
		},
	}
}

// convertAPIItem 转换单个API项
func (c *OpenAPIConverter) convertAPIItem(
	path, method string,
	pathItem *openapi3.PathItem,
	operation *openapi3.Operation,
	_ *openapi3.T,
	_ map[string]*openapi3.ParameterRef,
	_ []string,
	_ bool,
) APIItem {
	item := APIItem{
		Name:        operation.Summary,
		Description: operation.Description,
		Tags:        operation.Tags,
		Path:        path,
		Method:      method,
		Parameters: Parameters{
			Path:   []CommonParameter{},
			Query:  []CommonParameter{},
			Cookie: []CommonParameter{},
			Header: []CommonParameter{},
		},
		CommonParameters: struct{}{},
		Auth:             struct{}{},
		Responses:        []Response{},
		ResponseExamples: []any{},
		//RequestBody:      RequestBody{},
		RequestBody:     &openapi3.RequestBodyRef{},
		Cases:           []Case{},
		CustomAPIFields: "{}",
		CodeSamples:     []any{},
		OasExtensions:   "{}",
		SecurityScheme:  struct{}{},
		Callbacks:       "{}",
	}

	// 处理参数
	allParameters := append([]*openapi3.ParameterRef{}, pathItem.Parameters...)
	allParameters = append(allParameters, operation.Parameters...)

	for _, paramRef := range allParameters {
		param := paramRef.Value
		if param == nil {
			continue
		}
		p := NewCommonParameter(param)
		switch param.In {
		case openapi3.ParameterInPath:
			item.Parameters.Path = append(item.Parameters.Path, p)
		case openapi3.ParameterInQuery:
			item.Parameters.Query = append(item.Parameters.Query, p)
		case openapi3.ParameterInHeader:
			item.Parameters.Header = append(item.Parameters.Header, p)
		case openapi3.ParameterInCookie:
			item.Parameters.Cookie = append(item.Parameters.Cookie, p)
		}
	}

	// 处理响应
	for statusCode, responseRef := range operation.Responses.Map() {
		if responseRef.Value == nil {
			continue
		}

		description := ""
		if responseRef.Value.Description != nil {
			description = *responseRef.Value.Description
		}
		response := Response{
			Id:            generateID(),
			Name:          statusCode,
			Code:          parseStatusCode(statusCode),
			Headers:       []any{},
			Description:   description,
			OasExtensions: "{}",
		}

		// 处理响应内容
		if responseRef.Value.Content != nil {
			for contentType, mediaType := range responseRef.Value.Content {
				response.ContentType = contentType
				response.MediaType = contentType
				if mediaType.Schema != nil {
					if mediaType.Schema.Ref != "" {
						response.JsonSchema = JsonSchemaRef{
							Ref: mediaType.Schema.Ref,
						}
					} else {
						response.JsonSchema = JsonSchemaRef{
							Type: getSchemaType(mediaType.Schema),
						}
					}
				}
				break // 只取第一个
			}
		}

		item.Responses = append(item.Responses, response)
	}

	// 处理请求体
	if operation.RequestBody != nil && operation.RequestBody.Value != nil {
		requestBody := RequestBody{
			Type:                   "json",
			Parameters:             []any{},
			Required:               operation.RequestBody.Value.Required,
			AdditionalContentTypes: []any{},
		}

		if operation.RequestBody.Value.Content != nil {
			for contentType, mediaType := range operation.RequestBody.Value.Content {
				requestBody.MediaType = contentType
				if mediaType.Schema != nil {
					if mediaType.Schema.Ref != "" {
						requestBody.JsonSchema = &JsonSchemaRef{
							Ref: mediaType.Schema.Ref,
						}
					}
				}
				requestBody.OasExtensions = "{}"
				break // 只取第一个
			}
		}

		item.RequestBody = operation.RequestBody
	}

	return item
}

// convertDataSchema 转换数据模式
func (c *OpenAPIConverter) convertDataSchema(doc *openapi3.T) []SchemeCollection[SchemaItem] {
	if doc.Components == nil {
		return []SchemeCollection[SchemaItem]{}
	}

	folders := []SchemaFolder{}

	// 处理schemas
	if doc.Components.Schemas != nil {
		for name, schemaRef := range doc.Components.Schemas {
			if schemaRef.Value == nil {
				continue
			}

			schemaItem := c.createSchemaItem(name, schemaRef.Value, "schemas", "Schemas")
			c.addSchemaToFolder(&folders, schemaItem, "Schemas")
		}
	}

	// 处理responses
	if doc.Components.Responses != nil {
		for name, responseRef := range doc.Components.Responses {
			if responseRef.Value == nil || responseRef.Value.Content == nil {
				continue
			}

			for _, mediaType := range responseRef.Value.Content {
				if mediaType.Schema != nil && mediaType.Schema.Value != nil {
					schemaItem := c.createSchemaItem(name, mediaType.Schema.Value, "responses", "Response")
					c.addSchemaToFolder(&folders, schemaItem, "Response")
				}
				break
			}
		}
	}

	// 处理requestBodies
	if doc.Components.RequestBodies != nil {
		for name, requestBodyRef := range doc.Components.RequestBodies {
			if requestBodyRef.Value == nil || requestBodyRef.Value.Content == nil {
				continue
			}

			for _, mediaType := range requestBodyRef.Value.Content {
				if mediaType.Schema != nil {
					if mediaType.Schema.Ref != "" {
						c.refMap[fmt.Sprintf("#/components/requestBodies/%s", name)] = mediaType.Schema.Ref
						continue
					}
					if mediaType.Schema.Value != nil {
						schemaItem := c.createSchemaItem(name, mediaType.Schema.Value, "requestBodies", "RequestBodies")
						c.addSchemaToFolder(&folders, schemaItem, "RequestBodies")
					}
				}
				break
			}
		}
	}

	// 解析引用映射
	c.resolveRefMap()

	// 转换为树形结构
	result := []SchemeCollection[SchemaItem]{}
	for _, folder := range folders {
		result = append(result, SchemeCollection[SchemaItem]{
			Name:     folder.Name,
			Children: []any{},
			Items:    folder.Items,
			ModuleId: "",
			TempId:   0,
		})
	}

	return result
}

// convertSecurityScheme 转换安全方案
func (c *OpenAPIConverter) convertSecurityScheme(doc *openapi3.T) []Collection {
	rootFolder := Collection{
		Name:     "安全方案",
		Children: []any{},
		Items:    []any{},
	}

	if doc.Components == nil || doc.Components.SecuritySchemes == nil {
		return []Collection{rootFolder}
	}

	for name, schemeRef := range doc.Components.SecuritySchemes {
		if schemeRef.Value == nil {
			continue
		}

		scheme := schemeRef.Value
		securityItem := map[string]any{
			"name":        name,
			"type":        "SecurityScheme",
			"authType":    c.formatAuthType(scheme),
			"oasAuthType": scheme.Type,
			"authConfigs": scheme,
		}

		rootFolder.Items = append(rootFolder.Items, securityItem)
	}

	return []Collection{rootFolder}
}

// convertEnvironment 转换环境
func (c *OpenAPIConverter) convertEnvironment(doc *openapi3.T) []Environment {
	environments := []Environment{}

	if doc.Servers != nil {
		for _, server := range doc.Servers {
			if server.URL == "" {
				continue
			}

			env := Environment{
				BaseURLs: map[string]string{
					"default": server.URL,
				},
				Name:       server.Description,
				Variables:  []any{},
				Parameters: make(map[string]any),
			}

			if env.Name == "" {
				env.Name = server.URL
			}

			environments = append(environments, env)
		}
	}

	return environments
}

// 辅助方法

func (c *OpenAPIConverter) getCustomAPIFields() error {
	if len(c.projectCustomAPIFields) == 0 {
		if c.isExistedProject {
			// TODO: 从API获取自定义字段
			c.projectCustomAPIFields = []any{}
		} else {
			c.projectCustomAPIFields = []any{}
		}
	}
	return nil
}

func (c *OpenAPIConverter) getProjectSettingData() error {
	if c.projectLanguage == "" {
		if c.isExistedProject {
			// TODO: 从API获取项目语言
			c.projectLanguage = "zh-CN"
		} else {
			c.projectLanguage = "zh-CN"
		}
	}
	return nil
}

func (c *OpenAPIConverter) getFolderName(operation *openapi3.Operation) string {
	// 优先使用tags
	if len(operation.Tags) > 0 {
		parts := strings.Split(operation.Tags[0], "/")
		cleanParts := []string{}
		for _, part := range parts {
			cleanParts = append(cleanParts, strings.TrimSpace(part))
		}
		return strings.Join(cleanParts, "/")
	}
	return ""
}

func (c *OpenAPIConverter) createSchemaItem(name string, schema *openapi3.Schema, componentType, _ string) SchemaItem {
	//properties := make(map[string]*openapi3.Schema)
	//if schema.Properties != nil {
	//	for propName, propRef := range schema.Properties {
	//		if propRef.Value != nil {
	//			// 直接使用 openapi3.Schema 对象
	//			properties[propName] = propRef.Value
	//		}
	//	}
	//}
	return SchemaItem{
		Id:          fmt.Sprintf("#/components/%s/%s", componentType, name),
		Name:        name,
		JsonSchema:  schema,
		Description: schema.Description,
	}
	//	JsonSchema: JsonScheme{
	//		Type:       getSchemaTypeFromSchema(schema),
	//		Properties: properties,
	//		Required:   schema.Required,
	//	},
	//	Description: schema.Description,
	//}
}

func (c *OpenAPIConverter) addSchemaToFolder(folders *[]SchemaFolder, item SchemaItem, folderName string) {
	// 查找或创建文件夹
	var folder *SchemaFolder
	for i := range *folders {
		if (*folders)[i].Name == folderName {
			folder = &(*folders)[i]
			break
		}
	}

	if folder == nil {
		*folders = append(*folders, SchemaFolder{
			Name:  folderName,
			Items: []SchemaItem{},
		})
		folder = &(*folders)[len(*folders)-1]
	}

	folder.Items = append(folder.Items, item)
}

func (c *OpenAPIConverter) resolveRefMap() {
	// 递归解析引用
	var resolve func(string, map[string]bool) string
	resolve = func(key string, visited map[string]bool) string {
		if visited[key] {
			return ""
		}
		visited[key] = true

		if ref, ok := c.refMap[key]; ok {
			if nextRef, ok := c.refMap[ref]; ok {
				return resolve(nextRef, visited)
			}
			return ref
		}
		return ""
	}

	for key := range c.refMap {
		visited := make(map[string]bool)
		if resolved := resolve(key, visited); resolved != "" {
			c.refMap[key] = resolved
		}
	}
}

func (c *OpenAPIConverter) formatAuthType(scheme *openapi3.SecurityScheme) string {
	switch scheme.Type {
	case "http":
		return "bearer"
	case "apiKey":
		return "apikey"
	case "oauth2":
		return "oauth2"
	case "openIdConnect":
		return "openid"
	default:
		return scheme.Type
	}
}

func (c *OpenAPIConverter) buildTreeList(folders []FolderItem) []*FolderItem {
	// 设置父子关系
	for i := range folders {
		parts := strings.Split(folders[i].Name, "/")
		if len(parts) > 1 {
			parentParts := parts[:len(parts)-1]
			folders[i].ParentName = strings.Join(parentParts, "/")
			folders[i].Name = parts[len(parts)-1]
		}
	}

	// 构建树
	root := []*FolderItem{}
	for i := range folders {
		if folders[i].ParentName == "" {
			root = append(root, &folders[i])
		} else {
			// 查找父文件夹
			for j := range folders {
				if folders[j].Name == folders[i].ParentName {
					folders[j].Children = append(folders[j].Children, &folders[i])
					break
				}
			}
		}
	}

	return root
}

func (c *OpenAPIConverter) ensureFolderPath(folders *[]FolderItem, path string) {
	parts := strings.Split(path, "/")
	currentPath := ""

	for i, part := range parts {
		if i > 0 {
			currentPath += "/"
		}
		currentPath += part

		found := false
		for j := range *folders {
			if (*folders)[j].Name == currentPath {
				found = true
				break
			}
		}

		if !found {
			*folders = append(*folders, FolderItem{
				Name:     currentPath,
				Children: []*FolderItem{},
				Items:    []APIItem{},
			})
		}
	}
}

// 辅助函数

func generateID() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const idLength = 10

	b := make([]byte, idLength)
	if _, err := rand.Read(b); err != nil {
		// 如果随机数生成失败，使用备用方案
		return "DefaultIDx"
	}

	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}

	return string(b)
}

func getInfoTitle(doc *openapi3.T) string {
	if doc.Info != nil {
		return doc.Info.Title
	}
	return ""
}

func getInfoDescription(doc *openapi3.T) string {
	if doc.Info != nil {
		return doc.Info.Description
	}
	return ""
}

func getSchemaType(schemaRef *openapi3.SchemaRef) string {
	if schemaRef == nil || schemaRef.Value == nil {
		return "string"
	}
	typeSlice := schemaRef.Value.Type.Slice()
	if len(typeSlice) > 0 {
		return typeSlice[0]
	}
	return "string"
}

func getSchemaTypeFromSchema(schema *openapi3.Schema) string {
	if schema == nil {
		return "string"
	}
	typeSlice := schema.Type.Slice()
	if len(typeSlice) > 0 {
		return typeSlice[0]
	}
	return "string"
}

func parseStatusCode(code string) int {
	var result int
	_, _ = fmt.Sscanf(code, "%d", &result)
	return result
}

func findFolder(folders []FolderItem, name string) *FolderItem {
	for i := range folders {
		if folders[i].Name == name {
			return &folders[i]
		}
	}
	return nil
}

func convertFolderToChildren(folders []*FolderItem) []ChildrenItem {
	result := []ChildrenItem{}

	for _, folder := range folders {
		child := ChildrenItem{
			Name:     folder.Name,
			Children: []any{},
			Items:    []APIItemResponse{},
		}

		for _, item := range folder.Items {
			child.Items = append(child.Items, APIItemResponse{
				Name:             item.Name,
				Description:      item.Description,
				Tags:             item.Tags,
				Path:             item.Path,
				Method:           item.Method,
				Parameters:       item.Parameters,
				CommonParameters: item.CommonParameters,
				Auth:             item.Auth,
				Responses:        item.Responses,
				ResponseExamples: item.ResponseExamples,
				RequestBody:      item.RequestBody,
				Cases:            item.Cases,
				CustomAPIFields:  item.CustomAPIFields,
				CodeSamples:      item.CodeSamples,
				OasExtensions:    item.OasExtensions,
				SecurityScheme:   item.SecurityScheme,
				Callbacks:        item.Callbacks,
			})
		}
		// 排序Items
		sort.Slice(child.Items, func(i, j int) bool {
			return child.Items[i].Path < child.Items[j].Path
		})

		result = append(result, child)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name > result[j].Name
	})

	return result
}

// 内部数据结构

type FolderItem struct {
	Name       string
	ParentName string
	Children   []*FolderItem
	Items      []APIItem
}

type APIItem struct {
	Name             string
	Description      string
	Tags             []string
	Path             string
	Method           string
	Parameters       Parameters
	CommonParameters struct{}
	Auth             struct{}
	Responses        []Response
	ResponseExamples []any
	RequestBody      RequestBody
	Cases            []Case
	CustomAPIFields  string
	CodeSamples      []any
	OasExtensions    string
	SecurityScheme   struct{}
	Callbacks        string
}

type Parameters struct {
	Path   []CommonParameter `json:"path"`
	Query  []CommonParameter `json:"query"`
	Cookie []CommonParameter `json:"cookie"`
	Header []CommonParameter `json:"header"`
}

type CommonParameter struct {
	Id     string `json:"id"`
	Enable bool   `json:"enable"`
	*openapi3.Parameter
}

func NewCommonParameter(param *openapi3.Parameter) CommonParameter {
	return CommonParameter{
		Id:        generateID(),
		Enable:    true,
		Parameter: param,
	}
}

type PathParameter struct {
	Id     string `json:"id"`
	Enable bool   `json:"enable"`
	*openapi3.Parameter

	//Name        string     `json:"name"`
	//Required    bool       `json:"required"`
	//Description string     `json:"description"`
	//Example     any        `json:"example"`
	//Type        string     `json:"type"`
	//Schema      SchemaInfo `json:"schema"`
}

type QueryParameter struct {
	Id     string `json:"id"`
	Enable bool   `json:"enable"`
	*openapi3.Parameter

	//Name        string      `json:"name"`
	//Required    bool        `json:"required"`
	//Description string      `json:"description"`
	//Type        string      `json:"type"`
	//Schema      QuerySchema `json:"schema"`
}

type SchemaInfo struct {
	Type string `json:"type"`
}

type QuerySchema struct {
	Type    string `json:"type"`
	Default any    `json:"default,omitempty"`
	Enum    []any  `json:"enum,omitempty"`
}

type Response struct {
	Id            string        `json:"id"`
	Name          string        `json:"name"`
	Code          int           `json:"code"`
	ContentType   string        `json:"contentType"`
	MediaType     string        `json:"mediaType"`
	JsonSchema    JsonSchemaRef `json:"jsonSchema"`
	Headers       []any         `json:"headers"`
	Description   string        `json:"description"`
	OasExtensions string        `json:"oasExtensions"`
}

type JsonSchemaRef struct {
	Type string `json:"type,omitempty"`
	Ref  string `json:"$ref,omitempty"`
}

type RequestBody struct {
	Type                   string         `json:"type"`
	Parameters             []any          `json:"parameters"`
	JsonSchema             *JsonSchemaRef `json:"jsonSchema,omitempty"`
	MediaType              string         `json:"mediaType,omitempty"`
	OasExtensions          string         `json:"oasExtensions,omitempty"`
	Required               bool           `json:"required"`
	AdditionalContentTypes []any          `json:"additionalContentTypes"`
}

type Case struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Parameters  CaseParameters  `json:"parameters"`
	RequestBody CaseRequestBody `json:"requestBody"`
	ResponseId  string          `json:"responseId"`
	TempId      int             `json:"__tempId"`
}

type CaseParameters struct {
	Path   []CaseParameter `json:"path"`
	Query  []CaseParameter `json:"query"`
	Cookie []any           `json:"cookie"`
	Header []any           `json:"header"`
}

type CaseParameter struct {
	Name   string `json:"name"`
	Value  string `json:"value,omitempty"`
	Enable bool   `json:"enable"`
}

type CaseRequestBody struct {
	Parameters   []any  `json:"parameters"`
	Data         string `json:"data,omitempty"`
	Type         string `json:"type"`
	GenerateMode string `json:"generateMode"`
}

type SchemaFolder struct {
	Name  string
	Items []SchemaItem
}

type OpenAPIInfo struct {
	Contact struct{} `json:"contact"`
	Title   string   `json:"title"`
	Version string   `json:"version"`
}

// ConvertResult 转换结果
type ConvertResult struct {
	HttpCollections []HttpCollection               `json:"httpCollections"`
	DataSchemas     []SchemeCollection[SchemaItem] `json:"dataSchemas"`
	SecuritySchemes []Collection                   `json:"securitySchemes"`
	Environments    []Environment                  `json:"environments"`
	Extra           Extra                          `json:"extra"`
	BasePath        string                         `json:"basePath,omitempty"`
	Title           string                         `json:"title"`
	ContentVersion  string                         `json:"contentVersion"`
}

// GetResult 获取转换结果
func (c *OpenAPIConverter) GetResult() *ConvertResult {
	return &ConvertResult{
		HttpCollections: c.httpFolders,
		DataSchemas:     c.dataSchemaFolders,
		SecuritySchemes: c.securitySchemeFolders,
		Environments:    c.environments,
		Extra:           c.extra,
		BasePath:        c.basePath,
		Title:           c.title,
		ContentVersion:  c.contentVersion,
	}
}

func main() {
	// 定义命令行参数
	inputFile := flag.String("input", "", "输入的OpenAPI/Swagger JSON文件路径 (必需)")
	outputFile := flag.String("output", "", "输出JSON文件路径 (可选,默认输出到标准输出)")
	prettyPrint := flag.Bool("pretty", true, "是否格式化输出JSON (默认: true)")
	helpFlag := flag.Bool("help", false, "显示帮助信息")

	flag.Parse()

	// 显示帮助信息
	if *helpFlag || *inputFile == "" {
		fmt.Println("OpenAPI转换器 - 将OpenAPI/Swagger JSON转换为自定义格式")
		fmt.Println("\n使用方法:")
		fmt.Println("  go run g.go -input <swagger.json> [-output <output.json>] [-pretty]")
		fmt.Println("\n参数说明:")
		flag.PrintDefaults()
		fmt.Println("\n示例:")
		fmt.Println("  go run g.go -input swagger.json")
		fmt.Println("  go run g.go -input swagger.json -output result.json")
		fmt.Println("  go run g.go -input swagger.json -output result.json -pretty=false")

		if *inputFile == "" && !*helpFlag {
			os.Exit(1)
		}
		os.Exit(0)
	}

	// 读取输入文件
	log.Printf("正在读取文件: %s", *inputFile)
	data, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("读取文件失败: %v", err)
	}

	// 创建转换器
	log.Println("正在创建转换器...")
	converter := NewOpenAPIConverter(data, "json")

	// 执行转换
	log.Println("正在执行转换...")
	if err := converter.Convert(); err != nil {
		log.Fatalf("转换失败: %v", err)
	}

	// 获取转换结果
	result := converter.GetResult()

	// 序列化结果
	log.Println("正在序列化结果...")
	var resultJSON []byte
	if *prettyPrint {
		resultJSON, err = json.MarshalIndent(result, "", "  ")
	} else {
		resultJSON, err = json.Marshal(result)
	}
	if err != nil {
		log.Fatalf("序列化结果失败: %v", err)
	}

	// 输出结果
	if *outputFile != "" {
		// 写入文件
		log.Printf("正在写入输出文件: %s", *outputFile)
		if err := os.WriteFile(*outputFile, resultJSON, 0644); err != nil {
			log.Fatalf("写入文件失败: %v", err)
		}
		log.Printf("✓ 转换完成! 结果已保存到: %s", *outputFile)
	} else {
		// 输出到标准输出
		fmt.Println(string(resultJSON))
	}

	// 打印统计信息
	log.Println("\n转换统计:")
	log.Printf("  - API文档标题: %s", result.Title)
	log.Printf("  - OpenAPI版本: %s", result.ContentVersion)
	log.Printf("  - HTTP API集合数: %d", len(result.HttpCollections))

	totalAPIs := 0
	for _, collection := range result.HttpCollections {
		for _, child := range collection.Children {
			totalAPIs += len(child.Items)
		}
	}
	log.Printf("  - API接口总数: %d", totalAPIs)
	log.Printf("  - 数据模式数: %d", countSchemaItems(result.DataSchemas))
	log.Printf("  - 环境配置数: %d", len(result.Environments))
}

// countSchemaItems 统计schema项数量
func countSchemaItems(schemas []SchemeCollection[SchemaItem]) int {
	count := 0
	for _, schema := range schemas {
		count += len(schema.Items)
	}
	return count
}
