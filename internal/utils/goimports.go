package utils

import (
	"fmt"
	"os"

	"golang.org/x/tools/imports"
)

func WriteFormat(fileName string, src []byte) error {
	bs, err := imports.Process(fileName, src, &imports.Options{
		Fragment:   true,
		AllErrors:  true,
		Comments:   true,
		TabIndent:  true,
		TabWidth:   8,
		FormatOnly: false,
	})
	if err != nil {
		fmt.Println("format file failed:", string(src))
		return err
	}
	// 输出到文件中
	return os.WriteFile(fileName, bs, 0644)
}
