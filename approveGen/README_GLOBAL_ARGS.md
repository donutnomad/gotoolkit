# ApproveGen 全局Args功能增强

## 功能概述

ApproveGen 现在支持分离式定义全局模板和默认args参数。模板定义和args定义可以独立配置，当各个子 `func` 没有定义 `args` 时，会自动使用全局定义的参数。

## 使用方法

### 1. 分离式定义模板和args

```go
// 定义模板内容（保持原有方式）
// @Approve(global::template="ApproveFor={{.Receiver}}.approvalRepo.Create(ctx, \"{{index .Info.Attributes \"module\"}}\", \"{{index .Info.Attributes \"event\"}}\", {{.MethodArg}})")
// 定义全局args（新增功能）
// @Approve(global::template="ApproveFor::args=[\"env taas-backend/app/launchpad/internal/biz/approval#Env\"]")
func templateDefine() {}
```

### 2. 子方法自动继承全局args

```go
// 这个方法会自动继承全局定义的env参数
// @Approve func:name="ApproveFor"; module="LISTING"; event="CREATE"
func (s *Service) CreateListing(ctx context.Context, req *CreateListingRequest) error {
    return nil
}
```

生成的代码：
```go
func (s *Service) ApproveFor_CreateListing(ctx context.Context, req *CreateListingRequest, env approval.Env) error {
    return s.approvalRepo.Create(ctx, "LISTING", "CREATE", &_ServiceMethodCreateListing{Req: req})
}
```

### 3. 子方法覆盖全局args

```go
// 定义自己的args会覆盖全局args
// @Approve func:name="ApproveFor"; module="LISTING"; event="UPDATE"; args=["operatorID int64"]
func (s *Service) UpdateListing(ctx context.Context, id int64, req *UpdateListingRequest) error {
    return nil
}
```

生成的代码：
```go
func (s *Service) ApproveFor_UpdateListing(ctx context.Context, id int64, req *UpdateListingRequest, operatorID int64) error {
    return s.approvalRepo.Create(ctx, "LISTING", "UPDATE", &_ServiceMethodUpdateListing{Id: id, Req: req})
}
```

## 语法格式

### 模板定义（保持原有语法）
```
@Approve(global::template="FuncName=模板内容")
```

### Args定义（新增语法）
```
@Approve(global::template="FuncName::args=[\"参数定义\"]")
```

### 参数定义格式
- 简单类型：`"name string"`、`"count int"`
- 带导入路径：`"env package/path#TypeName"`

### 多个参数
```go
// @Approve(global::template="ValidateFor::args=[\"userID int64\", \"reason string\"]")
```

## 完整示例

```go
// 定义模板和args（分离配置）
// @Approve(global::template="ApproveFor={{.Receiver}}.approvalRepo.Create(ctx, \"{{index .Info.Attributes \"module\"}}\", \"{{index .Info.Attributes \"event\"}}\", {{.MethodArg}})")
// @Approve(global::template="ApproveFor::args=[\"env taas-backend/app/launchpad/internal/biz/approval#Env\"]")
func templateDefine() {}

// 另一个模板示例
// @Approve(global::template="ValidateFor={{.Receiver}}.validateWithReason(ctx, \"{{index .Info.Attributes \"module\"}}\", {{.MethodArg}})")
// @Approve(global::template="ValidateFor::args=[\"userID int64\", \"reason string\"]")
func templateDefineValidation() {}

// 使用全局args
// @Approve func:name="ApproveFor"; module="LISTING"; event="CREATE"
func (s *Service) CreateListing(ctx context.Context, req *CreateListingRequest) error {
    return nil
}

// 覆盖全局args
// @Approve func:name="ApproveFor"; module="LISTING"; event="UPDATE"; args=["operatorID int64"]
func (s *Service) UpdateListing(ctx context.Context, id int64, req *UpdateListingRequest) error {
    return nil
}
```

## 优先级规则

1. **子方法有args定义**：使用子方法的args，忽略全局args
2. **子方法无args定义**：自动继承对应函数名的全局args
3. **无全局args定义**：按原来的方式工作，不添加额外参数

## 兼容性

- ✅ **完全向后兼容**：原有的模板定义方式继续有效
- ✅ **独立配置**：模板内容和args可以分别定义
- ✅ **灵活组合**：可以只定义模板、只定义args，或两者都定义

## 支持的功能

- ✅ 简单类型参数（`string`, `int`, `bool` 等）
- ✅ 带导入路径的类型（`package/path#TypeName`）
- ✅ 多个参数定义
- ✅ 自动导入管理
- ✅ 向后兼容原有功能
- ✅ 支持 `nest=true` 功能
- ✅ 分离式配置模板和args

## 完整示例

查看 `testdata/test_complete_global_args.go` 获取完整的使用示例。