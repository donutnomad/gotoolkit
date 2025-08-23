package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/donutnomad/gotoolkit/internal/gofmt"
	parsers "github.com/donutnomad/gotoolkit/swagGen/parser"
)

// SwagGenApplication main application structure
type SwagGenApplication struct {
	config           *GenerationConfig
	interfaceParser  InterfaceParserInterface
	swaggerGenerator SwaggerGeneratorInterface
	ginGenerator     GinGeneratorInterface
	logger           LoggerInterface
	fileSystem       FileSystemInterface
}

// NewSwagGenApplication creates a new application instance
func NewSwagGenApplication(config *GenerationConfig) *SwagGenApplication {
	app := &SwagGenApplication{
		config:     config,
		logger:     NewConsoleLogger(config.Verbose),
		fileSystem: NewDefaultFileSystem(),
	}

	// Initialize components
	app.initializeComponents()
	return app
}

// initializeComponents initializes all components
func (app *SwagGenApplication) initializeComponents() {
	// Create import manager
	importMgr := NewEnhancedImportManager("")

	// Create parser adapter
	app.interfaceParser = NewInterfaceParserAdapter(importMgr)
	app.swaggerGenerator = NewSwaggerGeneratorAdapter(nil)
	app.ginGenerator = NewGinGeneratorAdapter(nil)
}

// Run executes the main application logic
func (app *SwagGenApplication) Run() error {
	app.logger.Info("starting swagGen execution...")

	// Validate configuration
	if err := app.config.Validate(); err != nil {
		return NewValidationError("configuration validation failed", err.Error())
	}

	// Parse interfaces
	collection, err := app.parseInterfaces()
	if err != nil {
		return err
	}

	// Filter interfaces
	if err := app.filterInterfaces(collection); err != nil {
		return err
	}

	// 排序collection.Interfaces按name从小到大
	sort.Slice(collection.Interfaces, func(i, j int) bool {
		return collection.Interfaces[i].Name < collection.Interfaces[j].Name
	})

	// Validate interfaces
	if err := app.validateInterfaces(collection); err != nil {
		return err
	}

	// Generate code
	output, err := app.generateCode(collection)
	if err != nil {
		return err
	}

	// Write file
	if err := app.writeOutput(output); err != nil {
		return err
	}

	app.logger.Info("swagGen execution completed")
	return nil
}

// parseInterfaces parses interface definitions
func (app *SwagGenApplication) parseInterfaces() (*InterfaceCollection, error) {
	app.logger.Info("starting interface parsing...")

	var collection *InterfaceCollection
	var err error

	// Check path type
	if app.fileSystem.IsDir(app.config.Path) {
		app.logger.Debug("parsing directory: %s", app.config.Path)
		collection, err = app.interfaceParser.ParseDirectory(app.config.Path)
	} else {
		app.logger.Debug("parsing file: %s", app.config.Path)
		collection, err = app.interfaceParser.ParseFile(app.config.Path)
	}

	if err != nil {
		return nil, NewParseError("interface parsing failed", "", err)
	}

	app.logger.Info("parsing completed, found %d interfaces", len(collection.Interfaces))
	return collection, nil
}

// filterInterfaces filters interfaces
func (app *SwagGenApplication) filterInterfaces(collection *InterfaceCollection) error {
	if len(app.config.Interfaces) == 0 {
		return nil
	}

	app.logger.Debug("filtering interfaces: %v", app.config.Interfaces)

	originalCount := len(collection.Interfaces)
	collection.Interfaces = app.filterInterfacesByNames(collection.Interfaces, app.config.Interfaces)

	app.logger.Info("interface filtering completed: %d -> %d", originalCount, len(collection.Interfaces))
	return nil
}

// filterInterfacesByNames filters interfaces by names
func (app *SwagGenApplication) filterInterfacesByNames(interfaces []SwaggerInterface, names []string) []SwaggerInterface {
	var filtered []SwaggerInterface

	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[strings.TrimSpace(name)] = true
	}

	for _, iface := range interfaces {
		if nameSet[iface.Name] {
			filtered = append(filtered, iface)
		}
	}

	return filtered
}

// validateInterfaces validates the validity of interfaces
func (app *SwagGenApplication) validateInterfaces(collection *InterfaceCollection) error {
	if len(collection.Interfaces) == 0 {
		return NewValidationError("no valid interfaces found", "please check if interface definitions contain correct Swagger comments")
	}

	app.logger.Debug("validating %d interfaces", len(collection.Interfaces))

	for _, iface := range collection.Interfaces {
		if len(iface.Methods) == 0 {
			app.logger.Warn("interface %s contains no methods", iface.Name)
			continue
		}

		app.logger.Debug("  - %s (%d methods)", iface.Name, len(iface.Methods))
	}

	return nil
}

// generateCode generates complete code
func (app *SwagGenApplication) generateCode(collection *InterfaceCollection) (string, error) {
	app.logger.Info("starting code generation...")

	// Set package path
	packagePath := app.getPackagePath()
	collection.ImportMgr.packagePath = packagePath

	// Set generator interface collection
	app.swaggerGenerator.SetInterfaces(collection)
	app.ginGenerator.SetInterfaces(collection)

	var parts []string

	// Generate file header
	header := app.swaggerGenerator.GenerateFileHeader(app.inferPackageName())
	parts = append(parts, header)

	// Mark used packages
	app.markUsedPackages(collection)

	// Generate import declarations
	imports := app.swaggerGenerator.GenerateImports()
	if imports != "" {
		parts = append(parts, imports, "")
	}

	// Generate type references
	if !app.config.SkipTypeReference {
		typeRefs, err := app.generateTypeReferences(collection)
		if err != nil {
			return "", NewGenerateError("type reference generation failed", "", err)
		}
		if typeRefs != "" {
			parts = append(parts, typeRefs, "")
		}
	}

	// Generate Swagger comments
	swaggerComments, err := app.swaggerGenerator.GenerateSwaggerComments()
	if err != nil {
		return "", NewGenerateError("swagger comment generation failed", "", err)
	}

	// Generate Gin binding code
	ginCode, err := app.ginGenerator.GenerateComplete(swaggerComments)
	if err != nil {
		return "", NewGenerateError("gin code generation failed", "", err)
	}

	if ginCode != "" {
		parts = append(parts, ginCode)
	}

	result := strings.Join(parts, "\n")
	app.logger.Info("code generation completed, total length: %d characters", len(result))

	return result, nil
}

// markUsedPackages marks used packages
func (app *SwagGenApplication) markUsedPackages(collection *InterfaceCollection) {
	for _, iface := range collection.Interfaces {
		for _, method := range iface.Methods {
			// Mark packages used by return types
			packages := parsers.ExtractPackages(method.ResponseType.FullName)
			for _, pkgName := range packages {
				app.markPackageAsUsed(collection.ImportMgr, pkgName)
			}

			// Mark packages used by parameter types
			for _, param := range method.Parameters {
				paramPackages := parsers.ExtractPackages(param.Type.FullName)
				for _, pkgName := range paramPackages {
					app.markPackageAsUsed(collection.ImportMgr, pkgName)
				}
			}
		}
	}
}

// markPackageAsUsed marks a package as used
func (app *SwagGenApplication) markPackageAsUsed(importMgr *EnhancedImportManager, pkgName string) {
	for _, info := range importMgr.imports {
		parts := strings.Split(info.Path, "/")
		if len(parts) > 0 && pkgName == parts[len(parts)-1] {
			info.DirectlyUsed = true
		}
	}
}

// generateTypeReferences generates type references
func (app *SwagGenApplication) generateTypeReferences(collection *InterfaceCollection) (string, error) {
	// Use original logic to generate type references
	// This feature uses simplified implementation for now, need to check original SwaggerGenerator implementation
	var refs []string

	// Collect all types that need references
	typeSet := make(map[string]bool)

	for _, iface := range collection.Interfaces {
		for _, method := range iface.Methods {
			// Add return type references
			if method.ResponseType.FullName != "" && method.ResponseType.Package != "" {
				typeDef := fmt.Sprintf("var _ %s", method.ResponseType.FullName)
				typeSet[typeDef] = true
			}

			// Add parameter type references
			for _, param := range method.Parameters {
				if param.Type.FullName != "" && param.Type.Package != "" {
					typeDef := fmt.Sprintf("var _ %s", param.Type.FullName)
					typeSet[typeDef] = true
				}
			}
		}
	}

	// Convert to slice and sort
	for typeDef := range typeSet {
		refs = append(refs, typeDef)
	}

	if len(refs) > 0 {
		result := TypeReferenceComment + "\n" + strings.Join(refs, "\n")
		return result, nil
	}

	return "", nil
}

// writeOutput writes output file
func (app *SwagGenApplication) writeOutput(output string) error {
	app.logger.Info("starting output file writing...")

	// Determine output path
	outputPath := app.determineOutputPath()

	// Format code
	formattedBytes, err := gofmt.FormatBytes([]byte(output))
	if err != nil {
		return NewGenerateError("code formatting failed", "", err)
	}

	// Write file
	if err := app.fileSystem.WriteFile(outputPath, formattedBytes, 0644); err != nil {
		return NewFileError("failed to write file", outputPath, err)
	}

	app.logger.Info("successfully generated file: %s", outputPath)
	return nil
}

// determineOutputPath determines output file path
func (app *SwagGenApplication) determineOutputPath() string {
	outputPath := app.config.OutputFile

	// If output path is already absolute, use it directly
	if filepath.IsAbs(outputPath) {
		return outputPath
	}

	// If output path is relative, determine base directory based on input path
	var baseDir string
	if app.fileSystem.IsDir(app.config.Path) {
		// Input is directory, put output file in that directory
		baseDir = app.config.Path
	} else {
		// Input is file, put output file in the same directory as the file
		baseDir = filepath.Dir(app.config.Path)
	}

	// Build final output path
	return filepath.Join(baseDir, outputPath)
}

// getPackagePath gets package path
func (app *SwagGenApplication) getPackagePath() string {
	path := app.config.Path

	if !app.fileSystem.IsDir(path) {
		path = filepath.Dir(path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return filepath.Base(path)
	}

	return absPath
}

// inferPackageName infers package name
func (app *SwagGenApplication) inferPackageName() string {
	if app.config.Package != "" {
		return app.config.Package
	}

	// Infer package name from file or directory
	if !app.fileSystem.IsDir(app.config.Path) {
		// If it's a file, parse file to get package name
		if pkgName := app.extractPackageNameFromFile(app.config.Path); pkgName != "" {
			return pkgName
		}
		// Use directory name of the file
		return filepath.Base(filepath.Dir(app.config.Path))
	}

	// If it's a directory, try to get package name from Go files in the directory
	if pkgName := app.extractPackageNameFromDir(app.config.Path); pkgName != "" {
		return pkgName
	}

	// Use directory name as package name
	return filepath.Base(app.config.Path)
}

// extractPackageNameFromFile extracts package name from file
func (app *SwagGenApplication) extractPackageNameFromFile(filename string) string {
	content, err := app.fileSystem.ReadFile(filename)
	if err != nil {
		return ""
	}

	// Need to implement actual package name extraction logic here
	// Use simple string matching for now
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "package ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}

	return ""
}

// extractPackageNameFromDir extracts package name from directory
func (app *SwagGenApplication) extractPackageNameFromDir(dir string) string {
	files, err := app.fileSystem.ListGoFiles(dir)
	if err != nil {
		return ""
	}

	for _, filename := range files {
		if !strings.HasSuffix(filename, "_test.go") {
			if pkgName := app.extractPackageNameFromFile(filename); pkgName != "" {
				return pkgName
			}
		}
	}

	return ""
}

// SetLogger sets logger
func (app *SwagGenApplication) SetLogger(logger LoggerInterface) {
	app.logger = logger
}

// SetFileSystem sets file system interface
func (app *SwagGenApplication) SetFileSystem(fs FileSystemInterface) {
	app.fileSystem = fs
}
