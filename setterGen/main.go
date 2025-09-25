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
	var structName = flag.String("struct", "", "结构体名称")
	var clean = flag.Bool("clean", false, "清理未使用的setter方法")
	var patch = flag.Bool("patch", false, "生成GORM Patch结构体和Build方法")
	var fields = flag.Bool("fields", false, "生成字段常量结构体")
	var debug = flag.Bool("debug", false, "调试模式，打印解析结果")
	flag.Parse()

	if *structName == "" {
		log.Fatal("请指定结构体名称，使用 -struct 参数")
	}

	// 查找包含指定结构体的文件
	files, err := findGoFiles(*dir)
	if err != nil {
		log.Fatalf("查找Go文件失败: %v", err)
	}

	var targetFile string
	for _, file := range files {
		if containsStruct(file, *structName) {
			targetFile = file
			break
		}
	}

	if targetFile == "" {
		log.Fatalf("在目录 %s 中未找到包含结构体 %s 的文件", *dir, *structName)
	}

	fmt.Printf("找到结构体 %s 在文件: %s\n", *structName, targetFile)

	// 解析结构体
	structInfo, err := parseStruct(targetFile, *structName)
	if err != nil {
		log.Fatalf("解析结构体失败: %v", err)
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

	if *patch {
		// GORM Patch生成模式
		gormModel := parseGormModel(structInfo)
		outputFile := strings.TrimSuffix(targetFile, ".go") + "_patch.go"
		err = generateGormPatchFile(outputFile, gormModel)
		if err != nil {
			log.Fatalf("生成GORM patch文件失败: %v", err)
		}
		fmt.Printf("成功生成GORM patch文件: %s\n", outputFile)
	} else if *fields {
		// 字段常量生成模式
		gormModel := parseGormModel(structInfo)
		outputFile := strings.TrimSuffix(targetFile, ".go") + "_fields.go"
		err = generateFieldsFile(outputFile, gormModel)
		if err != nil {
			log.Fatalf("生成字段常量文件失败: %v", err)
		}
		fmt.Printf("成功生成字段常量文件: %s\n", outputFile)
	} else {
		// 原有的setter生成模式
		outputFile := strings.TrimSuffix(targetFile, ".go") + "_setter.go"
		err = generateSetterFile(outputFile, structInfo, *clean)
		if err != nil {
			log.Fatalf("生成setter文件失败: %v", err)
		}
		fmt.Printf("成功生成setter文件: %s\n", outputFile)
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
