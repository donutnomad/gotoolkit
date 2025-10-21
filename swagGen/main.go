package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	path            = flag.String("path", "", "目录路径或文件路径")
	outputFile      = flag.String("out", "swagger_generated.go", "输出文件名")
	packageName     = flag.String("package", "", "包名（可选，默认从文件推断）")
	interfaces      = flag.String("interfaces", "", "要处理的接口名称，逗号分隔（可选，默认处理所有带注释的接口）")
	verbose         = flag.Bool("v", false, "详细输出")
	includeTypeRefs = flag.Bool("include-type-refs", false, "生成类型引用声明（var _ 声明）")
	version         = flag.Int("version", 2, "兼容的版本号")
	enableFormat    = flag.Bool("fmt", false, "启用代码格式化")
)

func main() {
	flag.Parse()

	// 创建配置
	config := createConfigFromFlags()

	// 创建应用程序实例
	app := NewSwagGenApplication(config)

	// 运行应用程序
	if err := app.Run(); err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
}

// createConfigFromFlags 从命令行参数创建配置
func createConfigFromFlags() *GenerationConfig {
	config := NewDefaultConfig()
	config.Path = *path
	config.OutputFile = *outputFile
	config.Package = *packageName
	config.Verbose = *verbose
	config.SkipTypeReference = !*includeTypeRefs // 如果用户要求包含类型引用，则不跳过
	config.EnableFormat = *enableFormat          // 设置是否启用格式化

	// 解析接口列表
	if *interfaces != "" {
		interfaceNames := strings.Split(*interfaces, ",")
		for i, name := range interfaceNames {
			interfaceNames[i] = strings.TrimSpace(name)
		}
		config.Interfaces = interfaceNames
	}

	// 验证配置
	if config.Path == "" {
		fmt.Println("错误: 必须指定 -path 参数")
		flag.Usage()
		os.Exit(1)
	}

	return config
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
  -include-type-refs
        生成类型引用声明（var _ 声明），用于确保 swaggo 识别类型
  -fmt
        启用代码格式化（swag fmt）
  -v    详细输出

示例:
  %s -path ./api -out swagger.go
  %s -path ./api/user.go -interfaces "IUserAPI,IAdminAPI"
  %s -path ./api -package myapi -v
  %s -path ./api -include-type-refs  # 包含类型引用
  %s -path ./api -fmt                # 启用格式化

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

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func init() {
	flag.Usage = printUsage
}
