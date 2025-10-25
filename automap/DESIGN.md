# AutoMap Parser 设计文档

## 核心需求分析

基于提供的示例代码，我们需要实现一个能够自动解析Go映射函数并生成字段映射逻辑的解析器。

### 支持的函数格式
1. `Func(a *A) *B` => 传递命令字符串为: "Func"
2. `Func(a *A) (*B, error)` => 传递命令字符串为: "Func"  
3. `(x *X) Func(a *A) *B` => 传递命令字符串为: "X.Func"
4. `(x *X) Func(a *A) (*B, error)` => 传递命令字符串为: "X.Func"

### 核心功能要求
1. 自动解析A/B类型定义和字段结构
2. 支持GORM标签解析（column:"name"）
3. 处理复杂的字段映射关系（一对一、一对多、多对一）
4. 验证ExportPatch方法存在
5. 生成最终的映射逻辑代码

## 核心数据结构

### 1. 函数签名信息
```go
type FuncSignature struct {
    PackageName string    // 包名
    Receiver    string    // 接收者类型（如"X"）
    FuncName    string    // 函数名
    InputType   TypeInfo  // 输入类型A
    OutputType  TypeInfo  // 输出类型B
    HasError    bool      // 是否返回error
}
```

### 2. 类型信息
```go
type TypeInfo struct {
    Name       string        // 类型名（如"A"）
    Package    string        // 包名（如""表示当前包）
    FullName   string        // 完整类型名（如"package.A"）
    FilePath   string        // 定义文件路径
    Fields     []FieldInfo   // 字段列表
    IsPointer  bool          // 是否为指针类型
}
```

### 3. 字段信息
```go
type FieldInfo struct {
    Name       string            // 字段名
    Type       string            // 字段类型
    GormTag    string            // GORM标签
    ColumnName string            // 数据库列名
    IsJSONType bool              // 是否为JSONType
    JSONFields []JSONFieldInfo   // JSON字段信息
    SourceType string            // 来源类型（嵌入字段）
}

type JSONFieldInfo struct {
    Name string // JSON字段名
    Type string // JSON字段类型
    Tag  string // JSON标签
}
```

### 4. 映射关系
```go
type MappingRelation struct {
    AField      string   // A类型字段名
    BFields     []string // B类型对应字段名列表
    IsJSONType  bool     // 是否为JSONType映射
    JSONField   string   // JSON字段名（如果IsJSONType为true）
}

type FieldMapping struct {
    OneToOne   map[string]string     // A字段 -> B字段（一对一）
    OneToMany  map[string][]string   // A字段 -> B字段列表（一对多）
    ManyToOne  map[string][]string   // B字段 -> A字段列表（多对一）
    JSONFields map[string]JSONMapping // JSON字段映射
}

type JSONMapping struct {
    FieldName string                       // JSON字段名
    SubFields map[string]string            // A字段 -> JSON子字段
}
```

### 5. 解析结果
```go
type ParseResult struct {
    FuncSignature FuncSignature
    AType         TypeInfo
    BType         TypeInfo
    FieldMapping  FieldMapping
    HasExportPatch bool
    GeneratedCode string
}
```

## 解析流程设计

### 1. 函数签名解析
1. 使用Go AST解析函数定义
2. 提取函数名、接收者、参数和返回值
3. 验证函数格式是否符合要求
4. 提取输入输出类型信息

### 2. 类型解析
1. 根据import路径找到类型定义文件
2. 解析结构体字段和标签
3. 处理嵌入字段和嵌套结构
4. 验证ExportPatch方法存在

### 3. 字段映射分析
1. 分析映射函数体中的字段赋值逻辑
2. 识别一对一、一对多、多对一映射关系
3. 处理JSONType字段的特殊映射
4. 生成字段映射关系图

### 4. 代码生成
1. 根据映射关系生成条件判断代码
2. 处理JSONType字段的set操作
3. 生成最终的map[string]any返回逻辑

## 关键技术点

### 1. 类型追踪
- 使用现有的`internal/structparse`和`internal/xast`工具
- 支持跨包类型解析
- 处理循环引用和嵌套结构

### 2. GORM标签解析
- 复用现有的`internal/gormparse`功能
- 支持column标签提取
- 处理默认命名规则（ToSnakeCase）

### 3. 映射关系识别
- 通过AST分析赋值语句
- 识别字段间的对应关系
- 处理条件逻辑（if/else）

### 4. 错误处理
- 类型未找到错误
- ExportPatch方法缺失错误
- 映射关系解析错误

## 实现计划

1. **核心数据结构定义** - 定义上述所有数据结构
2. **函数签名解析器** - 实现FuncSignature解析
3. **类型解析器** - 集成现有工具实现TypeInfo解析
4. **字段映射分析器** - 实现MappingRelation识别
5. **代码生成器** - 根据映射关系生成最终代码
6. **主解析器** - 整合所有组件
7. **测试和工具** - 编写测试用例和CLI工具

## 使用示例

```go
// 输入：函数名 "MapAToB"
// 输出：生成的映射代码
result, err := automap.Parse("MapAToB")
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.GeneratedCode)