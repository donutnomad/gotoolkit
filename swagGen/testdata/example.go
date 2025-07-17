package main

import (
	"context"
	"github.com/donutnomad/gotoolkit/swagGen/a/service"
	service2 "github.com/donutnomad/gotoolkit/swagGen/service"
	"github.com/gin-gonic/gin"
)

//go:generate ../../build/swagGen -path ./example.go -out example_out.go

// SendOTPReq 发送 OTP 请求
type SendOTPReq struct {
	Email string `json:"email" form:"email"`
	Scene int    `json:"scene" form:"scene"`
}

// ISwapAPI 示例接口
type ISwapAPI interface {
	// SendOTP Send OTP (Register/Forget Password)
	// Send OTP to email (1 minute, 5 times per email)
	// @POST(/api/v1/swap/v1/{id2})
	// @POST(/api/v1/swap/v2/{id2})
	// @JSON
	SendOTP(
		c *gin.Context,
		// @PARAM
		id2 string,
		// @FORM
		data SendOTPReq,
	) service2.BaseResponse[string]
}

// IUserAPI 另一个示例接口
type IUserAPI interface {
	// GetUser 获取用户信息
	// 根据用户ID获取用户详细信息
	// @GET(/api/v1/user/{userId})
	// @JSON
	GetUser(
		ctx context.Context,
		// @PARAM
		userId string,
	) service.BaseResponse[UserInfo]

	// CreateUser 创建用户
	// 创建新用户账户
	// @POST(/api/v1/user)
	// @JSON
	CreateUser(
		ctx context.Context,
		// @BODY
		user CreateUserReq,
	) service.BaseResponse[UserInfo]

	// UpdateUser 更新用户信息
	// 更新现有用户的信息
	// @PUT(/api/v1/user/{userId})
	// @FORM
	UpdateUser(
		ctx context.Context,
		// @PARAM(userId)
		uid string,
		// @FORM
		user service.UpdateUserReq,
	) service.BaseResponse[service.UserInfo]

	// GetUserByAge 根据年龄获取用户
	// 测试 int 类型参数转换
	// @GET(/api/v1/user/age/{age})
	// @JSON
	GetUserByAge(
		ctx context.Context,
		// @PARAM
		age int,
	) service.BaseResponse[[]UserInfo]

	// GetUserByID 根据ID获取用户
	// 测试 int64 类型参数转换
	// @GET(/api/v1/user/id/{id})
	// @JSON
	GetUserByID(
		ctx context.Context,
		// @PARAM
		id int64,
	) service.BaseResponse[UserInfo]

	// GetUsersByScore 根据分数获取用户
	// 测试 float64 类型参数转换
	// @GET(/api/v1/user/score/{score})
	// @JSON
	GetUsersByScore(
		ctx context.Context,
		// @PARAM
		score float64,
	) service.BaseResponse[[]UserInfo]
}

// UserInfo 用户信息
type UserInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CreateUserReq 创建用户请求
type CreateUserReq struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
