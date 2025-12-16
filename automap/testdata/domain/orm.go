package domain

import "time"

// Model GORM基础模型（模拟外部ORM包的Model）
type Model struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}
