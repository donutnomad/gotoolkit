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
