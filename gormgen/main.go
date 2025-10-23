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
	var outputDir = flag.String("out", "", "输出目录路径,支持$PROJECT_ROOT变量")
	var patch = flag.Bool("patch", false, "生成GORM Patch结构体和Build方法")
	var one = flag.Bool("one", false, "将query和patch代码生成到同一个文件中")
	var outputFile = flag.String("o", "", "指定输出文件名,所有内容输出到此文件(不按文件分组)")
	flag.Parse()

	if *structNames == "" {
		log.Fatal("[gormgen] 请指定结构体名称,使用 -struct 参数")
	}

	// 解析输出目录
	finalOutputDir := ""
	if *outputDir != "" {
		var err error
		finalOutputDir, err = resolveOutputDir(*outputDir)
		if err != nil {
			log.Fatalf("[gormgen] 解析输出目录失败: %v", err)
		}
	}

	// 解析结构体名称列表
	structList := strings.Split(*structNames, ",")
	for i := range structList {
		structList[i] = strings.TrimSpace(structList[i])
	}

	// 按照文件分组收集模型
	fileModelsMap := make(map[string][]*gormparse.GormModelInfo)
	fileOrderList := []string{} // 保持文件顺序

	// 处理每个结构体
	for _, structName := range structList {
		if structName == "" {
			continue
		}

		// 查找包含指定结构体的文件
		files, err := findGoFiles(*dir)
		if err != nil {
			log.Fatalf("[gormgen] 查找Go文件失败: %v", err)
		}

		var targetFile string
		for _, file := range files {
			if containsStruct(file, structName) {
				targetFile = file
				break
			}
		}

		if targetFile == "" {
			log.Fatalf("[gormgen] 在目录 %s 中未找到包含结构体 %s 的文件", *dir, structName)
		}

		fmt.Printf("[gormgen] 找到结构体 %s 在文件: %s\n", structName, targetFile)

		// 解析结构体
		structInfo, err := structparse.ParseStruct(targetFile, structName)
		if err != nil {
			log.Fatalf("[gormgen] 解析结构体 %s 失败: %v", structName, err)
		}

		// 推导表名
		tableName, err := inferTableName(targetFile, structName)
		if err != nil {
			log.Fatalf("[gormgen] 推导表名失败: %v", err)
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

		// 按文件分组
		if _, exists := fileModelsMap[targetFile]; !exists {
			fileOrderList = append(fileOrderList, targetFile)
		}
		fileModelsMap[targetFile] = append(fileModelsMap[targetFile], gormModel)
	}

	// 检查是否使用 -o 指定单一输出文件
	if *outputFile != "" {
		// 收集所有模型
		var allModels []*gormparse.GormModelInfo
		for _, targetFile := range fileOrderList {
			allModels = append(allModels, fileModelsMap[targetFile]...)
		}

		// 所有内容输出到指定文件
		if *one && *patch {
			// query和patch合并到一个文件
			err := generateGormQueryAndPatchFile(*outputFile, allModels)
			if err != nil {
				log.Fatalf("[gormgen] 生成合并文件失败: %v", err)
			}
			fmt.Printf("[gormgen] 成功生成 %s (包含 %d 个结构体, query+patch)\n", *outputFile, len(allModels))
		} else {
			// 只生成query
			err := generateGormQueryFileForMultiple(*outputFile, allModels)
			if err != nil {
				log.Fatalf("[gormgen] 生成查询文件失败: %v", err)
			}
			fmt.Printf("[gormgen] 成功生成 %s (包含 %d 个结构体)\n", *outputFile, len(allModels))

			// 如果需要patch，生成到单独文件
			if *patch {
				patchFile := (*outputFile)[:len(*outputFile)-3] + "_patch.go"
				err := generateGormPatchFileForMultiple(patchFile, allModels)
				if err != nil {
					log.Fatalf("[gormgen] 生成GORM patch文件失败: %v", err)
				}
				fmt.Printf("[gormgen] 成功生成 %s (包含 %d 个结构体)\n", patchFile, len(allModels))
			}
		}
	} else {
		// 按文件分组生成
		for _, targetFile := range fileOrderList {
			models := fileModelsMap[targetFile]

			// 确定输出文件路径
			var queryFile string
			if finalOutputDir != "" {
				// 使用指定的输出目录
				fileName := strings.TrimSuffix(filepath.Base(targetFile), ".go") + "_query.go"
				queryFile = filepath.Join(finalOutputDir, fileName)
			} else {
				// 使用原文件所在目录
				queryFile = strings.TrimSuffix(targetFile, ".go") + "_query.go"
			}

			// 检查是否合并到一个文件
			if *one && *patch {
				// query和patch合并到一个文件
				err := generateGormQueryAndPatchFile(queryFile, models)
				if err != nil {
					log.Fatalf("[gormgen] 生成合并文件失败: %v", err)
				}
				fmt.Printf("[gormgen] 成功生成 %s (包含 %d 个结构体, query+patch)\n", queryFile, len(models))
			} else {
				// 生成query文件
				err := generateGormQueryFileForMultiple(queryFile, models)
				if err != nil {
					log.Fatalf("[gormgen] 生成查询文件失败: %v", err)
				}
				fmt.Printf("[gormgen] 成功生成 %s (包含 %d 个结构体)\n", queryFile, len(models))

				// 生成patch文件
				if *patch {
					patchFile := queryFile[:len(queryFile)-9] + "_patch.go"
					err := generateGormPatchFileForMultiple(patchFile, models)
					if err != nil {
						log.Fatalf("[gormgen] 生成GORM patch文件失败: %v", err)
					}
					fmt.Printf("[gormgen] 成功生成 %s (包含 %d 个结构体)\n", patchFile, len(models))
				}
			}
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

// resolveOutputDir 解析输出目录，支持$PROJECT_ROOT变量
func resolveOutputDir(outputDir string) (string, error) {
	if !strings.Contains(outputDir, "$PROJECT_ROOT") {
		// 没有变量，直接返回
		return outputDir, nil
	}

	// 查找项目根目录
	projectRoot, err := findProjectRoot(".")
	if err != nil {
		return "", fmt.Errorf("[gormgen] 查找项目根目录失败: %v", err)
	}

	// 替换$PROJECT_ROOT变量
	resolvedDir := strings.ReplaceAll(outputDir, "$PROJECT_ROOT", projectRoot)

	// 确保目录存在
	if err := os.MkdirAll(resolvedDir, 0755); err != nil {
		return "", fmt.Errorf("[gormgen] 创建输出目录失败: %v", err)
	}

	return resolvedDir, nil
}

// findProjectRoot 向上递归查找包含go.mod的目录
func findProjectRoot(startDir string) (string, error) {
	currentDir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return currentDir, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// 已经到达根目录
			break
		}
		currentDir = parentDir
	}

	return "", fmt.Errorf("[gormgen] 未找到包含go.mod的目录")
}
