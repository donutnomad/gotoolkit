package main

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
