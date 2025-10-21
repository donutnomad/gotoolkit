package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

func NameWithoutPoint(name string) string {
	if strings.HasPrefix(name, "*") {
		return name[1:]
	}
	return name
}

func GetFullPathWithPackage(filePath string) (string, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}

	// Configure package loading
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedModule,
		Dir:  filepath.Dir(absPath),
		Env:  append(os.Environ(), "GO111MODULE=on"),
	}

	// Load the package containing the file
	pkgs, err := packages.Load(cfg, "file="+absPath)
	if err != nil {
		return "", err
	}

	if len(pkgs) == 0 {
		return "", fmt.Errorf("no package found for file: %s", filePath)
	}

	pkg := pkgs[0]

	importPath := pkg.PkgPath
	if importPath == "" {
		if pkg.Module != nil {
			dir := filepath.Dir(absPath)
			relPath, err := filepath.Rel(pkg.Module.Dir, dir)
			if err == nil {
				importPath = filepath.Join(pkg.Module.Path, relPath)
			}
		}
	}

	return importPath, nil
}
