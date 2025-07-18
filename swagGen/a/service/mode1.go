package service

type BaseResponse[T any] struct {
}

// UserInfo 用户信息
type UserInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// UpdateUserReq 更新用户请求
type UpdateUserReq struct {
	Name  string `form:"name"`
	Email string `form:"email"`
}
