package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/donutnomad/gotoolkit/internal/gormparse"
	"github.com/donutnomad/gotoolkit/internal/structparse"
)

func main() {
	var dir = flag.String("dir", ".", "目录路径")
	var structNames = flag.String("struct", "", "结构体名称,多个使用逗号分隔")
	var prefix = flag.String("prefix", "", "生成的结构体前缀")
	flag.Parse()

	if *structNames == "" {
		log.Fatal("请指定结构体名称,使用 -struct 参数")
	}

	// 解析结构体名称列表
	structList := strings.Split(*structNames, ",")
	for i := range structList {
		structList[i] = strings.TrimSpace(structList[i])
	}

	// 收集所有要生成的模型
	var allModels []*gormparse.GormModelInfo
	outputFile := ""

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
		structInfo, err := structparse.ParseStruct(targetFile, structName)
		if err != nil {
			log.Fatalf("解析结构体 %s 失败: %v", structName, err)
		}

		// 推导表名
		tableName, err := inferTableName(targetFile, structName)
		if err != nil {
			log.Fatalf("推导表名失败: %v", err)
		}

		// 转换为GORM模型
		si := &gormparse.StructInfo{
			Name:        structInfo.Name,
			PackageName: structInfo.PackageName,
			Imports:     structInfo.Imports,
		}
		for _, f := range structInfo.Fields {
			si.Fields = append(si.Fields, gormparse.FieldInfo{
				Name:       f.Name,
				Type:       f.Type,
				PkgPath:    f.PkgPath,
				Tag:        f.Tag,
				SourceType: f.SourceType,
			})
		}

		gormModel := gormparse.ParseGormModel(si)
		gormModel.TableName = tableName
		gormModel.Prefix = *prefix

		allModels = append(allModels, gormModel)

		// 使用第一个文件作为输出文件的基础
		if outputFile == "" {
			outputFile = strings.TrimSuffix(targetFile, ".go") + "_query.go"
		}
	}

	// 一次性生成所有代码到同一个文件
	if len(allModels) > 0 {
		err := generateGormQueryFileForMultiple(outputFile, allModels)
		if err != nil {
			log.Fatalf("生成查询文件失败: %v", err)
		}
		fmt.Printf("成功生成查询文件: %s (包含 %d 个结构体)\n", outputFile, len(allModels))
	}
}

// findGoFiles 查找目录中的所有Go文件
func findGoFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_query.go") {
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

	// 简单的字符串匹配,检查是否包含 "type StructName struct"
	return strings.Contains(string(content), fmt.Sprintf("type %s struct", structName))
}
