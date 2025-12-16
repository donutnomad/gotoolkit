package testdata

import (
	"github.com/donutnomad/gotoolkit/automap/testdata/domain"
)

// ExternalUserPO 使用外部包Domain的PO类型
type ExternalUserPO struct {
	ID    int64  `gorm:"column:id"`
	Name  string `gorm:"column:name"`
	Email string `gorm:"column:email"`
}

// ToPO 从外部包的Domain转换为PO
func (p *ExternalUserPO) ToPO(input *domain.ExternalUserDomain) *ExternalUserPO {
	return &ExternalUserPO{
		ID:    input.ID,
		Name:  input.Name,
		Email: input.Email,
	}
}

// ApprovalPO 使用外部包嵌入类型的PO
type ApprovalPO struct {
	domain.Model        // 嵌入外部包的Model
	Title        string `gorm:"column:title"`
	Status       int    `gorm:"column:status"`
}

// ToPO 从Domain转换为PO（包含外部嵌入类型）
func (p *ApprovalPO) ToPO(entity *domain.ApprovalDomain) *ApprovalPO {
	return &ApprovalPO{
		Model: domain.Model{
			ID:        entity.ID,
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
		},
		Title:  entity.Title,
		Status: entity.Status,
	}
}
