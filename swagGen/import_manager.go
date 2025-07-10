package main

import (
	"fmt"
	"github.com/samber/lo"
	"path/filepath"
	"sort"
	"strings"
)

// NewEnhancedImportManager 创建增强的导入管理器
func NewEnhancedImportManager(packagePath string) *EnhancedImportManager {
	return &EnhancedImportManager{
		imports:        make(map[string]*ImportInfo),
		aliasCounter:   make(map[string]int),
		aliasMapping:   make(map[string]string),
		typeReferences: make(map[string][]string),
		packagePath:    packagePath,
	}
}

// AddTypeReference 添加类型引用并返回别名
func (mgr *EnhancedImportManager) AddTypeReference(pkgPath, typeName string) string {
	if pkgPath == "" || pkgPath == mgr.packagePath {
		return ""
	}

	alias := mgr.ensureAlias(pkgPath)

	// 添加类型引用
	if _, exists := mgr.typeReferences[pkgPath]; !exists {
		mgr.typeReferences[pkgPath] = []string{}
	}

	// 避免重复添加相同的类型
	if !lo.Contains(mgr.typeReferences[pkgPath], typeName) {
		mgr.typeReferences[pkgPath] = append(mgr.typeReferences[pkgPath], typeName)
	}

	return alias
}

// ensureAlias 确保包有别名
func (mgr *EnhancedImportManager) ensureAlias(pkgPath string) string {
	if alias, exists := mgr.aliasMapping[pkgPath]; exists {
		return alias
	}

	baseName := filepath.Base(pkgPath)

	// 清理 baseName，移除版本后缀
	baseName = strings.TrimSuffix(baseName, ".git")
	if strings.HasPrefix(baseName, "v") && len(baseName) > 1 {
		// 检查是否是版本号 (v1, v2, v3, etc.)
		for i := 1; i < len(baseName); i++ {
			if baseName[i] < '0' || baseName[i] > '9' {
				break
			}
			if i == len(baseName)-1 {
				// 获取上一级目录名
				parentPath := filepath.Dir(pkgPath)
				if parentPath != "." && parentPath != "/" {
					baseName = filepath.Base(parentPath)
				}
			}
		}
	}

	if mgr.aliasCounter[baseName] == 0 {
		// 第一次出现，使用原名
		mgr.aliasMapping[pkgPath] = baseName
		mgr.aliasCounter[baseName] = 1
		mgr.imports[pkgPath] = &ImportInfo{
			Path:         pkgPath,
			Alias:        baseName,
			Used:         true,
			DirectlyUsed: false, // 默认为仅类型引用
		}
		return baseName
	}

	// 已存在同名包，使用别名
	mgr.aliasCounter[baseName]++
	alias := fmt.Sprintf("%s%d", baseName, mgr.aliasCounter[baseName])
	mgr.aliasMapping[pkgPath] = alias
	mgr.imports[pkgPath] = &ImportInfo{
		Path:         pkgPath,
		Alias:        alias,
		Used:         true,
		DirectlyUsed: false, // 默认为仅类型引用
	}
	return alias
}

// AddImport 添加导入 - 标记为直接使用
func (mgr *EnhancedImportManager) AddImport(pkgPath string) string {
	alias := mgr.ensureAlias(pkgPath)
	if info, exists := mgr.imports[pkgPath]; exists {
		info.DirectlyUsed = true // 标记为直接使用
	}
	return alias
}

// GetAlias 获取包别名
func (mgr *EnhancedImportManager) GetAlias(pkgPath string) string {
	if alias, exists := mgr.aliasMapping[pkgPath]; exists {
		return alias
	}
	return ""
}

// GetImportDeclarations 获取导入声明
func (mgr *EnhancedImportManager) GetImportDeclarations() string {
	if len(mgr.imports) == 0 {
		return ""
	}

	var imports []string
	var standardImports []string
	var thirdPartyImports []string
	var localImports []string

	// 对导入进行分类
	for _, info := range mgr.imports {
		if !info.Used {
			continue
		}

		var line string
		if info.DirectlyUsed {
			// 直接使用的包 - 使用别名
			line = fmt.Sprintf("\t%s \"%s\"", info.Alias, info.Path)
		} else {
			// 仅类型引用的包 - 使用 _ 前缀
			line = fmt.Sprintf("\t_ \"%s\"", info.Path)
		}

		// 分类导入
		if isStandardLibrary(info.Path) {
			standardImports = append(standardImports, line)
		} else if strings.Contains(info.Path, mgr.packagePath) {
			localImports = append(localImports, line)
		} else {
			thirdPartyImports = append(thirdPartyImports, line)
		}
	}

	// 排序
	sort.Strings(standardImports)
	sort.Strings(thirdPartyImports)
	sort.Strings(localImports)

	// 组合导入
	imports = append(imports, standardImports...)
	if len(thirdPartyImports) > 0 {
		if len(imports) > 0 {
			imports = append(imports, "")
		}
		imports = append(imports, thirdPartyImports...)
	}
	if len(localImports) > 0 {
		if len(imports) > 0 {
			imports = append(imports, "")
		}
		imports = append(imports, localImports...)
	}

	return "import (\n" + strings.Join(imports, "\n") + "\n)"
}

// GetTypeReferences 获取类型引用声明 - 已禁用
func (mgr *EnhancedImportManager) GetTypeReferences() string {
	// 不再生成强制导入的变量声明
	return ""
}

// isStandardLibrary 检查是否是标准库
func isStandardLibrary(path string) bool {
	// 简单的标准库检查
	standardLibPrefixes := []string{
		"bufio", "bytes", "compress", "container", "context", "crypto",
		"database", "debug", "encoding", "errors", "expvar", "flag",
		"fmt", "go", "hash", "html", "image", "index", "io", "log",
		"math", "mime", "net", "os", "path", "plugin", "reflect",
		"regexp", "runtime", "sort", "strconv", "strings", "sync",
		"syscall", "testing", "text", "time", "unicode", "unsafe",
	}

	for _, prefix := range standardLibPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}
