# SwagGen - Swagger 文档和 Gin 绑定代码生成器

SwagGen 是一个 Go 工具，用于从带有特殊注释的 Go 接口定义中自动生成 Swagger 文档注释和 Gin HTTP 路由绑定代码。

## 功能特性

- **多接口支持**：一次处理多个接口定义
- **智能包管理**：自动处理包别名冲突（如 `types`, `types2`, `types3`）
- **类型引用强制导入**：生成 `var _` 声明确保 swaggo 正确识别类型
- **完整的 Swagger 注释**：生成标准的 Swagger 文档注释
- **Gin 绑定代码**：自动生成 Gin 路由绑定和参数处理代码

## 支持的注释

### HTTP 方法和路径
```go
// @POST(/api/v1/user/{id})
// @GET(/api/v1/user/{id})
// @PUT(/api/v1/user/{id})
// @DELETE(/api/v1/user/{id})
```

### 内容类型
```go
// @FORM                     - application/x-www-form-urlencoded
// @JSON                     - application/json
// @MULTIPART               - multipart/form-data
// @MIME(application/json)   - 自定义 MIME 类型
```

### 参数类型
```go
// @PARAM     - 路径参数
// @QUERY    - 查询参数
// @FORM     - 表单参数
// @BODY     - 请求体参数
// @HEADER   - 头部参数
```

## 使用方法

### 命令行参数

```bash
go run swagGen/main.go [选项]
```

**选项：**
- `-path string` - 目录路径或文件路径（必需）
- `-out string` - 输出文件名（默认 "swagger_generated.go"）
- `-package string` - 包名（可选，默认从文件推断）
- `-interfaces string` - 要处理的接口名称，逗号分隔（可选）
- `-v` - 详细输出

### 示例用法

```bash
# 处理整个目录
go run swagGen/main.go -path ./api -out swagger.go

# 处理单个文件
go run swagGen/main.go -path ./api/user.go

# 指定特定接口
go run swagGen/main.go -path ./api -interfaces "IUserAPI,IAdminAPI"

# 详细输出
go run swagGen/main.go -path ./api -package myapi -v
```

## 接口定义示例

```go
package api

import (
    "context"
    "github.com/gin-gonic/gin"
    "service"
)

// IUserAPI 用户 API 接口
type IUserAPI interface {
    // SendOTP Send OTP (Register/Forget Password)
    // Send OTP to email (1 minute, 5 times per email)
    // @POST(/api/v1/user/send-otp)
    // @JSON
    SendOTP(
        ctx context.Context,
        // @BODY
        req SendOTPReq,
    ) service.BaseResponse[string]

    // GetUser 获取用户信息
    // 根据用户ID获取用户详细信息
    // @GET(/api/v1/user/{userId})
    // @JSON
    GetUser(
        ctx context.Context,
        // @PARAM
        userId string,
    ) service.BaseResponse[UserInfo]
}

type SendOTPReq struct {
    Email string `json:"email"`
    Scene int    `json:"scene"`
}

type UserInfo struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}
```

## 生成的代码示例

### Swagger 注释
```go
// SendOTP
// @Summary SendOTP Send OTP (Register/Forget Password)
// @Description Send OTP to email (1 minute, 5 times per email)
// @Tags IUserAPI
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param req body SendOTPReq true "req"
// @Success 200 {object} service.BaseResponse[string]
// @Router /api/v1/user/send-otp [post]
```

### Gin 绑定代码
```go
type IUserAPIWrap struct {
    inner IUserAPI
}

func (a *IUserAPIWrap) BindSendOTP(router gin.IRoutes, preHandlers ...gin.HandlerFunc) {
    a.bind(router, "POST", "/api/v1/user/send-otp", preHandlers, func(c *gin.Context) {
        var req SendOTPReq
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
        data := a.inner.SendOTP(c, req)
        c.JSON(200, data)
    })
}

func NewIUserAPIWrap(inner IUserAPI) *IUserAPIWrap {
    return &IUserAPIWrap{inner: inner}
}
```

### 强制类型导入
```go
// 强制导入所有使用的类型，确保 swaggo 能正确识别
var _ service.BaseResponse
var _ SendOTPReq
var _ UserInfo
```

## 包别名处理

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

## 注意事项

1. 接口方法必须包含 Swagger 注释才会被处理
2. `context.Context` 和 `*gin.Context` 参数会被自动处理
3. 生成的文件只包含注释，通过 `var _` 声明强制导入类型
4. 路径参数会从 URL 路径中自动提取

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
└── testdata/
    ├── example.go       # 示例接口定义
    └── test_output.go   # 生成的输出示例
```

这个工具基于现有的 `approveGen` 和 `sliceGen` 工具的架构，复用了项目中已有的工具和模式。