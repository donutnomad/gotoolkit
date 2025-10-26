package automap

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"unicode"

	"github.com/davecgh/go-spew/spew"
)

// MappingAnalyzer 映射分析器
type MappingAnalyzer struct {
	fset         *token.FileSet
	funcDecl     *ast.FuncDecl
	aType        *TypeInfo
	bType        *TypeInfo
	mappingRel   []MappingRelation
	fieldMapping FieldMapping
	varMap       map[string]ast.Expr // 变量名到表达式的映射
	currentOrder int                 // 当前赋值顺序
}

// NewMappingAnalyzer 创建映射分析器
func NewMappingAnalyzer(fset *token.FileSet) *MappingAnalyzer {
	return &MappingAnalyzer{
		fset: fset,
		fieldMapping: FieldMapping{
			OneToOne:   make(map[string]string),
			OneToMany:  make(map[string][]string),
			ManyToOne:  make(map[string][]string),
			JSONFields: make(map[string]JSONMapping),
		},
		varMap: make(map[string]ast.Expr),
	}
}

// AnalyzeMapping 分析函数体中的映射关系
func (ma *MappingAnalyzer) AnalyzeMapping(funcDecl *ast.FuncDecl, aType, bType *TypeInfo) ([]MappingRelation, FieldMapping, error) {
	ma.funcDecl = funcDecl
	ma.aType = aType
	ma.bType = bType
	ma.currentOrder = 0 // 重置顺序计数器

	// 分析函数体
	if funcDecl.Body == nil {
		return nil, ma.fieldMapping, fmt.Errorf("函数体为空")
	}

	// 先分析变量声明，用于处理局部变量（如tokenName）
	// 同时也会分析结构体字面量赋值
	if err := ma.analyzeVariableDeclarations(funcDecl); err != nil {
		return nil, ma.fieldMapping, fmt.Errorf("分析变量声明失败: %w", err)
	}

	// 构建字段映射
	ma.buildFieldMapping()

	return ma.mappingRel, ma.fieldMapping, nil
}

// analyzeVariableDeclarations 分析变量声明
func (ma *MappingAnalyzer) analyzeVariableDeclarations(funcDecl *ast.FuncDecl) error {
	for _, stmt := range funcDecl.Body.List {
		if assignStmt, ok := stmt.(*ast.AssignStmt); ok {
			// 处理赋值语句，如 tokenName := a.TokenName
			if len(assignStmt.Lhs) == 1 && len(assignStmt.Rhs) == 1 {
				if ident, ok := assignStmt.Lhs[0].(*ast.Ident); ok {
					// 记录变量映射
					ma.varMap[ident.Name] = assignStmt.Rhs[0]

					// 如果是结构体字面量赋值，特殊处理
					if compLit, ok := assignStmt.Rhs[0].(*ast.CompositeLit); ok {
						// 这可能是我们要分析的结构体字面量
						ma.analyzeStructLiteral(compLit)
					} else if unaryExpr, ok := assignStmt.Rhs[0].(*ast.UnaryExpr); ok {
						// 检查是否是取地址操作符
						if compLit, ok := unaryExpr.X.(*ast.CompositeLit); ok {
							ma.analyzeStructLiteral(compLit)
						}
					}
				}
			}
		} else if retStmt, ok := stmt.(*ast.ReturnStmt); ok {
			// 处理return语句中的结构体字面量
			for _, expr := range retStmt.Results {
				if unaryExpr, ok := expr.(*ast.UnaryExpr); ok && unaryExpr.Op == token.AND {
					if compLit, ok := unaryExpr.X.(*ast.CompositeLit); ok {
						ma.analyzeStructLiteral(compLit)
					}
				} else if compLit, ok := expr.(*ast.CompositeLit); ok {
					ma.analyzeStructLiteral(compLit)
				}
			}
		}
	}
	return nil
}

// findStructLiteralInReturn 查找return语句中的结构体字面量
func (ma *MappingAnalyzer) findStructLiteralInReturn(funcDecl *ast.FuncDecl) *ast.CompositeLit {
	for _, stmt := range funcDecl.Body.List {
		if retStmt, ok := stmt.(*ast.ReturnStmt); ok {
			for _, expr := range retStmt.Results {
				if compLit, ok := expr.(*ast.CompositeLit); ok {
					return compLit
				}
			}
		}
	}
	return nil
}

// analyzeStructLiteral 分析结构体字面量
func (ma *MappingAnalyzer) analyzeStructLiteral(structLit *ast.CompositeLit) error {
	for _, elt := range structLit.Elts {
		switch e := elt.(type) {
		case *ast.KeyValueExpr:
			// Key: Value 格式，如 BookName: a.Book.Name 或 Base: {...}
			bField := ma.extractFieldName(e.Key)

			// 检查Value是否为CompositeLit（嵌入结构体）
			if compLit, ok := e.Value.(*ast.CompositeLit); ok {
				// 这是嵌入结构体，如 Base: {...}
				if err := ma.analyzeEmbeddedStructByName(bField, compLit); err != nil {
					return err
				}
			} else {
				// 普通字段映射
				aFields := ma.extractAFieldsFromExpr(e.Value)

				if len(aFields) > 0 {
					relation := MappingRelation{
						AField:  aFields[0], // 对于简单的字段映射，取第一个
						BFields: []string{bField},
						Order:   ma.currentOrder,
					}
					ma.mappingRel = append(ma.mappingRel, relation)
					ma.currentOrder++
				}
			}

		case *ast.CompositeLit:
			// 这种情况通常是嵌入结构体，但在实际的结构体字面量中，
			// 嵌入结构体应该以KeyValueExpr的形式出现，如 Base: {...}
			// 所以这里可能是其他情况，暂时跳过

		default:
			// 其他情况
		}
	}

	return nil
}

// analyzeEmbeddedStructByName 根据字段名分析嵌入结构体
func (ma *MappingAnalyzer) analyzeEmbeddedStructByName(fieldName string, embeddedLit *ast.CompositeLit) error {
	// 分析嵌入结构体中的字段
	for _, elt := range embeddedLit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			bFieldName := ma.extractFieldName(kv.Key)
			bField := fieldName + "." + bFieldName
			aFields := ma.extractAFieldsFromExpr(kv.Value)

			if len(aFields) > 0 {
				relation := MappingRelation{
					AField:  aFields[0],
					BFields: []string{bField},
					Order:   ma.currentOrder,
				}
				ma.mappingRel = append(ma.mappingRel, relation)
				ma.currentOrder++
			}
		}
	}

	return nil
}

// analyzeEmbeddedStruct 分析嵌入结构体（保留以兼容）
func (ma *MappingAnalyzer) analyzeEmbeddedStruct(embeddedLit *ast.CompositeLit) error {
	// 查找对应的嵌入字段名
	embeddedFieldName := ma.findEmbeddedFieldName(embeddedLit)
	if embeddedFieldName == "" {
		return fmt.Errorf("无法确定嵌入字段名")
	}

	return ma.analyzeEmbeddedStructByName(embeddedFieldName, embeddedLit)
}

// findEmbeddedFieldName 查找嵌入字段名
func (ma *MappingAnalyzer) findEmbeddedFieldName(embeddedLit *ast.CompositeLit) string {
	// 在B类型中查找嵌入字段
	for _, field := range ma.bType.Fields {
		if field.IsEmbedded {
			// 对于嵌入字段，field.Name可能为空，需要使用类型名
			if field.Name != "" {
				return field.Name
			}
			// 如果Name为空，尝试从Type中提取类型名
			if typeName := ma.extractTypeName(field.Type); typeName != "" {
				return typeName
			}
		}
	}
	return ""
}

// extractFieldName 提取字段名
func (ma *MappingAnalyzer) extractFieldName(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

// extractTypeName 从类型表达式中提取类型名
func (ma *MappingAnalyzer) extractTypeName(typeExpr string) string {
	// 如果类型表达式包含包名，提取类型名部分
	if dotIndex := strings.LastIndex(typeExpr, "."); dotIndex != -1 {
		return typeExpr[dotIndex+1:]
	}
	return typeExpr
}

// extractAFieldsFromExpr 从表达式中提取A类型字段
func (ma *MappingAnalyzer) extractAFieldsFromExpr(expr ast.Expr) []string {
	var fields []string

	switch e := expr.(type) {
	case *ast.SelectorExpr:
		// 处理 a.Field 或 a.Field.SubField
		fieldPath := ma.buildFieldPath(e)
		if fieldPath != "" {
			fields = append(fields, fieldPath)
		}

	case *ast.CallExpr:
		// 处理函数调用，如 datatypes.NewJSONType(...) 或 entity.PublishTime.Unix()
		if ma.isJSONTypeConstructor(e) {
			// 尝试从当前上下文推断B字段名
			bFieldName := ma.inferBFieldNameFromContext(e)
			jsonFields := ma.extractJSONFields(e, bFieldName)
			fields = append(fields, jsonFields...)
		} else if ma.isJSONSliceConstructor(e) {
			// 处理 datatypes.NewJSONSlice(...)
			if len(e.Args) > 0 {
				fieldPath := ma.buildFieldPath(e.Args[0])
				if fieldPath != "" {
					fields = append(fields, fieldPath)
				}
			}
		} else {
			// 处理其他函数调用，如 entity.PublishTime.Unix() 或 decimal.NewFromBigInt(...)

			// 首先检查是否是方法调用（如 entity.PublishTime.Unix()）
			if selectorExpr, ok := e.Fun.(*ast.SelectorExpr); ok {
				// 检查是否是包函数调用（如 decimal.NewFromBigInt）还是方法调用
				if _, isPackage := selectorExpr.X.(*ast.Ident); isPackage && !ma.isParameter(selectorExpr.X.(*ast.Ident)) {
					// 这是包函数调用，如 decimal.NewFromBigInt 或 timeToInt，只处理参数
				} else {
					// 这是方法调用，尝试从接收者中提取字段
					receiverFieldPath := ma.buildFieldPath(selectorExpr.X)
					if receiverFieldPath != "" {
						fields = append(fields, receiverFieldPath)
					}
				}
			}

			// 然后检查普通参数，支持嵌套函数调用
			for _, arg := range e.Args {
				// 递归处理参数，支持如 timeToInt(entity.PublishTime) 这样的嵌套调用
				nestedFields := ma.extractAFieldsFromExpr(arg)
				fields = append(fields, nestedFields...)
			}
		}

	case *ast.Ident:
		// 处理简单标识符，检查是否是变量
		if originalExpr, exists := ma.varMap[e.Name]; exists {
			// 如果是变量，获取原始表达式
			aFields := ma.extractAFieldsFromExpr(originalExpr)
			fields = append(fields, aFields...)
		}

	case *ast.StarExpr:
		// 处理解引用操作，如 *entity.TokenMaxSupply
		aFields := ma.extractAFieldsFromExpr(e.X)
		fields = append(fields, aFields...)

	case *ast.UnaryExpr:
		// 处理其他一元操作，如 &entity
		if e.Op == token.AND { // & 操作符
			// 处理取地址操作，递归处理操作数
			aFields := ma.extractAFieldsFromExpr(e.X)
			fields = append(fields, aFields...)
		}

	default:
		// 其他表达式类型
	}

	return fields
}

// buildFieldPath 构建字段路径
func (ma *MappingAnalyzer) buildFieldPath(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		// 首先检查是否是 a.Book 这样的嵌套选择器
		if nestedSelector, ok := e.X.(*ast.SelectorExpr); ok {
			if nestedIdent, ok := nestedSelector.X.(*ast.Ident); ok && ma.isParameter(nestedIdent) {
				// 这是 a.Book.Name 的情况，返回Book
				return ma.getTopLevelField(nestedSelector.Sel.Name)
			}
		}
		// 检查是否是参数（如 a.Field）
		if ident, ok := e.X.(*ast.Ident); ok && ma.isParameter(ident) {
			// 对于 a.Field 这样的表达式，返回字段名
			return ma.getTopLevelField(e.Sel.Name)
		}
		return ""

	case *ast.Ident:
		// 简单标识符
		if ma.isParameter(e) {
			return ""
		}
		return e.Name

	case *ast.CallExpr:
		// 处理函数调用，如 lo.Map(entity.ExchangeRules, ...)
		if len(e.Args) > 0 {
			// 递归处理第一个参数，通常是我们需要的字段
			return ma.buildFieldPath(e.Args[0])
		}
		return ""

	case *ast.StarExpr:
		// 处理解引用操作，如 *entity.TokenMaxSupply
		return ma.buildFieldPath(e.X)

	case *ast.UnaryExpr:
		// 处理取地址操作，如 &entity
		if e.Op == token.AND {
			// 递归处理操作数
			return ma.buildFieldPath(e.X)
		}
		return ""

	default:
		return ""
	}
}

// getTopLevelField 获取顶层字段名
func (ma *MappingAnalyzer) getTopLevelField(fieldName string) string {
	// 在A类型中查找该字段是否属于某个结构体字段
	if ma.hasDirectField(fieldName) {
		return fieldName
	}

	// 检查是否属于已知的结构体字段
	for _, field := range ma.aType.Fields {
		if field.IsEmbedded {
			// 这是嵌入字段，假设fieldName属于这个字段
			return field.Name
		}
	}
	return fieldName
}

// contains 检查字符串是否包含子字符串
func (ma *MappingAnalyzer) contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// hasDirectField 检查A类型是否有直接的字段
func (ma *MappingAnalyzer) hasDirectField(fieldName string) bool {
	for _, field := range ma.aType.Fields {
		if field.Name == fieldName && !field.IsEmbedded {
			return true
		}
	}
	return false
}

// isParameter 检查是否为函数参数
func (ma *MappingAnalyzer) isParameter(ident *ast.Ident) bool {
	if ma.funcDecl.Type.Params == nil {
		return false
	}

	for _, param := range ma.funcDecl.Type.Params.List {
		for _, name := range param.Names {
			if name.Name == ident.Name {
				return true
			}
		}
	}

	return false
}

// isJSONTypeConstructor 检查是否为JSONType构造函数
func (ma *MappingAnalyzer) isJSONTypeConstructor(expr *ast.CallExpr) bool {
	if fun, ok := expr.Fun.(*ast.SelectorExpr); ok {
		if x, ok := fun.X.(*ast.Ident); ok {
			return x.Name == "datatypes" && fun.Sel.Name == "NewJSONType"
		}
	}
	return false
}

// isJSONSliceConstructor 检查是否为JSONSlice构造函数
func (ma *MappingAnalyzer) isJSONSliceConstructor(expr *ast.CallExpr) bool {
	if fun, ok := expr.Fun.(*ast.SelectorExpr); ok {
		if x, ok := fun.X.(*ast.Ident); ok {
			return x.Name == "datatypes" && fun.Sel.Name == "NewJSONSlice"
		}
	}
	return false
}

// extractJSONFields 提取JSON字段
func (ma *MappingAnalyzer) extractJSONFields(callExpr *ast.CallExpr, bFieldNameHint string) []string {
	var fields []string

	if len(callExpr.Args) == 0 {
		return fields
	}

	arg := callExpr.Args[0]
	if compLit, ok := arg.(*ast.CompositeLit); ok {
		// 这是 B_Token{...} 结构体字面量
		// 分析其中的字段映射到A类型的字段
		for _, elt := range compLit.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				aFields := ma.extractAFieldsFromExpr(kv.Value)
				if len(aFields) > 0 {
					fields = append(fields, aFields[0])
				}
			}
		}

		// 如果找到了JSON字段映射，添加到映射关系中
		if len(fields) > 0 {
			// 需要找到当前这个NewJSONType调用对应的是B结构体中的哪个字段
			// 通过分析上下文来确定具体的字段映射
			ma.processJSONFieldMapping(callExpr, compLit, fields, bFieldNameHint)
		}
	}

	// 对于JSON字段，返回空数组，因为它们会通过JSONMapping处理
	return []string{}
}

// processJSONFieldMapping 处理JSON字段映射
func (ma *MappingAnalyzer) processJSONFieldMapping(callExpr *ast.CallExpr, compLit *ast.CompositeLit, fields []string, bFieldNameHint string) {
	// 使用传入的字段名提示，如果为空则尝试分析
	bFieldName := bFieldNameHint
	if bFieldName == "" {
		bFieldName = ma.findBFieldNameForJSONCall(callExpr)
	}

	if bFieldName == "" {
		// 如果找不到对应的B字段，使用默认逻辑
		ma.processDefaultJSONMapping(compLit, fields)
		return
	}

	// 找到对应的B字段信息
	var targetBField *FieldInfo
	for _, bField := range ma.bType.Fields {
		if bField.Name == bFieldName {
			targetBField = &bField
			break
		}
	}

	if targetBField != nil && targetBField.IsJSONType {
		// 为这个特定的JSON字段创建映射
		jsonMapping := JSONMapping{
			FieldName: targetBField.GetColumnName(), // 数据库的字段名
			SubFields: make(map[string]string),
		}

		// 分析具体的JSON字段映射
		ma.analyzeJSONSubFields(compLit, &jsonMapping)
		ma.fieldMapping.JSONFields[targetBField.Name] = jsonMapping

		// 为每个A字段创建多对一映射关系，标记为JSON类型
		for _, aField := range fields {
			relation := MappingRelation{
				AField:     aField,
				BFields:    []string{targetBField.Name},
				IsJSONType: true,
				JSONField:  targetBField.Name,
				Order:      ma.currentOrder,
			}
			ma.mappingRel = append(ma.mappingRel, relation)
		}
		ma.currentOrder++
	}
}

// findBFieldNameForJSONCall 通过分析调用栈找到NewJSONType对应的B字段名
func (ma *MappingAnalyzer) findBFieldNameForJSONCall(callExpr *ast.CallExpr) string {
	// 这个方法需要分析当前函数的AST来找到这个NewJSONType调用被赋值给了哪个B字段
	// 由于实现比较复杂，暂时使用一个简化的方法
	// 通过检查当前上下文中的字段名来推断

	// 检查是否在分析过程中
	if ma.funcDecl == nil {
		return ""
	}

	var result string
	// 遍历函数体中的所有语句，寻找包含这个NewJSONType调用的赋值语句
	ast.Inspect(ma.funcDecl, func(n ast.Node) bool {
		if assign, ok := n.(*ast.AssignStmt); ok {
			for i, rhs := range assign.Rhs {
				if rhs == callExpr && i < len(assign.Lhs) {
					// 找到了赋值语句，分析左侧的选择器表达式
					if selector, ok := assign.Lhs[i].(*ast.SelectorExpr); ok {
						// 提取字段名，如 b.Token 中的 "Token"
						if ident, ok := selector.X.(*ast.Ident); ok && ident.Name == "b" {
							result = selector.Sel.Name
							return false // 找到了，停止遍历
						}
					}
				}
			}
		}
		return true
	})

	return result
}

// inferBFieldNameFromContext 从上下文推断B字段名
func (ma *MappingAnalyzer) inferBFieldNameFromContext(callExpr *ast.CallExpr) string {
	// 简单的策略：基于当前已处理的JSON字段数量来推断
	// 这是临时的解决方案，更准确的方案需要分析赋值语句

	// 计算已经映射的JSON字段数量
	processedCount := len(ma.fieldMapping.JSONFields)

	// 获取所有JSONType字段
	var jsonFields []FieldInfo
	for _, field := range ma.bType.Fields {
		if field.IsJSONType {
			jsonFields = append(jsonFields, field)
		}
	}

	// 按照字段在结构体中出现的顺序排序
	if processedCount < len(jsonFields) {
		return jsonFields[processedCount].Name
	}

	return ""
}

// processDefaultJSONMapping 默认的JSON映射处理（向后兼容）
func (ma *MappingAnalyzer) processDefaultJSONMapping(compLit *ast.CompositeLit, fields []string) {
	// 找到第一个可用的JSONType字段
	for _, bField := range ma.bType.Fields {
		if bField.IsJSONType {
			jsonMapping := JSONMapping{
				FieldName: bField.ColumnName,
			}
			if jsonMapping.FieldName == "" {
				jsonMapping.FieldName = ma.toSnakeCase(bField.Name)
			}
			jsonMapping.SubFields = make(map[string]string)

			// 分析具体的JSON字段映射
			ma.analyzeJSONSubFields(compLit, &jsonMapping)
			ma.fieldMapping.JSONFields[bField.Name] = jsonMapping

			// 为每个A字段创建多对一映射关系，标记为JSON类型
			for _, aField := range fields {
				relation := MappingRelation{
					AField:     aField,
					BFields:    []string{bField.Name},
					IsJSONType: true,
					JSONField:  bField.Name,
					Order:      ma.currentOrder,
				}
				ma.mappingRel = append(ma.mappingRel, relation)
			}
			ma.currentOrder++
			break
		}
	}
}

// analyzeJSONSubFields 分析JSON子字段映射
func (ma *MappingAnalyzer) analyzeJSONSubFields(compLit *ast.CompositeLit, jsonMapping *JSONMapping) {
	for _, elt := range compLit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			jsonFieldName := ma.extractFieldName(kv.Key)   // go的字段名，po对象
			aFields := ma.extractAFieldsFromExpr(kv.Value) // 右边，a对象

			if len(aFields) > 0 && jsonFieldName != "" {
				// 检查值是否为嵌套的复合字面量
				if nestedCompLit, ok := kv.Value.(*ast.CompositeLit); ok {
					// 这是嵌套结构，需要递归分析
					ma.analyzeNestedJSONField(jsonFieldName, nestedCompLit, jsonMapping)
				} else {
					// 这是简单字段
					// 尝试获取JSON标签名，如果没有则使用字段名的snake_case
					jsonTagName := ma.getJSONTagName(kv.Key, jsonFieldName)

					// 映射A字段到JSON子字段
					jsonMapping.SubFields[aFields[0]] = jsonTagName
				}
			} else if jsonFieldName != "" {
				// 即使没有找到A字段，也要检查是否是嵌套结构
				if nestedCompLit, ok := kv.Value.(*ast.CompositeLit); ok {
					ma.analyzeNestedJSONFieldWithoutAField(jsonFieldName, nestedCompLit, jsonMapping)
				}
			}
		}
	}
}

// analyzeNestedJSONField 分析嵌套JSON字段
func (ma *MappingAnalyzer) analyzeNestedJSONField(parentField string, compLit *ast.CompositeLit, jsonMapping *JSONMapping) {
	for _, elt := range compLit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			nestedFieldName := ma.extractFieldName(kv.Key)
			aFields := ma.extractAFieldsFromExpr(kv.Value)

			if len(aFields) > 0 && nestedFieldName != "" {
				// 构建嵌套的JSON字段名，如 "support_resource.placement_agreements"
				nestedJSONField := parentField + "." + nestedFieldName

				// 映射A字段到嵌套的JSON子字段
				jsonMapping.SubFields[aFields[0]] = nestedJSONField
			}
		}
	}
}

// analyzeNestedJSONFieldWithoutAField 分析没有直接A字段对应的嵌套JSON字段
func (ma *MappingAnalyzer) analyzeNestedJSONFieldWithoutAField(parentField string, compLit *ast.CompositeLit, jsonMapping *JSONMapping) {
	for _, elt := range compLit.Elts {
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			nestedFieldName := ma.extractFieldName(kv.Key)
			aFields := ma.extractAFieldsFromExpr(kv.Value)

			if len(aFields) > 0 && nestedFieldName != "" {
				// 构建嵌套的JSON字段名，如 "support_resource.placement_agreements"
				nestedJSONField := parentField + "." + nestedFieldName

				// 映射A字段到嵌套的JSON子字段
				jsonMapping.SubFields[aFields[0]] = nestedJSONField
			}
		}
	}
}

// getJSONTagName 获取字段的JSON标签名
func (ma *MappingAnalyzer) getJSONTagName(expr ast.Expr, fieldName string) string {
	// 这里需要查找字段定义来获取JSON标签
	// 由于我们处理的是B_Token结构体，需要在这个类型中查找字段定义

	// 首先尝试通过类型解析获取JSON标签
	if jsonTag := ma.getJSONTagFromType(expr); jsonTag != "" {
		return jsonTag
	}

	// 如果找不到JSON标签，返回snake_case形式
	return ma.toSnakeCase(fieldName)
}

// getJSONTagFromType 从类型定义中获取JSON标签
func (ma *MappingAnalyzer) getJSONTagFromType(expr ast.Expr) string {
	spew.Dump("getJSONTagFromType", expr)
	fieldName := ma.extractFieldName(expr)
	// 默认返回snake_case
	return ma.toSnakeCase(fieldName)
}

// buildFieldMapping 构建字段映射
func (ma *MappingAnalyzer) buildFieldMapping() {
	// 清空之前的映射
	clear(ma.fieldMapping.OneToOne)
	clear(ma.fieldMapping.OneToMany)
	clear(ma.fieldMapping.ManyToOne)

	// 按A字段分组映射关系，保持顺序
	aFieldMappings := make(map[string][]string)

	// 首先按照Order排序映射关系
	sortedRelations := make([]MappingRelation, len(ma.mappingRel))
	copy(sortedRelations, ma.mappingRel)

	// 简单的排序，按照Order字段
	for i := 0; i < len(sortedRelations)-1; i++ {
		for j := i + 1; j < len(sortedRelations); j++ {
			if sortedRelations[i].Order > sortedRelations[j].Order {
				sortedRelations[i], sortedRelations[j] = sortedRelations[j], sortedRelations[i]
			}
		}
	}

	for _, rel := range sortedRelations {
		if rel.AField == "" || len(rel.BFields) == 0 {
			continue
		}

		// 跳过JSON字段映射
		if rel.IsJSONType {
			continue
		}

		// 将B字段添加到A字段的映射中
		aFieldMappings[rel.AField] = append(aFieldMappings[rel.AField], rel.BFields...)
	}

	// 设置有序的映射关系（包含非JSON字段）
	ma.fieldMapping.OrderedRelations = []MappingRelation{}
	ma.fieldMapping.OrderedJSONRelations = []MappingRelation{}
	for _, rel := range sortedRelations {
		if rel.AField != "" && len(rel.BFields) > 0 {
			if rel.IsJSONType {
				ma.fieldMapping.OrderedJSONRelations = append(ma.fieldMapping.OrderedJSONRelations, rel)
			} else {
				ma.fieldMapping.OrderedRelations = append(ma.fieldMapping.OrderedRelations, rel)
			}
		}
	}

	// 构建最终映射
	for aField, bFields := range aFieldMappings {
		// 去重
		uniqueBFields := make(map[string]bool)
		var finalBFields []string
		for _, bField := range bFields {
			if !uniqueBFields[bField] {
				uniqueBFields[bField] = true
				finalBFields = append(finalBFields, bField)
			}
		}

		if len(finalBFields) == 1 {
			// 一对一映射
			ma.fieldMapping.OneToOne[aField] = finalBFields[0]
		} else if len(finalBFields) > 1 {
			// 一对多映射
			ma.fieldMapping.OneToMany[aField] = finalBFields
		}
	}

	// 重新构建有序关系，合并相同A字段的多个B字段
	finalOrderedRelations := make(map[string]MappingRelation)
	for _, rel := range ma.fieldMapping.OrderedRelations {
		if existing, exists := finalOrderedRelations[rel.AField]; exists {
			// 合并B字段
			existing.BFields = append(existing.BFields, rel.BFields...)
			// 去重
			uniqueFields := make(map[string]bool)
			var mergedBFields []string
			for _, bField := range existing.BFields {
				if !uniqueFields[bField] {
					uniqueFields[bField] = true
					mergedBFields = append(mergedBFields, bField)
				}
			}
			existing.BFields = mergedBFields
			finalOrderedRelations[rel.AField] = existing
		} else {
			finalOrderedRelations[rel.AField] = rel
		}
	}

	// 重新构建有序关系切片
	ma.fieldMapping.OrderedRelations = []MappingRelation{}
	for _, rel := range ma.mappingRel {
		if !rel.IsJSONType && rel.AField != "" {
			if merged, exists := finalOrderedRelations[rel.AField]; exists {
				// 检查是否已经添加过
				found := false
				for _, existing := range ma.fieldMapping.OrderedRelations {
					if existing.AField == merged.AField {
						found = true
						break
					}
				}
				if !found {
					ma.fieldMapping.OrderedRelations = append(ma.fieldMapping.OrderedRelations, merged)
				}
			}
		}
	}
}

// toSnakeCase 转换为snake_case
func (ma *MappingAnalyzer) toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	return string(result)
}
