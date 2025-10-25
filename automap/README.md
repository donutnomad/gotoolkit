# AutoMap

AutoMap是一个Go语言AST解析器，用于自动解析Go映射函数并生成字段映射逻辑代码。

## 功能特性

- 支持4种函数格式：
  - `Func(a *A) *B`
  - `Func(a *A) (*B, error)`
  - `(x *X) Func(a *A) *B`
  - `(x *X) Func(a *A) (*B, error)`

- 自动解析A/B类型定义和字段结构
- 支持GORM标签解析（column:"name"）
- 处理复杂的字段映射关系（一对一、一对多、多对一）
- 验证ExportPatch方法存在
- 生成最终的映射逻辑代码

## 使用示例

```go
package main

import (
    "fmt"
    "github.com/donutnomad/gotoolkit/automap"
)

func main() {
    // 解析映射函数
    result, err := automap.Parse("MapAToB")
    if err != nil {
        panic(err)
    }

    fmt.Printf("函数名: %s\n", result.FuncSignature.FuncName)
    fmt.Printf("输入类型: %s\n", result.AType.Name)
    fmt.Printf("输出类型: %s\n", result.BType.Name)
    fmt.Printf("是否有ExportPatch方法: %v\n", result.HasExportPatch)

    // 生成完整代码
    code, err := automap.ParseAndGenerate("MapAToB")
    if err != nil {
        panic(err)
    }

    fmt.Println("生成的代码:")
    fmt.Println(code)
}
```

## 生成的代码示例

```go
func Do(input *A) map[string]any {
    b := MapAToB(input)
    fields := input.ExportPatch()
    var ret = make(map[string]any)

    // A的一个字段，对应B的一个字段
    if fields.ID.IsPresent() {
        ret["id"] = b.ID
    }

    // A的一个字段，对应B的多个字段
    if fields.Book.IsPresent() {
        ret["book_name"] = b.BookName
        ret["book_author"] = b.BookAuthor
        ret["book_year"] = b.BookYear
    }

    // JSON字段映射
    {
        set := datatypes.JSONSet("token1")
        if fields.TokenName.IsPresent() {
            ret["token1"] = set.Set("name", fields.TokenName.MustGet())
        }
        if fields.TokenSymbol.IsPresent() {
            ret["token1"] = set.Set("symbol", fields.TokenSymbol.MustGet())
        }
    }

    return ret
}
```

## API

### 主要方法

- `Parse(funcName string) (*ParseResult, error)` - 解析映射函数
- `ParseAndGenerate(funcName string) (string, error)` - 解析并生成完整代码
- `ValidateFunction(funcName string) error` - 验证函数是否符合要求

### ParseResult 结构体

```go
type ParseResult struct {
    FuncSignature    FuncSignature
    AType            TypeInfo
    BType            TypeInfo
    FieldMapping     FieldMapping
    HasExportPatch   bool
    GeneratedCode    string
    MappingRelations []MappingRelation
}
```

## 核心组件

1. **Parser** - AST解析器，解析函数签名和类型定义
2. **TypeResolver** - 类型解析器，支持跨包类型解析
3. **MappingAnalyzer** - 映射分析器，分析字段映射关系
4. **Validator** - 验证器，验证解析结果的正确性
5. **CodeGenerator** - 代码生成器，生成最终映射代码

## 注意事项

1. A类型必须实现`ExportPatch() *APatch`方法
2. 支持GORM标签解析，自动提取column名称
3. 支持JSONType字段的特殊映射处理
4. 禁止硬编码任何结构体名称或路径，所有信息基于自动解析

## 测试

运行测试：

```bash
go test -v
```

运行特定测试：

```bash
go test -v -run TestSimpleCall
```