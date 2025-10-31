package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 日志等级常量
const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

// 日志等级优先级映射
var logLevelPriority = map[string]int{
	LogLevelDebug: 0,
	LogLevelInfo:  1,
	LogLevelWarn:  2,
	LogLevelError: 3,
}

// ConsoleLogger 控制台日志记录器实现
type ConsoleLogger struct {
	level string // debug/info/warn/error
}

// NewConsoleLogger 创建控制台日志记录器
func NewConsoleLogger(level string) *ConsoleLogger {
	// 默认日志等级为 info
	if level == "" {
		level = LogLevelInfo
	}
	return &ConsoleLogger{
		level: level,
	}
}

// shouldLog 判断是否应该输出日志
func (l *ConsoleLogger) shouldLog(msgLevel string) bool {
	currentPriority, ok1 := logLevelPriority[l.level]
	msgPriority, ok2 := logLevelPriority[msgLevel]

	// 如果等级未知，默认输出
	if !ok1 || !ok2 {
		return true
	}

	// 只有消息等级 >= 设置的等级时才输出
	return msgPriority >= currentPriority
}

// Debug 记录调试信息
func (l *ConsoleLogger) Debug(format string, args ...interface{}) {
	if l.shouldLog(LogLevelDebug) {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// Info 记录信息
func (l *ConsoleLogger) Info(format string, args ...interface{}) {
	if l.shouldLog(LogLevelInfo) {
		fmt.Printf("[INFO] "+format+"\n", args...)
	}
}

// Warn 记录警告
func (l *ConsoleLogger) Warn(format string, args ...interface{}) {
	if l.shouldLog(LogLevelWarn) {
		fmt.Printf("[WARN] "+format+"\n", args...)
	}
}

// Error 记录错误
func (l *ConsoleLogger) Error(format string, args ...interface{}) {
	if l.shouldLog(LogLevelError) {
		fmt.Printf("[ERROR] "+format+"\n", args...)
	}
}

// SetLevel 设置日志级别
func (l *ConsoleLogger) SetLevel(level string) {
	l.level = level
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
