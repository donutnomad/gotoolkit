package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/donutnomad/gotoolkit/automap/parser"
)

func main() {
	// 定义命令行参数
	var (
		funcName    = flag.String("func", "", "映射函数名称 (必需)")
		packagePath = flag.String("package", "", "包路径 (可选)")
		output      = flag.String("output", "", "输出文件路径 (可选，默认打印到控制台)")
		verbose     = flag.Bool("verbose", false, "显示详细信息")
		validate    = flag.Bool("validate", false, "只验证不生成代码")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "AutoMap - Go结构体映射代码生成器\n\n")
		fmt.Fprintf(os.Stderr, "使用方法:\n")
		fmt.Fprintf(os.Stderr, "  automap -func <函数名> [选项]\n\n")
		fmt.Fprintf(os.Stderr, "支持的函数格式:\n")
		fmt.Fprintf(os.Stderr, "  1. Func(a *A) *B              => -func Func\n")
		fmt.Fprintf(os.Stderr, "  2. Func(a *A) (*B, error)     => -func Func\n")
		fmt.Fprintf(os.Stderr, "  3. (x *X) Func(a *A) *B       => -func X.Func\n")
		fmt.Fprintf(os.Stderr, "  4. (x *X) Func(a *A) (*B, error) => -func X.Func\n\n")
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  automap -func MapAToB\n")
		fmt.Fprintf(os.Stderr, "  automap -func X.MapAToB -verbose\n")
		fmt.Fprintf(os.Stderr, "  automap -func MapAToB -output generated.go\n")
		fmt.Fprintf(os.Stderr, "  automap -func MapAToB -validate\n")
	}

	flag.Parse()

	// 验证必需参数
	if *funcName == "" {
		fmt.Fprintf(os.Stderr, "错误: 必须指定函数名称\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// 创建解析器
	autoParser := parser.NewAutoMapParser(*packagePath)

	// 解析函数
	fmt.Printf("正在解析函数: %s\n", *funcName)
	result, err := autoParser.Parse(*funcName)
	if err != nil {
		log.Fatalf("解析失败: %v", err)
	}

	// 验证结果
	if err := parser.ValidateResult(result); err != nil {
		log.Fatalf("验证失败: %v", err)
	}

	fmt.Printf("解析成功！\n")

	// 如果只需要验证，显示摘要信息
	if *validate {
		fmt.Println("\n=== 验证结果 ===")
		fmt.Println(parser.GetMappingSummary(result))
		fmt.Println("\n函数签名验证通过，可以生成映射代码。")
		return
	}

	// 显示详细信息
	if *verbose {
		fmt.Println("\n=== 详细信息 ===")
		fmt.Println(parser.GetMappingSummary(result))

		fmt.Println("\n=== 生成的代码 ===")
		fmt.Println(result.GeneratedCode)
	}

	// 输出代码
	if *output != "" {
		// 写入文件
		if err := os.WriteFile(*output, []byte(result.GeneratedCode), 0644); err != nil {
			log.Fatalf("写入文件失败: %v", err)
		}
		fmt.Printf("\n代码已写入: %s\n", *output)
	} else {
		// 打印到控制台
		fmt.Println("\n=== 生成的代码 ===")
		fmt.Println(result.GeneratedCode)
	}
}

// 辅助函数，用于显示帮助信息
func printHelp() {
	fmt.Println("AutoMap - Go结构体映射代码生成器")
	fmt.Println()
	fmt.Println("这个工具可以自动分析Go映射函数并生成相应的字段映射代码。")
	fmt.Println()
	fmt.Println("主要功能:")
	fmt.Println("  - 自动解析函数签名（支持4种格式）")
	fmt.Println("  - 分析A/B类型的字段结构")
	fmt.Println("  - 识别字段映射关系（一对一、一对多、多对一）")
	fmt.Println("  - 处理GORM标签和JSONType字段")
	fmt.Println("  - 生成完整的映射逻辑代码")
	fmt.Println()
	fmt.Println("要求:")
	fmt.Println("  - A类型必须有ExportPatch()方法")
	fmt.Println("  - 映射函数必须遵循指定的签名格式")
	fmt.Println("  - 类型定义必须包含GORM标签（可选）")
	fmt.Println()
	fmt.Println("使用步骤:")
	fmt.Println("  1. 定义映射函数 MapAToB(a *A) *B")
	fmt.Println("  2. 确保A类型有ExportPatch方法")
	fmt.Println("  3. 运行: automap -func MapAToB")
	fmt.Println("  4. 生成的代码将自动处理字段映射逻辑")
}
