package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/donutnomad/gotoolkit/swagGen/gofmt"
	parsers "github.com/donutnomad/gotoolkit/swagGen/parser"
)

// SwagGenApplication 应用程序主结构
type SwagGenApplication struct {
	config           *GenerationConfig
	interfaceParser  InterfaceParserInterface
	swaggerGenerator SwaggerGeneratorInterface
	ginGenerator     GinGeneratorInterface
	logger           LoggerInterface
	fileSystem       FileSystemInterface
}

// NewSwagGenApplication 创建新的应用程序实例
func NewSwagGenApplication(config *GenerationConfig) *SwagGenApplication {
	app := &SwagGenApplication{
		config:     config,
		logger:     NewConsoleLogger(config.Verbose),
		fileSystem: NewDefaultFileSystem(),
	}

	// 初始化组件
	app.initializeComponents()
	return app
}

// initializeComponents 初始化各个组件
func (app *SwagGenApplication) initializeComponents() {
	// 创建导入管理器
	importMgr := NewEnhancedImportManager("")

	// 创建解析器适配器
	app.interfaceParser = NewInterfaceParserAdapter(importMgr)
	app.swaggerGenerator = NewSwaggerGeneratorAdapter(nil)
	app.ginGenerator = NewGinGeneratorAdapter(nil)
}

// Run 运行应用程序主逻辑
func (app *SwagGenApplication) Run() error {
	app.logger.Info("开始执行 swagGen...")

	// 验证配置
	if err := app.config.Validate(); err != nil {
		return NewValidationError("配置验证失败", err.Error())
	}

	// 解析接口
	collection, err := app.parseInterfaces()
	if err != nil {
		return err
	}

	// 过滤接口
	if err := app.filterInterfaces(collection); err != nil {
		return err
	}

	// 验证接口
	if err := app.validateInterfaces(collection); err != nil {
		return err
	}

	// 生成代码
	output, err := app.generateCode(collection)
	if err != nil {
		return err
	}

	// 写入文件
	if err := app.writeOutput(output); err != nil {
		return err
	}

	app.logger.Info("swagGen 执行完成")
	return nil
}

// parseInterfaces 解析接口定义
func (app *SwagGenApplication) parseInterfaces() (*InterfaceCollection, error) {
	app.logger.Info("开始解析接口...")

	var collection *InterfaceCollection
	var err error

	// 检查路径类型
	if app.fileSystem.IsDir(app.config.Path) {
		app.logger.Debug("解析目录: %s", app.config.Path)
		collection, err = app.interfaceParser.ParseDirectory(app.config.Path)
	} else {
		app.logger.Debug("解析文件: %s", app.config.Path)
		collection, err = app.interfaceParser.ParseFile(app.config.Path)
	}

	if err != nil {
		return nil, NewParseError("接口解析失败", "", err)
	}

	app.logger.Info("解析完成，找到 %d 个接口", len(collection.Interfaces))
	return collection, nil
}

// filterInterfaces 过滤接口
func (app *SwagGenApplication) filterInterfaces(collection *InterfaceCollection) error {
	if len(app.config.Interfaces) == 0 {
		return nil
	}

	app.logger.Debug("过滤接口: %v", app.config.Interfaces)

	originalCount := len(collection.Interfaces)
	collection.Interfaces = app.filterInterfacesByNames(collection.Interfaces, app.config.Interfaces)

	app.logger.Info("接口过滤完成: %d -> %d", originalCount, len(collection.Interfaces))
	return nil
}

// filterInterfacesByNames 根据名称过滤接口
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

// validateInterfaces 验证接口的有效性
func (app *SwagGenApplication) validateInterfaces(collection *InterfaceCollection) error {
	if len(collection.Interfaces) == 0 {
		return NewValidationError("未找到有效接口", "请检查接口定义是否包含正确的 Swagger 注释")
	}

	app.logger.Debug("验证 %d 个接口", len(collection.Interfaces))

	for _, iface := range collection.Interfaces {
		if len(iface.Methods) == 0 {
			app.logger.Warn("接口 %s 没有包含任何方法", iface.Name)
			continue
		}

		app.logger.Debug("  - %s (%d 个方法)", iface.Name, len(iface.Methods))
	}

	return nil
}

// generateCode 生成完整代码
func (app *SwagGenApplication) generateCode(collection *InterfaceCollection) (string, error) {
	app.logger.Info("开始生成代码...")

	// 设置包路径
	packagePath := app.getPackagePath()
	collection.ImportMgr.packagePath = packagePath

	// 设置生成器的接口集合
	app.swaggerGenerator.SetInterfaces(collection)
	app.ginGenerator.SetInterfaces(collection)

	var parts []string

	// 生成文件头部
	header := app.swaggerGenerator.GenerateFileHeader(app.inferPackageName())
	parts = append(parts, header)

	// 标记使用的包
	app.markUsedPackages(collection)

	// 生成导入声明
	imports := app.swaggerGenerator.GenerateImports()
	if imports != "" {
		parts = append(parts, imports, "")
	}

	// 生成类型引用
	if !app.config.SkipTypeReference {
		typeRefs, err := app.generateTypeReferences(collection)
		if err != nil {
			return "", NewGenerateError("类型引用生成失败", "", err)
		}
		if typeRefs != "" {
			parts = append(parts, typeRefs, "")
		}
	}

	// 生成 Swagger 注释
	swaggerComments, err := app.swaggerGenerator.GenerateSwaggerComments()
	if err != nil {
		return "", NewGenerateError("Swagger 注释生成失败", "", err)
	}

	// 生成 Gin 绑定代码
	ginCode, err := app.ginGenerator.GenerateComplete(swaggerComments)
	if err != nil {
		return "", NewGenerateError("Gin 代码生成失败", "", err)
	}

	if ginCode != "" {
		parts = append(parts, ginCode)
	}

	result := strings.Join(parts, "\n")
	app.logger.Info("代码生成完成，总长度: %d 字符", len(result))

	return result, nil
}

// markUsedPackages 标记使用的包
func (app *SwagGenApplication) markUsedPackages(collection *InterfaceCollection) {
	for _, iface := range collection.Interfaces {
		for _, method := range iface.Methods {
			// 标记返回类型使用的包
			packages := parsers.ExtractPackages(method.ResponseType.FullName)
			for _, pkgName := range packages {
				app.markPackageAsUsed(collection.ImportMgr, pkgName)
			}

			// 标记参数类型使用的包
			for _, param := range method.Parameters {
				paramPackages := parsers.ExtractPackages(param.Type.FullName)
				for _, pkgName := range paramPackages {
					app.markPackageAsUsed(collection.ImportMgr, pkgName)
				}
			}
		}
	}
}

// markPackageAsUsed 标记包为已使用
func (app *SwagGenApplication) markPackageAsUsed(importMgr *EnhancedImportManager, pkgName string) {
	for _, info := range importMgr.imports {
		parts := strings.Split(info.Path, "/")
		if len(parts) > 0 && pkgName == parts[len(parts)-1] {
			info.DirectlyUsed = true
		}
	}
}

// generateTypeReferences 生成类型引用
func (app *SwagGenApplication) generateTypeReferences(collection *InterfaceCollection) (string, error) {
	// 使用原有的逻辑生成类型引用
	// 这个功能暂时使用简化实现，需要查看原有的 SwaggerGenerator 中的实现
	var refs []string

	// 收集所有需要引用的类型
	typeSet := make(map[string]bool)

	for _, iface := range collection.Interfaces {
		for _, method := range iface.Methods {
			// 添加返回类型的引用
			if method.ResponseType.FullName != "" && method.ResponseType.Package != "" {
				typeDef := fmt.Sprintf("var _ %s", method.ResponseType.FullName)
				typeSet[typeDef] = true
			}

			// 添加参数类型的引用
			for _, param := range method.Parameters {
				if param.Type.FullName != "" && param.Type.Package != "" {
					typeDef := fmt.Sprintf("var _ %s", param.Type.FullName)
					typeSet[typeDef] = true
				}
			}
		}
	}

	// 转换为切片并排序
	for typeDef := range typeSet {
		refs = append(refs, typeDef)
	}

	if len(refs) > 0 {
		result := TypeReferenceComment + "\n" + strings.Join(refs, "\n")
		return result, nil
	}

	return "", nil
}

// writeOutput 写入输出文件
func (app *SwagGenApplication) writeOutput(output string) error {
	app.logger.Info("开始写入输出文件...")

	// 确定输出路径
	outputPath := app.determineOutputPath()

	// 格式化代码
	formattedBytes, err := gofmt.FormatBytes([]byte(output))
	if err != nil {
		return NewGenerateError("代码格式化失败", "", err)
	}

	// 写入文件
	if err := app.fileSystem.WriteFile(outputPath, formattedBytes, 0644); err != nil {
		return NewFileError("写入文件失败", outputPath, err)
	}

	app.logger.Info("成功生成文件: %s", outputPath)
	return nil
}

// determineOutputPath 确定输出文件路径
func (app *SwagGenApplication) determineOutputPath() string {
	outputPath := app.config.OutputFile

	// 如果输出路径已经是绝对路径，直接使用
	if filepath.IsAbs(outputPath) {
		return outputPath
	}

	// 如果输出路径是相对路径，需要根据输入路径确定基础目录
	var baseDir string
	if app.fileSystem.IsDir(app.config.Path) {
		// 输入是目录，输出文件放在该目录下
		baseDir = app.config.Path
	} else {
		// 输入是文件，输出文件放在该文件所在目录
		baseDir = filepath.Dir(app.config.Path)
	}

	// 构建最终的输出路径
	return filepath.Join(baseDir, outputPath)
}

// getPackagePath 获取包路径
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

// inferPackageName 推断包名
func (app *SwagGenApplication) inferPackageName() string {
	if app.config.Package != "" {
		return app.config.Package
	}

	// 从文件或目录推断包名
	if !app.fileSystem.IsDir(app.config.Path) {
		// 如果是文件，解析文件获取包名
		if pkgName := app.extractPackageNameFromFile(app.config.Path); pkgName != "" {
			return pkgName
		}
		// 使用文件所在目录名
		return filepath.Base(filepath.Dir(app.config.Path))
	}

	// 如果是目录，尝试从目录中的 Go 文件获取包名
	if pkgName := app.extractPackageNameFromDir(app.config.Path); pkgName != "" {
		return pkgName
	}

	// 使用目录名作为包名
	return filepath.Base(app.config.Path)
}

// extractPackageNameFromFile 从文件中提取包名
func (app *SwagGenApplication) extractPackageNameFromFile(filename string) string {
	content, err := app.fileSystem.ReadFile(filename)
	if err != nil {
		return ""
	}

	// 这里需要实现实际的包名提取逻辑
	// 暂时使用简单的字符串匹配
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

// extractPackageNameFromDir 从目录中提取包名
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

// SetLogger 设置日志记录器
func (app *SwagGenApplication) SetLogger(logger LoggerInterface) {
	app.logger = logger
}

// SetFileSystem 设置文件系统接口
func (app *SwagGenApplication) SetFileSystem(fs FileSystemInterface) {
	app.fileSystem = fs
}
