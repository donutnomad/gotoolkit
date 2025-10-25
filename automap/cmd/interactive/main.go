package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/donutnomad/gotoolkit/automap"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("=== AutoMap Go AST 解析器 ===")
	fmt.Println("输入 <文件路径> <函数名>（输入 'quit' 退出）:")
	fmt.Println("示例: ./mod.go MapAToB")

	for {
		fmt.Print("\n文件路径和函数名: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "quit" || input == "exit" {
			break
		}

		if input == "" {
			continue
		}

		// 分割文件路径和函数名
		parts := strings.Fields(input)
		if len(parts) < 2 {
			fmt.Println("❌ 请输入: <文件路径> <函数名>")
			continue
		}

		filePath := parts[0]
		funcName := parts[1]

		// 检查文件是否存在
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			fmt.Printf("❌ 文件不存在: %s\n", filePath)
			continue
		}

		// 转换为绝对路径并切换目录
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			fmt.Printf("❌ 获取绝对路径失败: %v\n", err)
			continue
		}

		fileDir := filepath.Dir(absPath)
		if err := os.Chdir(fileDir); err != nil {
			fmt.Printf("❌ 切换目录失败: %v\n", err)
			continue
		}

		// 解析映射函数
		result, err := automap.Parse(funcName)
		if err != nil {
			fmt.Printf("❌ 解析失败: %v\n", err)
			continue
		}

		// 显示解析结果
		fmt.Printf("\n=== 解析结果 ===\n")
		fmt.Printf("✅ 函数名: %s\n", result.FuncSignature.FuncName)
		fmt.Printf("✅ 输入类型: %s\n", result.AType.Name)
		fmt.Printf("✅ 输出类型: %s\n", result.BType.Name)
		fmt.Printf("✅ 是否有ExportPatch: %t\n", result.HasExportPatch)
		fmt.Printf("✅ 字段映射数量: 一对一(%d), 一对多(%d), JSON字段(%d)\n",
			len(result.FieldMapping.OneToOne),
			len(result.FieldMapping.OneToMany),
			len(result.FieldMapping.JSONFields))

		// 显示映射详情
		fmt.Printf("\n=== 映射详情 ===\n")
		if len(result.FieldMapping.OneToOne) > 0 {
			fmt.Println("一对一映射:")
			for aField, bField := range result.FieldMapping.OneToOne {
				fmt.Printf("  %s -> %s\n", aField, bField)
			}
		}

		if len(result.FieldMapping.OneToMany) > 0 {
			fmt.Println("一对多映射:")
			for aField, bFields := range result.FieldMapping.OneToMany {
				fmt.Printf("  %s -> %v\n", aField, bFields)
			}
		}

		if len(result.FieldMapping.JSONFields) > 0 {
			fmt.Println("JSON字段映射:")
			for bField, jsonMapping := range result.FieldMapping.JSONFields {
				fmt.Printf("  %s (%s):\n", bField, jsonMapping.FieldName)
				for aField, jsonField := range jsonMapping.SubFields {
					fmt.Printf("    %s -> %s\n", aField, jsonField)
				}
			}
		}

		// 询问是否显示生成的代码
		fmt.Print("\n是否显示生成的代码? (y/n): ")
		if !scanner.Scan() {
			break
		}

		if strings.ToLower(strings.TrimSpace(scanner.Text())) == "y" {
			fmt.Printf("\n=== 生成的代码 ===\n")
			code, err := automap.ParseAndGenerate(funcName)
			if err != nil {
				fmt.Printf("❌ 生成代码失败: %v\n", err)
				continue
			}
			fmt.Print(code)
		}
	}

	fmt.Println("\n👋 再见!")
}
