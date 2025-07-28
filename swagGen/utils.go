package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ConsoleLogger 控制台日志记录器实现
type ConsoleLogger struct {
	verbose bool
	level   string
}

// NewConsoleLogger 创建控制台日志记录器
func NewConsoleLogger(verbose bool) *ConsoleLogger {
	level := "info"
	if verbose {
		level = "debug"
	}
	return &ConsoleLogger{
		verbose: verbose,
		level:   level,
	}
}

// Debug 记录调试信息
func (l *ConsoleLogger) Debug(format string, args ...interface{}) {
	if l.verbose {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// Info 记录信息
func (l *ConsoleLogger) Info(format string, args ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", args...)
}

// Warn 记录警告
func (l *ConsoleLogger) Warn(format string, args ...interface{}) {
	fmt.Printf("[WARN] "+format+"\n", args...)
}

// Error 记录错误
func (l *ConsoleLogger) Error(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
}

// SetLevel 设置日志级别
func (l *ConsoleLogger) SetLevel(level string) {
	l.level = level
	l.verbose = (level == "debug")
}

// DefaultFileSystem 默认文件系统实现
type DefaultFileSystem struct{}

// NewDefaultFileSystem 创建默认文件系统
func NewDefaultFileSystem() *DefaultFileSystem {
	return &DefaultFileSystem{}
}

// ReadFile 读取文件内容
func (fs *DefaultFileSystem) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// WriteFile 写入文件内容
func (fs *DefaultFileSystem) WriteFile(filename string, data []byte, perm uint32) error {
	return os.WriteFile(filename, data, os.FileMode(perm))
}

// Exists 检查文件是否存在
func (fs *DefaultFileSystem) Exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// IsDir 检查是否为目录
func (fs *DefaultFileSystem) IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ListGoFiles 列出目录下的所有 Go 文件
func (fs *DefaultFileSystem) ListGoFiles(dir string) ([]string, error) {
	var goFiles []string

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// 只处理当前目录的文件，不递归到子目录
		if file.IsDir() {
			continue
		}

		if strings.HasSuffix(file.Name(), ".go") {
			goFiles = append(goFiles, filepath.Join(dir, file.Name()))
		}
	}

	return goFiles, nil
}

// BasicValidator 基础验证器实现
type BasicValidator struct{}

// NewBasicValidator 创建基础验证器
func NewBasicValidator() *BasicValidator {
	return &BasicValidator{}
}

// ValidateInterface 验证接口定义的有效性
func (v *BasicValidator) ValidateInterface(iface SwaggerInterface) error {
	if iface.Name == "" {
		return NewValidationError("interface name is empty", "interface must have a name")
	}

	if len(iface.Methods) == 0 {
		return NewValidationError("interface methods are empty", fmt.Sprintf("interface %s has no methods defined", iface.Name))
	}

	// 验证每个方法
	for _, method := range iface.Methods {
		if err := v.ValidateMethod(method); err != nil {
			return fmt.Errorf("interface %s method validation failed: %w", iface.Name, err)
		}
	}

	return nil
}

// ValidateMethod 验证方法定义的有效性
func (v *BasicValidator) ValidateMethod(method SwaggerMethod) error {
	if method.Name == "" {
		return NewValidationError("method name is empty", "method must have a name")
	}

	// 检查是否有路由定义
	paths := method.GetPaths()
	if len(paths) == 0 {
		return NewValidationError("method missing route definition",
			fmt.Sprintf("方法 %s 必须包含路由注释（如 @GET、@POST 等）", method.Name))
	}

	// 验证HTTP方法
	httpMethod := method.GetHTTPMethod()
	if !v.isValidHTTPMethod(httpMethod) {
		return NewValidationError("invalid HTTP method",
			fmt.Sprintf("方法 %s 包含无效的HTTP方法: %s", method.Name, httpMethod))
	}

	// 验证参数
	for _, param := range method.Parameters {
		if err := v.ValidateParameter(param); err != nil {
			return fmt.Errorf("method %s parameter validation failed: %w", method.Name, err)
		}
	}

	return nil
}

// ValidateParameter 验证参数定义的有效性
func (v *BasicValidator) ValidateParameter(param Parameter) error {
	if param.Name == "" {
		return NewValidationError("parameter name is empty", "parameter must have a name")
	}

	if param.Type.FullName == "" {
		return NewValidationError("parameter type is empty",
			fmt.Sprintf("参数 %s 必须有类型定义", param.Name))
	}

	// 验证参数来源
	if param.Source != "" && !v.isValidParameterSource(param.Source) {
		return NewValidationError("invalid parameter source",
			fmt.Sprintf("参数 %s 的来源 %s 无效", param.Name, param.Source))
	}

	return nil
}

// ValidateConfig 验证配置的有效性
func (v *BasicValidator) ValidateConfig(config *GenerationConfig) error {
	return config.Validate()
}

// isValidHTTPMethod 检查是否为有效的HTTP方法
func (v *BasicValidator) isValidHTTPMethod(method string) bool {
	validMethods := []string{
		HTTPMethodGET, HTTPMethodPOST, HTTPMethodPUT,
		HTTPMethodPATCH, HTTPMethodDELETE,
	}

	for _, valid := range validMethods {
		if method == valid {
			return true
		}
	}
	return false
}

// isValidParameterSource 检查是否为有效的参数来源
func (v *BasicValidator) isValidParameterSource(source string) bool {
	validSources := []string{
		ParamSourcePath, ParamSourceQuery, ParamSourceHeader,
		ParamSourceBody, ParamSourceForm,
	}

	for _, valid := range validSources {
		if source == valid {
			return true
		}
	}
	return false
}

// NullLogger 空日志记录器，用于测试
type NullLogger struct{}

// NewNullLogger 创建空日志记录器
func NewNullLogger() *NullLogger {
	return &NullLogger{}
}

// Debug 不记录任何信息
func (l *NullLogger) Debug(format string, args ...interface{}) {}

// Info 不记录任何信息
func (l *NullLogger) Info(format string, args ...interface{}) {}

// Warn 不记录任何信息
func (l *NullLogger) Warn(format string, args ...interface{}) {}

// Error 不记录任何信息
func (l *NullLogger) Error(format string, args ...interface{}) {}

// SetLevel 不做任何操作
func (l *NullLogger) SetLevel(level string) {}

// MemoryFileSystem 内存文件系统实现，用于测试
type MemoryFileSystem struct {
	files map[string][]byte
	dirs  map[string]bool
}

// NewMemoryFileSystem 创建内存文件系统
func NewMemoryFileSystem() *MemoryFileSystem {
	return &MemoryFileSystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

// ReadFile 从内存读取文件内容
func (fs *MemoryFileSystem) ReadFile(filename string) ([]byte, error) {
	if content, exists := fs.files[filename]; exists {
		return content, nil
	}
	return nil, fmt.Errorf("file does not exist: %s", filename)
}

// WriteFile 向内存写入文件内容
func (fs *MemoryFileSystem) WriteFile(filename string, data []byte, perm uint32) error {
	fs.files[filename] = data
	// 同时标记父目录存在
	dir := filepath.Dir(filename)
	if dir != "." {
		fs.dirs[dir] = true
	}
	return nil
}

// Exists 检查文件是否存在
func (fs *MemoryFileSystem) Exists(filename string) bool {
	_, exists := fs.files[filename]
	if !exists {
		_, exists = fs.dirs[filename]
	}
	return exists
}

// IsDir 检查是否为目录
func (fs *MemoryFileSystem) IsDir(path string) bool {
	return fs.dirs[path]
}

// ListGoFiles 列出目录下的所有 Go 文件
func (fs *MemoryFileSystem) ListGoFiles(dir string) ([]string, error) {
	var goFiles []string

	for filename := range fs.files {
		fileDir := filepath.Dir(filename)
		if fileDir == dir && strings.HasSuffix(filename, ".go") {
			goFiles = append(goFiles, filename)
		}
	}

	return goFiles, nil
}

// AddFile 添加文件到内存文件系统（用于测试）
func (fs *MemoryFileSystem) AddFile(filename string, content []byte) {
	fs.files[filename] = content
}

// AddDir 添加目录到内存文件系统（用于测试）
func (fs *MemoryFileSystem) AddDir(dirname string) {
	fs.dirs[dirname] = true
}
