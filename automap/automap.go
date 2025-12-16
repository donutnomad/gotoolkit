package automap

// Option 解析选项
type Option func(string) string

// WithFileContext 设置文件上下文
func WithFileContext(filePath string) Option {
	return func(current string) string {
		return filePath
	}
}
