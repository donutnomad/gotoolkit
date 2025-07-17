package main

// Parser 定义解析器的通用接口
type Parser interface {
	// Parse 解析输入并返回解析结果
	Parse(input string) (interface{}, error)
}

// InterfaceParserInterface 接口解析器接口
type InterfaceParserInterface interface {
	// ParseFile 解析单个文件
	ParseFile(filename string) (*InterfaceCollection, error)

	// ParseDirectory 解析目录下的所有 Go 文件
	ParseDirectory(dirPath string) (*InterfaceCollection, error)

	// SetConfig 设置解析配置
	SetConfig(config *GenerationConfig)
}

// SwaggerGeneratorInterface Swagger 文档生成器接口
type SwaggerGeneratorInterface interface {
	// GenerateSwaggerComments 生成 Swagger 注释
	GenerateSwaggerComments() (map[string]string, error)

	// GenerateFileHeader 生成文件头部
	GenerateFileHeader(packageName string) string

	// GenerateImports 生成导入声明
	GenerateImports() string

	// SetInterfaces 设置要生成的接口
	SetInterfaces(collection *InterfaceCollection)
}

// GinGeneratorInterface Gin 绑定代码生成器接口
type GinGeneratorInterface interface {
	// GenerateGinCode 生成 Gin 绑定代码
	GenerateGinCode(comments map[string]string) (string, string, error)

	// GenerateComplete 生成完整代码
	GenerateComplete(comments map[string]string) (string, error)

	// SetInterfaces 设置要生成的接口
	SetInterfaces(collection *InterfaceCollection)
}

// ImportManagerInterface 导入管理器接口
type ImportManagerInterface interface {
	// AddImport 添加导入包
	AddImport(pkgPath, alias string) string

	// GetImports 获取所有导入
	GetImports() []string

	// AddTypeReference 添加类型引用
	AddTypeReference(pkgPath, typeName string)

	// GenerateTypeReferences 生成类型引用声明
	GenerateTypeReferences() string

	// Reset 重置管理器状态
	Reset()
}

// TypeParserInterface 类型解析器接口
type TypeParserInterface interface {
	// ParseType 解析类型信息
	ParseType(typeExpr string) (TypeInfo, error)

	// ParseReturnType 解析返回类型
	ParseReturnType(returnType string) (TypeInfo, error)

	// SetImportManager 设置导入管理器
	SetImportManager(mgr ImportManagerInterface)
}

// AnnotationParserInterface 注释解析器接口
type AnnotationParserInterface interface {
	// ParseMethodComments 解析方法注释
	ParseMethodComments(comments []string) ([]interface{}, error)

	// ParseInterfaceComments 解析接口注释
	ParseInterfaceComments(comments []string) ([]interface{}, error)

	// ParseParameterComment 解析参数注释
	ParseParameterComment(comment string) (interface{}, error)
}

// CodeFormatterInterface 代码格式化器接口
type CodeFormatterInterface interface {
	// FormatCode 格式化 Go 代码
	FormatCode(source []byte) ([]byte, error)

	// FormatAndWrite 格式化并写入文件
	FormatAndWrite(filename string, source []byte) error
}

// FileSystemInterface 文件系统操作接口，用于测试时的模拟
type FileSystemInterface interface {
	// ReadFile 读取文件内容
	ReadFile(filename string) ([]byte, error)

	// WriteFile 写入文件内容
	WriteFile(filename string, data []byte, perm uint32) error

	// Exists 检查文件是否存在
	Exists(filename string) bool

	// IsDir 检查是否为目录
	IsDir(path string) bool

	// ListGoFiles 列出目录下的所有 Go 文件
	ListGoFiles(dir string) ([]string, error)
}

// ApplicationInterface 应用程序主接口
type ApplicationInterface interface {
	// Run 运行应用程序
	Run(config *GenerationConfig) error

	// SetParsers 设置解析器组件
	SetParsers(
		interfaceParser InterfaceParserInterface,
		swaggerGenerator SwaggerGeneratorInterface,
		ginGenerator GinGeneratorInterface,
	)

	// SetFileSystem 设置文件系统接口（用于测试）
	SetFileSystem(fs FileSystemInterface)
}

// ComponentFactory 组件工厂接口，用于创建各种组件
type ComponentFactory interface {
	// CreateInterfaceParser 创建接口解析器
	CreateInterfaceParser(config *GenerationConfig) InterfaceParserInterface

	// CreateSwaggerGenerator 创建 Swagger 生成器
	CreateSwaggerGenerator(config *GenerationConfig) SwaggerGeneratorInterface

	// CreateGinGenerator 创建 Gin 生成器
	CreateGinGenerator(config *GenerationConfig) GinGeneratorInterface

	// CreateImportManager 创建导入管理器
	CreateImportManager(packagePath string) ImportManagerInterface

	// CreateTypeParser 创建类型解析器
	CreateTypeParser(mgr ImportManagerInterface) TypeParserInterface

	// CreateAnnotationParser 创建注释解析器
	CreateAnnotationParser() AnnotationParserInterface

	// CreateCodeFormatter 创建代码格式化器
	CreateCodeFormatter() CodeFormatterInterface

	// CreateFileSystem 创建文件系统接口
	CreateFileSystem() FileSystemInterface
}

// ValidatorInterface 验证器接口
type ValidatorInterface interface {
	// ValidateInterface 验证接口定义的有效性
	ValidateInterface(iface SwaggerInterface) error

	// ValidateMethod 验证方法定义的有效性
	ValidateMethod(method SwaggerMethod) error

	// ValidateParameter 验证参数定义的有效性
	ValidateParameter(param Parameter) error

	// ValidateConfig 验证配置的有效性
	ValidateConfig(config *GenerationConfig) error
}

// LoggerInterface 日志记录器接口
type LoggerInterface interface {
	// Debug 记录调试信息
	Debug(format string, args ...interface{})

	// Info 记录信息
	Info(format string, args ...interface{})

	// Warn 记录警告
	Warn(format string, args ...interface{})

	// Error 记录错误
	Error(format string, args ...interface{})

	// SetLevel 设置日志级别
	SetLevel(level string)
}

// TemplateRendererInterface 模板渲染器接口
type TemplateRendererInterface interface {
	// RenderSwaggerComment 渲染 Swagger 注释模板
	RenderSwaggerComment(data interface{}) (string, error)

	// RenderGinHandler 渲染 Gin 处理器模板
	RenderGinHandler(data interface{}) (string, error)

	// RenderBindMethod 渲染绑定方法模板
	RenderBindMethod(data interface{}) (string, error)

	// LoadCustomTemplate 加载自定义模板
	LoadCustomTemplate(name, content string) error
}
