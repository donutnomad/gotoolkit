package testdata

import "time"

// CrossFilePO 跨文件测试 - 结构体定义
// ToPO 函数在 cross_file_mapper.go 中
type CrossFilePO struct {
	Model           // 嵌入字段
	Username string `gorm:"column:username"`
	Email    string `gorm:"column:email"`
	Score    int    `gorm:"column:score"`
}

// CrossFileDomain 跨文件测试 - Domain 类型
type CrossFileDomain struct {
	ID        uint64
	CreatedAt time.Time
	UpdatedAt time.Time
	Username  string
	Email     string
	Score     int
}
