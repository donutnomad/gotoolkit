# SwagGen - Swagger 文档和 Gin 绑定代码生成器

SwagGen 是一个用于从 Go 接口定义生成 Swagger 文档和 Gin 绑定代码的工具。它通过解析带有特定注释的接口，自动生成相应的 Swagger 注释和 Gin 路由绑定代码。

## 功能特性

- 🚀 **自动生成代码**：从接口定义自动生成 Swagger 注释和 Gin 绑定代码
- 📝 **丰富的注释支持**：支持路由、参数、请求/响应类型、安全认证等多种注释
- 🎯 **类型安全**：自动进行类型转换，支持基本类型和复杂类型
- 🔧 **灵活配置**：支持接口级别和方法级别的注释配置
- 🌐 **中间件支持**：支持为路由添加中间件
- 📊 **排除机制**：支持排除特定方法或在 BindAll 中排除方法
- 🔄 **智能包管理**：自动处理包别名冲突（如 `types`, `types2`, `types3`）
- 💪 **类型引用强制导入**：生成 `var _` 声明确保 swaggo 正确识别类型

## 安装

```bash
# 从源码构建
make buildSwag

# 或者直接构建
go build -o swagGen ./swagGen
```

## 基本用法

### 命令行选项

```bash
swagGen [选项]
```

**选项说明：**

- `-path string`：目录路径或文件路径（必需）
- `-out string`：输出文件名（默认 "swagger_generated.go"）
- `-package string`：包名（可选，默认从文件推断）
- `-interfaces string`：要处理的接口名称，逗号分隔（可选，默认处理所有带注释的接口）
- `-v`：详细输出

### 使用示例

```bash
# 处理整个目录
swagGen -path ./api -out swagger.go

# 处理单个文件
swagGen -path ./api/user.go -interfaces "IUserAPI,IAdminAPI"

# 指定包名和详细输出
swagGen -path ./api -package myapi -v
```

## 注释语法

### 1. 路由注释

定义 HTTP 方法和路径：

```go
type IUserAPI interface {
    // @GET(/api/v1/user/{id})
    GetUser(ctx context.Context, id string) UserResponse
    
    // @POST(/api/v1/user)
    CreateUser(ctx context.Context, user CreateUserReq) UserResponse
    
    // @PUT(/api/v1/user/{id})
    UpdateUser(ctx context.Context, id string, user UpdateUserReq) UserResponse
    
    // @PATCH(/api/v1/user/{id})
    PatchUser(ctx context.Context, id string, patch PatchUserReq) UserResponse
    
    // @DELETE(/api/v1/user/{id})
    DeleteUser(ctx context.Context, id string) error
}
```

### 2. 参数注释

#### 路径参数

```go
// @GET(/api/v1/user/{userId})
GetUser(
    ctx context.Context,
    // @PARAM
    userId string,
) UserResponse

// 带别名的路径参数
// @GET(/api/v1/user/{user_id})
GetUserById(
    ctx context.Context,
    // @PARAM(user_id)
    userId string,
) UserResponse
```

#### 查询参数

```go
// @GET(/api/v1/users)
GetUsers(
    ctx context.Context,
    // @QUERY
    req GetUsersReq,
) UsersResponse
```

#### 头部参数

```go
// @GET(/api/v1/user/{id})
GetUser(
    ctx context.Context,
    // @HEADER
    token string,
    // @PARAM
    id string,
) UserResponse
```

#### 请求体参数

```go
// @POST(/api/v1/user)
CreateUser(
    ctx context.Context,
    // @BODY
    user CreateUserReq,
) UserResponse

// 表单参数
// @POST(/api/v1/user)
CreateUser(
    ctx context.Context,
    // @FORM
    user CreateUserReq,
) UserResponse
```

### 3. 请求内容类型

```go
// JSON 请求
// @POST(/api/v1/user)
// @JSON-REQ
CreateUser(ctx context.Context, user CreateUserReq) UserResponse

// 表单请求
// @POST(/api/v1/user)
// @FORM-REQ
CreateUser(ctx context.Context, user CreateUserReq) UserResponse

// 自定义请求类型
// @POST(/api/v1/user)
// @MIME-REQ(application/xml)
CreateUser(ctx context.Context, user CreateUserReq) UserResponse
```

### 4. 响应内容类型

```go
// JSON 响应
// @GET(/api/v1/user/{id})
// @JSON
GetUser(ctx context.Context, id string) UserResponse

// 自定义响应类型
// @GET(/api/v1/user/{id})
// @MIME(application/xml)
GetUser(ctx context.Context, id string) UserResponse
```

### 5. 接口级别注释

接口级别的注释会应用到该接口的所有方法：

```go
// @TAG(User)
// @SECURITY(ApiKeyAuth)
// @HEADER(X-API-Version,true,"API version")
type IUserAPI interface {
    // @GET(/api/v1/user/{id})
    GetUser(ctx context.Context, id string) UserResponse
    
    // @POST(/api/v1/user)
    CreateUser(ctx context.Context, user CreateUserReq) UserResponse
}
```

### 6. 中间件注释

```go
// @GET(/api/v1/user/{id})
// @MID(auth,log)
GetUser(ctx context.Context, id string) UserResponse
```

### 7. 控制注释

```go
// 移除方法（不生成代码）
// @GET(/api/v1/user/{id})
// @Removed
GetUser(ctx context.Context, id string) UserResponse

// 排除在 BindAll 方法之外
// @GET(/api/v1/user/{id})
// @ExcludeFromBindAll
GetUser(ctx context.Context, id string) UserResponse
```

### 8. 排除语法

可以在接口级别注释中排除特定方法：

```go
// @TAG(Company;exclude="StartTransfer")
// @SECURITY(ApiKeyAuth;exclude="StartTransfer,GetTokenHistory")
type ICompanyAPI interface {
    // @GET(/api/company/tokens/{token_id}/balance)
    GetTokenBalance(ctx *gin.Context, tokenId uint) service.BaseResponse[string]
    
    // @GET(/api/company/tokens/{token_id}/history)
    GetTokenHistory(ctx *gin.Context, tokenId uint, req string) service.BaseResponse[string]
    
    // @POST(/api/company/tokens/{token_id}/start-transfer)
    StartTransfer(ctx *gin.Context, tokenId uint, req string) service.BaseResponse[string]
}
```

## 完整示例

### 输入文件 (user_api.go)

```go
package api

import (
    "context"
    "github.com/gin-gonic/gin"
)

//go:generate swagGen -path ./user_api.go -out user_api_generated.go

// SendOTPReq 发送 OTP 请求
type SendOTPReq struct {
    Email string `json:"email" form:"email"`
    Scene int    `json:"scene" form:"scene"`
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

// BaseResponse 基础响应
type BaseResponse[T any] struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    T      `json:"data"`
}

// @TAG(User)
// @SECURITY(ApiKeyAuth)
type IUserAPI interface {
    // GetUser 获取用户信息
    // 根据用户ID获取用户详细信息
    // @GET(/api/v1/user/{userId})
    // @JSON
    GetUser(
        ctx context.Context,
        // @PARAM
        userId string,
    ) BaseResponse[UserInfo]

    // CreateUser 创建用户
    // 创建新用户账户
    // @POST(/api/v1/user)
    // @JSON
    CreateUser(
        ctx context.Context,
        // @BODY
        user CreateUserReq,
    ) BaseResponse[UserInfo]

    // UpdateUser 更新用户信息
    // @PUT(/api/v1/user/{userId})
    // @FORM-REQ
    UpdateUser(
        ctx context.Context,
        // @PARAM(userId)
        uid string,
        // @FORM
        user UpdateUserReq,
    ) BaseResponse[UserInfo]

    // GetUserByAge 根据年龄获取用户
    // @GET(/api/v1/user/age/{age})
    // @JSON
    GetUserByAge(
        ctx context.Context,
        // @PARAM
        age int,
    ) BaseResponse[[]UserInfo]
}
```

### 生成的代码

运行 `swagGen -path ./user_api.go -out user_api_generated.go` 后，会生成包含以下内容的文件：

#### 1. 文件头部和导入

```go
// Code generated by swagGen. DO NOT EDIT.
//
// This file contains Swagger documentation and Gin binding code.
// Generated from interface definitions with Swagger annotations.

package api

import (
    "context"
    "github.com/gin-gonic/gin"
    "github.com/spf13/cast"
    "strings"
)

// 强制导入所有使用的类型，确保 swaggo 能正确识别
var _ BaseResponse[UserInfo]
var _ CreateUserReq
var _ UserInfo
```

#### 2. Swagger 注释

```go
// GetUser
// @Summary GetUser 获取用户信息
// @Description 根据用户ID获取用户详细信息
// @Tags User
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param userId path string true "userId"
// @Success 200 {object} BaseResponse[UserInfo]
// @Router /api/v1/user/{userId} [get]
func (a *UserAPIWrap) GetUser(ctx *gin.Context) {
    userId := cast.ToString(ctx.Param("userId"))
    var result BaseResponse[UserInfo] = a.inner.GetUser(ctx, userId)
    onGinResponse(ctx, result)
}
```

#### 3. Gin 绑定代码

```go
func NewUserAPIWrap(inner IUserAPI) *UserAPIWrap {
    return &UserAPIWrap{
        inner: inner,
    }
}

type UserAPIWrap struct {
    inner IUserAPI
}

func (a *UserAPIWrap) bind(router gin.IRoutes, method, path string, preHandlers, innerHandlers []gin.HandlerFunc, f gin.HandlerFunc) {
    var basePath string
    if v, ok := router.(interface {
        BasePath() string
    }); ok {
        basePath = v.BasePath()
    }
    handlers := make([]gin.HandlerFunc, 0, len(preHandlers)+len(innerHandlers)+1)
    handlers = append(handlers, preHandlers...)
    handlers = append(handlers, innerHandlers...)
    handlers = append(handlers, f)
    router.Handle(method, strings.TrimPrefix(path, basePath), handlers...)
}

func (a *UserAPIWrap) BindGetUser(router gin.IRoutes, preHandlers ...gin.HandlerFunc) {
    a.bind(router, "GET", "/api/v1/user/:userId", preHandlers, nil, a.GetUser)
}

func (a *UserAPIWrap) BindCreateUser(router gin.IRoutes, preHandlers ...gin.HandlerFunc) {
    a.bind(router, "POST", "/api/v1/user", preHandlers, nil, a.CreateUser)
}

func (a *UserAPIWrap) BindAll(router gin.IRoutes, preHandlers ...gin.HandlerFunc) {
    a.BindGetUser(router, preHandlers...)
    a.BindCreateUser(router, preHandlers...)
}
```

### 使用生成的代码

```go
// 实现业务接口
type UserService struct{}

func (s *UserService) GetUser(ctx context.Context, userId string) BaseResponse[UserInfo] {
    // 业务逻辑
    return BaseResponse[UserInfo]{
        Code:    200,
        Message: "success",
        Data:    UserInfo{ID: userId, Name: "John", Email: "john@example.com"},
    }
}

func (s *UserService) CreateUser(ctx context.Context, user CreateUserReq) BaseResponse[UserInfo] {
    // 业务逻辑
    return BaseResponse[UserInfo]{
        Code:    200,
        Message: "success",
        Data:    UserInfo{ID: "123", Name: user.Name, Email: user.Email},
    }
}

// 使用生成的绑定代码
func main() {
    router := gin.Default()
    
    // 创建服务实例
    userService := &UserService{}
    
    // 创建包装器
    userWrapper := NewUserAPIWrap(userService)
    
    // 绑定所有路由
    userWrapper.BindAll(router)
    
    // 或者单独绑定路由
    // userWrapper.BindGetUser(router)
    // userWrapper.BindCreateUser(router)
    
    router.Run(":8080")
}
```

## 高级特性

### 1. 包别名处理

当遇到同名包时，SwagGen 会自动生成别名：

```go
import (
    "go.com/pkg/types"              // 第一个保持原名
    types2 "go.com/pkg/v2/types"    // 第二个使用别名 types2
    types3 "go.com/pkg/v3/types"    // 第三个使用别名 types3
)

var _ types.Response
var _ types2.Request  
var _ types3.Error
```

### 2. 类型支持

SwagGen 支持以下类型的自动转换：

- **基本类型**：`string`, `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`, `bool`
- **复杂类型**：结构体、切片、指针、泛型
- **特殊类型**：`*gin.Context`, `context.Context`

### 3. 中间件支持

当接口方法包含中间件注释时，会生成相应的中间件接口：

```go
// 如果有中间件注释，会生成这样的接口
type IUserAPIHandler interface {
    Auth() gin.HandlerFunc
    Log() gin.HandlerFunc
}

// 构造函数会接受中间件实现
func NewUserAPIWrap(inner IUserAPI, handler IUserAPIHandler) *UserAPIWrap {
    return &UserAPIWrap{
        inner: inner,
        handler: handler,
    }
}
```

### 4. 辅助函数

生成的代码依赖以下辅助函数（需要用户实现）：

```go
// 绑定请求参数
func onGinBind(c *gin.Context, val any, typ string) bool {
    var err error
    switch typ {
    case "JSON":
        err = c.ShouldBindJSON(val)
    case "FORM":
        err = c.ShouldBind(val)
    case "QUERY":
        err = c.ShouldBindQuery(val)
    default:
        err = c.ShouldBind(val)
    }
    if err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return false
    }
    return true
}

// 处理绑定错误
func onGinBindErr(c *gin.Context, err error) {
    c.JSON(500, gin.H{"error": err.Error()})
}

// 返回响应
func onGinResponse[T any](c *gin.Context, data T) {
    c.JSON(200, data)
}
```

## 构建和测试

```bash
# 构建工具
make buildSwag

# 运行测试
cd swagGen/testdata
go generate

# 测试 Swagger 文档生成
make testSwag
```

## 项目结构

```
swagGen/
├── main.go              # 主程序入口
├── types.go             # 类型定义
├── import_manager.go    # 导入管理器
├── annotation_parser.go # 注释解析器
├── type_parser.go       # 类型解析器
├── interface_parser.go  # 接口解析器
├── swagger_generator.go # Swagger 生成器
├── gin_generator.go     # Gin 生成器
├── gofmt/              # 代码格式化
├── parser/             # 注释解析器
└── testdata/           # 测试数据
    ├── example.go       # 示例接口定义
    ├── example_out.go   # 生成的输出示例
    ├── company_api.go   # 公司API示例
    └── company_api_out.go # 生成的输出示例
```

## 注意事项

1. **接口方法必须包含 Swagger 注释**：只有带有路由注释（如 `@GET`、`@POST`）的方法才会被处理
2. **参数顺序**：生成的代码会按照接口定义的参数顺序传递参数
3. **类型转换**：路径参数和查询参数会自动进行类型转换
4. **特殊参数**：`context.Context` 和 `*gin.Context` 参数会被自动处理
5. **辅助函数**：需要实现 `onGinBind`, `onGinBindErr`, `onGinResponse` 等辅助函数
6. **路径参数映射**：支持自动映射路径参数名（如 `request_id` -> `requestID`）

## 贡献

欢迎提交 Issue 和 Pull Request 来改进这个工具。

## 许可证

MIT License