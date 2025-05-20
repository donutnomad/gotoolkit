package xast

import (
	"fmt"
	utils2 "github.com/donutnomad/gotoolkit/internal/utils"
	"github.com/samber/lo"
	"go/ast"
	"iter"
	"path/filepath"
)

// ImportInfo tracks package imports and their aliases
type ImportInfo struct {
	Path  string // Full import path
	Alias string // Import alias (empty for default import name)
}

func (i ImportInfo) String() string {
	var alias = i.GetAlias()
	var path = utils2.RemoveQuotes(i.GetPath())
	if alias != "" {
		return fmt.Sprintf("%s \"%s\"", alias, path)
	} else {
		return fmt.Sprintf("\"%s\"", path)
	}
}

func (i ImportInfo) GetPath() string {
	return i.Path
}

func (i ImportInfo) GetAlias() string {
	if i.Alias != "" {
		base := filepath.Base(i.Path)
		if i.Alias != base {
			return i.Alias
		}
	}
	return ""
}

func (i ImportInfo) HasAlias() bool {
	return i.GetAlias() != ""
}

func (i ImportInfo) GetBase() string {
	if i.Alias != "" {
		return i.Alias
	}
	return filepath.Base(i.Path)
}

type ImportInfoSlice []ImportInfo

func (s ImportInfoSlice) Find(pkg string) *ImportInfo {
	find, ok := lo.Find(s, func(item ImportInfo) bool {
		return item.GetBase() == pkg
	})
	if ok {
		return &find
	}
	return nil
}

func (s ImportInfoSlice) From(specs []*ast.ImportSpec) ImportInfoSlice {
	var out ImportInfoSlice
	for _, n := range specs {
		// lo2 "github.com/samber/lo"
		// ↑         ↑
		// alias    path
		var alias = ""
		if n.Name != nil {
			alias = n.Name.Name
		}
		var path = utils2.RemoveQuotes(n.Path.Value)
		out = append(out, ImportInfo{
			Alias: alias,
			Path:  path,
		})
	}
	return out
}

type ImportManager struct {
	imports     map[string]*ImportInfo // key: full import path, value: import info
	packagePath string
}

func NewImportManager(packagePath string) *ImportManager {
	return &ImportManager{
		packagePath: packagePath,
		imports:     make(map[string]*ImportInfo),
	}
}

func (g *ImportManager) Iter() iter.Seq2[string, *ImportInfo] {
	return utils2.IterSortMap(g.imports)
}

func (g *ImportManager) GetAliasAndPath(path string) (alias, _ string) {
	if alias, ok := g.imports[path]; ok {
		if len(alias.Alias) == 0 {
			return "", path
		} else {
			return alias.Alias, path
		}
	}
	return "", path
}

//func (g *ImportManager) GetAliasAndPath2(path string) (alias, _ string) {
//	if alias, ok := g.imports[path]; ok {
//		if len(alias.Alias) == 0 {
//			return filepath.Base(path), path
//		} else {
//			return alias.Alias, path
//		}
//	}
//	return "", path
//}

// AddImport adds a package import and returns the package identifier to use
func (g *ImportManager) AddImport(path string) {
	path = utils2.RemoveQuotes(path)
	//fmt.Printf("\nDEBUG: Adding import for path: %s\n", path)

	// Skip if it's in the same package
	if path == g.packagePath {
		return
	}

	// Check if we already have this import
	if _, exists := g.imports[path]; exists {
		return
	}

	// Get base package name
	baseName := filepath.Base(path)
	//fmt.Printf("DEBUG: Base name for import: %s\n", baseName)

	// 检查是否已存在相同的包名，如果存在则添加数字后缀
	alias := g.countAlias(baseName)

	g.imports[path] = &ImportInfo{
		Path:  path,
		Alias: lo.Ternary(alias != baseName, alias, ""),
	}
}

func (g *ImportManager) countAlias(baseName string) string {
	var cache = make(map[string]struct{})

	for path, value := range g.imports {
		if value.Alias != "" {
			cache[value.Alias] = struct{}{}
		} else {
			cache[filepath.Base(path)] = struct{}{}
		}
	}

	if _, ok := cache[baseName]; !ok {
		return baseName
	}

	for i := 2; ; i++ {
		alias := fmt.Sprintf("%s%d", baseName, i)
		if _, ok := cache[alias]; !ok {
			return alias
		}
	}
}
