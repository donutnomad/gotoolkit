package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/donutnomad/gotoolkit/internal/utils"
	"github.com/donutnomad/gotoolkit/sliceGen/generator"
	"github.com/samber/lo"
)

var (
	structName    = flag.String("struct", "", "struct name to process, format: [package_path/]struct_name1,struct_name2...")
	ignoreFields  = flag.String("ignoreFields", "", "fields to ignore, comma separated")
	includeFields = flag.String("includeFields", "", "fields to include (takes precedence over ignoreFields), comma separated")
	extraMethods  = flag.String("methods", "", "extra methods to generate: filter,map,reduce,sort,groupby (comma separated)")
	usePointer    = flag.Bool("ptr", true, "generate pointer-based methods and slice types (e.g., []*StructName instead of []StructName)")
	outputFile    = flag.String("o", "slice_generated.go", "output file name (default: slice_generated.go)")
)

func main() {
	flag.Parse()
	if err := run(*structName, *ignoreFields, *includeFields, *extraMethods, *usePointer, *outputFile); err != nil {
		panic(err)
	}
}

func run(structName, ignoreFields, includeFields, methods string, usePointer bool, outputFile string) error {
	if structName == "" {
		return fmt.Errorf("struct parameter is required")
	}

	// Parse type name and package path
	var packagePath string
	var structNames []string

	parts := strings.Split(structName, "/")
	if len(parts) == 1 {
		structNames = strings.Split(parts[0], ",")
		packagePath = "."
	} else {
		structNames = strings.Split(parts[len(parts)-1], ",")
		packagePath = strings.Join(parts[:len(parts)-1], "/")
	}

	// Clean struct names
	for i, name := range structNames {
		structNames[i] = strings.TrimSpace(name)
	}

	// Parse include fields (takes precedence over ignoreFields)
	includeFieldsMap := make(map[string]bool)
	if includeFields != "" {
		for _, f := range strings.Split(includeFields, ",") {
			includeFieldsMap[strings.TrimSpace(f)] = true
		}
	}

	// Parse ignore fields (only used if includeFields is empty)
	ignoreFieldsMap := make(map[string]bool)
	if includeFields == "" && ignoreFields != "" {
		for _, f := range strings.Split(ignoreFields, ",") {
			ignoreFieldsMap[strings.TrimSpace(f)] = true
		}
	}

	var allExtraMethods = utils.DefSlice(
		generator.MethodFilter,
		generator.MethodMap,
		generator.MethodGroupBy,
		generator.MethodReduce,
		generator.MethodSort,
	)

	// Parse extra methods
	var methodsList []string
	if methods != "" {
		methodsList = strings.Split(methods, ",")
		for _, method := range methodsList {
			if !lo.ContainsBy(allExtraMethods, func(item generator.MyGenerator[generator.MethodTemplateData]) bool {
				return strings.EqualFold(item.Name, method)
			}) {
				return fmt.Errorf("unknown extra method: %s", method)
			}
		}
	}
	extraMethodsMap := lo.SliceToMap(methodsList, func(method string) (string, generator.IExecute) {
		v, _ := lo.Find(allExtraMethods, func(item generator.MyGenerator[generator.MethodTemplateData]) bool {
			return strings.EqualFold(item.Name, method)
		})
		return method, &v
	})

	// Create generator
	g := generator.NewGenerator(structNames, packagePath, ignoreFieldsMap, includeFieldsMap, extraMethodsMap, usePointer)

	// Generate code
	generatedCode, err := g.Generate()
	if err != nil {
		return err
	}

	// Format the generated code
	formattedCode, err := format.Source([]byte(generatedCode.String()))
	if err != nil {
		return fmt.Errorf("failed to format generated code: %v", err)
	}

	// Write to file
	outputDir := lo.Ternary(packagePath == ".", "", packagePath)
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}

	outputName := filepath.Join(outputDir, outputFile)
	if err := os.WriteFile(outputName, formattedCode, 0644); err != nil {
		return fmt.Errorf("failed to write generated code: %v", err)
	}

	return nil
}
