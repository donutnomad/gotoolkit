package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var dir = flag.String("dir", ".", "目录路径")
	var structNames = flag.String("struct", "", "结构体名称,多个使用逗号分隔")
	var clean = flag.Bool("clean", false, "清理未使用的setter方法")
	var patch = flag.Bool("patch", false, "生成GORM Patch结构体和Build方法")
	var fields = flag.Bool("fields", false, "生成字段常量结构体")
	var debug = flag.Bool("debug", false, "调试模式，打印解析结果")
	flag.Parse()

	if *structNames == "" {
		log.Fatal("请指定结构体名称，使用 -struct 参数")
	}

	// 解析结构体名称列表
	structList := strings.Split(*structNames, ",")
	for i := range structList {
		structList[i] = strings.TrimSpace(structList[i])
	}

	// 按文件分组收集结构体信息
	type StructData struct {
		StructInfo *StructInfo
		GormModel  *GormModelInfo
	}
	fileToStructs := make(map[string][]StructData)

	// 处理每个结构体
	for _, structName := range structList {
		if structName == "" {
			continue
		}

		// 查找包含指定结构体的文件
		files, err := findGoFiles(*dir)
		if err != nil {
			log.Fatalf("查找Go文件失败: %v", err)
		}

		var targetFile string
		for _, file := range files {
			if containsStruct(file, structName) {
				targetFile = file
				break
			}
		}

		if targetFile == "" {
			log.Fatalf("在目录 %s 中未找到包含结构体 %s 的文件", *dir, structName)
		}

		fmt.Printf("找到结构体 %s 在文件: %s\n", structName, targetFile)

		// 解析结构体
		structInfo, err := parseStruct(targetFile, structName)
		if err != nil {
			log.Fatalf("解析结构体 %s 失败: %v", structName, err)
		}

		if *debug {
			fmt.Printf("\n=== 调试信息 ===\n")
			fmt.Printf("结构体名称: %s\n", structInfo.Name)
			fmt.Printf("包名: %s\n", structInfo.PackageName)
			fmt.Printf("字段数量: %d\n", len(structInfo.Fields))

			for i, field := range structInfo.Fields {
				fmt.Printf("字段 %d: %s (%s) - Tag: %s\n", i+1, field.Name, field.Type, field.Tag)
			}
			fmt.Printf("=== 调试信息结束 ===\n\n")
		}

		// 收集结构体信息，按文件分组
		data := StructData{
			StructInfo: structInfo,
		}
		if *patch || *fields {
			data.GormModel = parseGormModel(structInfo)
		}
		fileToStructs[targetFile] = append(fileToStructs[targetFile], data)
	}

	// 按文件生成输出
	for targetFile, structs := range fileToStructs {
		if *patch {
			// GORM Patch生成模式
			outputFile := strings.TrimSuffix(targetFile, ".go") + "_patch.go"
			var gormModels []*GormModelInfo
			for _, data := range structs {
				gormModels = append(gormModels, data.GormModel)
			}
			err := generateGormPatchFileForMultiple(outputFile, gormModels)
			if err != nil {
				log.Fatalf("生成GORM patch文件失败: %v", err)
			}
			fmt.Printf("成功生成GORM patch文件: %s (包含 %d 个结构体)\n", outputFile, len(gormModels))
		} else if *fields {
			// 字段常量生成模式
			outputFile := strings.TrimSuffix(targetFile, ".go") + "_fields.go"
			var gormModels []*GormModelInfo
			for _, data := range structs {
				gormModels = append(gormModels, data.GormModel)
			}
			err := generateFieldsFileForMultiple(outputFile, gormModels)
			if err != nil {
				log.Fatalf("生成字段常量文件失败: %v", err)
			}
			fmt.Printf("成功生成字段常量文件: %s (包含 %d 个结构体)\n", outputFile, len(gormModels))
		} else {
			// 原有的setter生成模式
			outputFile := strings.TrimSuffix(targetFile, ".go") + "_setter.go"
			var structInfos []*StructInfo
			for _, data := range structs {
				structInfos = append(structInfos, data.StructInfo)
			}
			err := generateSetterFileForMultiple(outputFile, structInfos, *clean)
			if err != nil {
				log.Fatalf("生成setter文件失败: %v", err)
			}
			fmt.Printf("成功生成setter文件: %s (包含 %d 个结构体)\n", outputFile, len(structInfos))
		}
	}
}

// findGoFiles 查找目录中的所有Go文件
func findGoFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// containsStruct 检查文件是否包含指定的结构体
func containsStruct(filename, structName string) bool {
	content, err := os.ReadFile(filename)
	if err != nil {
		return false
	}

	// 简单的字符串匹配，检查是否包含 "type StructName struct"
	return strings.Contains(string(content), fmt.Sprintf("type %s struct", structName))
}
