package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/donutnomad/gotoolkit/automap"
)

func main() {
	//if len(os.Args) < 3 {
	//	fmt.Println("用法: automap-cli <文件路径> <函数名>")
	//	fmt.Println("示例: automap-cli ./mod.go MapAToB")
	//	fmt.Println("      automap-cli /path/to/your/file.go YourFunction")
	//	os.Exit(1)
	//}
	filePath := "/Users/ubuntu/Projects/go/work/taas-backend/internal/app/launchpad/infra/persistence/listingrepo/mapper.go"
	funcName := "ListingPO.ToPO"

	//filePath := os.Args[1]
	//funcName := os.Args[2]

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Fatalf("文件不存在: %s", filePath)
	}

	// 转换为绝对路径
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatalf("获取绝对路径失败: %v", err)
	}

	// 改变工作目录到文件所在目录
	fileDir := filepath.Dir(absPath)
	if err := os.Chdir(fileDir); err != nil {
		log.Fatalf("切换目录失败: %v", err)
	}

	fmt.Printf("解析文件: %s\n", absPath)
	fmt.Printf("函数名: %s\n\n", funcName)

	// 解析映射函数
	result, err := automap.ParseWithOptions(funcName, automap.WithFileContext(absPath))
	if err != nil {
		log.Fatalf("解析失败: %v", err)
	}

	// 显示解析结果
	fmt.Printf("=== 解析结果 ===\n")
	fmt.Printf("函数名: %s\n", result.FuncSignature.FuncName)
	fmt.Printf("输入类型: %s\n", result.AType.Name)
	fmt.Printf("输出类型: %s\n", result.BType.Name)
	fmt.Printf("是否有ExportPatch: %t\n", result.HasExportPatch)
	fmt.Printf("字段映射数量: 一对一(%d), 一对多(%d), JSON字段(%d)\n\n",
		len(result.FieldMapping.OneToOne),
		len(result.FieldMapping.OneToMany),
		len(result.FieldMapping.JSONFields))

	//// 生成完整代码
	fmt.Printf("=== 生成的代码 ===\n")
	code, err := automap.ParseAndGenerate(funcName, automap.WithFileContext(absPath))
	if err != nil {
		log.Fatalf("生成代码失败: %v", err)
	}

	fmt.Print(code)
}
