package service

type BaseResponse[T any] struct {
}

// UpdateUserReq 更新用户请求
type UpdateUserReq struct {
	Name  string `form:"name"`
	Email string `form:"email"`
}
