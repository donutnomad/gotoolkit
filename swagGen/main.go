package main

import (
	"flag"
	"fmt"
	"github.com/donutnomad/gotoolkit/swagGen/gofmt"
	parsers "github.com/donutnomad/gotoolkit/swagGen/parser"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

var (
	path        = flag.String("path", "", "目录路径或文件路径")
	outputFile  = flag.String("out", "swagger_generated.go", "输出文件名")
	packageName = flag.String("package", "", "包名（可选，默认从文件推断）")
	interfaces  = flag.String("interfaces", "", "要处理的接口名称，逗号分隔（可选，默认处理所有带注释的接口）")
	verbose     = flag.Bool("v", false, "详细输出")
)

func main() {
	flag.Parse()

	if *path == "" {
		fmt.Println("错误: 必须指定 -path 参数")
		flag.Usage()
		os.Exit(1)
	}

	if err := run(); err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 创建增强的导入管理器
	importMgr := NewEnhancedImportManager("") // 包路径稍后设置

	// 创建接口解析器
	interfaceParser := NewInterfaceParser(importMgr)

	// 解析接口
	var collection *InterfaceCollection
	var err error

	// 检查路径是文件还是目录
	fileInfo, err := os.Stat(*path)
	if err != nil {
		return fmt.Errorf("无法访问路径 %s: %v", *path, err)
	}

	if fileInfo.IsDir() {
		if *verbose {
			fmt.Printf("正在解析目录: %s\n", *path)
		}
		collection, err = interfaceParser.ParseDirectory(*path)
	} else {
		if *verbose {
			fmt.Printf("正在解析文件: %s\n", *path)
		}
		collection, err = interfaceParser.ParseFile(*path)
	}

	if err != nil {
		return fmt.Errorf("解析失败: %v", err)
	}

	// 过滤接口（如果指定了接口名称）
	if *interfaces != "" {
		interfaceNames := strings.Split(*interfaces, ",")
		for i, name := range interfaceNames {
			interfaceNames[i] = strings.TrimSpace(name)
		}
		collection = collection.FilterInterfacesByName(interfaceNames)

		if *verbose {
			fmt.Printf("过滤接口: %v\n", interfaceNames)
		}
	}

	// 检查是否找到接口
	if len(collection.Interfaces) == 0 {
		return fmt.Errorf("未找到任何带有 Swagger 注释的接口")
	}

	if *verbose {
		fmt.Printf("找到 %d 个接口:\n", len(collection.Interfaces))
		for _, iface := range collection.Interfaces {
			fmt.Printf("  - %s (%d 个方法)\n", iface.Name, len(iface.Methods))
		}
	}

	// 推断包名
	pkgName := *packageName
	if pkgName == "" {
		pkgName = inferPackageName(*path)
	}

	// 设置导入管理器的包路径
	collection.ImportMgr.packagePath = getPackagePathFromDir(*path)

	// 生成代码
	output, err := generateCode(collection, pkgName)
	if err != nil {
		return fmt.Errorf("生成代码失败: %v", err)
	}

	// 确定输出路径
	outputPath := *outputFile
	if !filepath.IsAbs(outputPath) {
		if fileInfo.IsDir() {
			outputPath = filepath.Join(*path, *outputFile)
		} else {
			outputPath = filepath.Join(filepath.Dir(*path), *outputFile)
		}
	}

	bytes, err := gofmt.FormatBytes([]byte(output))
	if err != nil {
		panic(err)
	}

	// 写入文件
	if err := os.WriteFile(outputPath, bytes, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fmt.Printf("成功生成文件: %s\n", outputPath)
	return nil
}

// generateCode 生成完整的代码
func generateCode(collection *InterfaceCollection, packageName string) (string, error) {
	var parts []string

	// 创建 Swagger 生成器
	swaggerGen := NewSwaggerGenerator(collection)

	// 创建 Gin 生成器
	ginGen := NewGinGenerator(collection)

	// 生成文件头部
	header := swaggerGen.GenerateFileHeader(packageName)
	parts = append(parts, header)

	// 生成导入声明
	for _, iface := range ginGen.collection.Interfaces {
		for _, method := range iface.Methods {
			for _, pkgName := range parsers.ExtractPackages(method.ResponseType.FullName) {
				for _, info := range swaggerGen.collection.ImportMgr.imports {
					parts2 := strings.Split(info.Path, "/")
					if len(parts2) > 0 {
						if pkgName == parts2[len(parts2)-1] {
							info.DirectlyUsed = true
						}
					}
				}
			}
		}
	}
	imports := swaggerGen.GenerateImports()
	if imports != "" {
		parts = append(parts, imports)
		parts = append(parts, "")
	}

	// 生成类型引用
	typeRefs := swaggerGen.GenerateTypeReferences()
	if typeRefs != "" {
		parts = append(parts, typeRefs)
		parts = append(parts, "")
	}

	// 生成 Swagger 注释
	swaggerCommentsMap := swaggerGen.GenerateSwaggerComments()

	// 生成 Gin 绑定代码
	ginCode := ginGen.GenerateComplete(swaggerCommentsMap)
	if ginCode != "" {
		parts = append(parts, ginCode)
	}

	return strings.Join(parts, "\n"), nil
}

// inferPackageName 推断包名
func inferPackageName(path string) string {
	// 如果是文件，直接解析文件内容
	if fileInfo, err := os.Stat(path); err == nil && !fileInfo.IsDir() {
		// 解析文件获取包名
		if pkgName := extractPackageNameFromFile(path); pkgName != "" {
			return pkgName
		}
		// 如果解析失败，使用文件所在目录名
		path = filepath.Dir(path)
	} else {
		// 如果是目录，尝试找到其中的 Go 文件并解析包名
		if pkgName := extractPackageNameFromDir(path); pkgName != "" {
			return pkgName
		}
	}

	// 使用目录名作为包名
	return filepath.Base(path)
}

// extractPackageNameFromFile 从单个文件中提取包名
func extractPackageNameFromFile(filename string) string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.PackageClauseOnly)
	if err != nil {
		return ""
	}

	if file.Name != nil {
		return file.Name.Name
	}

	return ""
}

// extractPackageNameFromDir 从目录中的第一个 Go 文件提取包名
func extractPackageNameFromDir(dir string) string {
	files, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if strings.HasSuffix(file.Name(), ".go") && !strings.HasSuffix(file.Name(), "_test.go") {
			filename := filepath.Join(dir, file.Name())
			if pkgName := extractPackageNameFromFile(filename); pkgName != "" {
				return pkgName
			}
		}
	}

	return ""
}

// getPackagePathFromDir 从目录获取包路径
func getPackagePathFromDir(path string) string {
	// 简化处理，实际项目中应该使用 go/packages 来获取完整的包路径
	if fileInfo, err := os.Stat(path); err == nil && !fileInfo.IsDir() {
		path = filepath.Dir(path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return filepath.Base(path)
	}

	return absPath
}

// printUsage 打印使用说明
func printUsage() {
	fmt.Printf(`swagGen - Swagger 文档和 Gin 绑定代码生成器

用法:
  %s [选项]

选项:
  -path string
        目录路径或文件路径（必需）
  -out string
        输出文件名 (默认 "swagger_generated.go")
  -package string
        包名（可选，默认从文件推断）
  -interfaces string
        要处理的接口名称，逗号分隔（可选，默认处理所有带注释的接口）
  -v    详细输出

示例:
  %s -path ./api -out swagger.go
  %s -path ./api/user.go -interfaces "IUserAPI,IAdminAPI"
  %s -path ./api -package myapi -v

支持的注释:

  路由注释:
    @GET(/api/v1/user/{id})    - GET 请求
    @POST(/api/v1/user)        - POST 请求
    @PUT(/api/v1/user/{id})    - PUT 请求
    @PATCH(/api/v1/user/{id})  - PATCH 请求
    @DELETE(/api/v1/user/{id}) - DELETE 请求

  参数注释:
    @PARAM                     - 路径参数（自动推断）
    @PARAM(alias)              - 带别名的路径参数
    @QUERY                     - 查询参数
    @HEADER                    - 头部参数
    @BODY                      - 请求体参数
    @FORM                      - 表单参数

  请求内容类型:
    @JSON-REQ                  - JSON 请求
    @FORM-REQ                  - 表单请求
    @MIME-REQ(content-type)    - 自定义请求类型

  响应内容类型:
    @JSON                      - JSON 响应
    @MIME(content-type)        - 自定义响应类型

  接口级别注释:
    @TAG(tag1,tag2)            - 为所有方法添加标签
    @SECURITY(auth)            - 为所有方法添加安全认证
    @HEADER(X-Token,true,"说明") - 为所有方法添加头部参数

  中间件注释:
    @MID(auth,log)             - 为方法添加中间件

  控制注释:
    @Removed                   - 移除方法（不生成代码）
    @ExcludeFromBindAll        - 排除在 BindAll 方法之外

  排除语法:
    @TAG(Company;exclude="StartTransfer")     - 为所有方法添加标签，但排除 StartTransfer
    @SECURITY(ApiKeyAuth;exclude="method1,method2") - 为所有方法添加安全认证，但排除指定方法

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func init() {
	flag.Usage = printUsage
}
