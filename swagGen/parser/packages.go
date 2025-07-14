package parsers

import "regexp"

// ExtractPackages 函数用于从输入字符串中提取所有 "包名"
// 它会返回一个去重后的字符串切片
func ExtractPackages(input string) []string {
	// 1. 定义正则表达式
	// (\w+) 匹配并捕获一个或多个单词字符（字母、数字、下划线）
	// \.   匹配后面紧跟着的点
	re := regexp.MustCompile(`(\w+)\.`)

	// 2. 查找所有匹配项及其子匹配（捕获组）
	// -1 表示查找所有匹配项，而不是只找第一个
	// 返回值是一个二维切片，例如：
	// 对于 "service.Base" 会返回 [["service.", "service"]]
	// 其中 "service." 是整个匹配项，"service" 是第一个捕获组的内容
	allMatches := re.FindAllStringSubmatch(input, -1)

	// 用于去重的map，key是包名，value是空结构体（节省空间）
	seen := make(map[string]struct{})
	// 最终的结果切片
	var result []string

	// 3. 遍历所有匹配项，提取捕获组
	for _, match := range allMatches {
		// `match` 是一个切片，例如 ["service.", "service"]
		// 我们需要的是索引为 1 的元素，即第一个捕获组的内容
		// `len(match) > 1` 是一个安全检查，确保捕获组存在
		if len(match) > 1 {
			packageName := match[1]
			// 4. 使用 map 进行去重
			if _, ok := seen[packageName]; !ok {
				seen[packageName] = struct{}{}
				result = append(result, packageName)
			}
		}
	}

	return result
}
