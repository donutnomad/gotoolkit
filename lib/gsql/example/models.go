package example

//go:generate ../../../gormgen/gormgen -dir . -struct User,Order

import (
	"time"
)

// User 示例模型
type User struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	OrgID     uint64    `gorm:"column:org_id;index:idx_users_orgid"`
	Status    string    `gorm:"column:status;index:idx_users_status"`
	CreatedAt time.Time `gorm:"column:created_at;index:idx_users_created_at"`
}

func (User) TableName() string { return "users" }

// Order 示例模型
type Order struct {
	ID        uint64    `gorm:"column:id;primaryKey"`
	UserID    uint64    `gorm:"column:user_id;index:idx_orders_userid"`
	Amount    int64     `gorm:"column:amount"`
	CreatedAt time.Time `gorm:"column:created_at;index:idx_orders_created_at"`
}

func (Order) TableName() string { return "orders" }
