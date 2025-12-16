package domain

import "time"

// ExternalUserDomain 外部包的用户Domain类型
type ExternalUserDomain struct {
	ID    int64
	Name  string
	Email string
}

// ExportPatch 导出patch信息（用于测试）
func (e *ExternalUserDomain) ExportPatch() *ExternalUserDomainPatch {
	return &ExternalUserDomainPatch{}
}

// ExternalUserDomainPatch patch类型
type ExternalUserDomainPatch struct {
	ID    PatchField
	Name  PatchField
	Email PatchField
}

// PatchField patch字段
type PatchField struct {
	present bool
}

// IsPresent 返回字段是否存在
func (f PatchField) IsPresent() bool {
	return f.present
}

// ApprovalDomain 审批Domain类型（用于测试外部嵌入类型）
type ApprovalDomain struct {
	ID        uint64
	CreatedAt time.Time
	UpdatedAt time.Time
	Title     string
	Status    int
}
