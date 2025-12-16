package testdata

// ToPO 跨文件测试 - ToPO 函数在单独文件中
// CrossFilePO 结构体定义在 cross_file_po.go 中
func (p *CrossFilePO) ToPO(d *CrossFileDomain) *CrossFilePO {
	if d == nil {
		return nil
	}
	return &CrossFilePO{
		Model: Model{
			ID:        d.ID,
			CreatedAt: d.CreatedAt,
			UpdatedAt: d.UpdatedAt,
		},
		Username: d.Username,
		Email:    d.Email,
		Score:    d.Score,
	}
}
