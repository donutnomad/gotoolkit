package automap

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"slices"
	"strings"

	"github.com/donutnomad/gotoolkit/internal/gormparse"
	"github.com/donutnomad/gotoolkit/internal/structparse"
)

// collectTypeSpecs 收集所有类型定义和方法定义
// 会扫描同包中的所有Go文件，以便找到跨文件的类型定义和方法
func (m *Mapper) collectTypeSpecs() {
	// 首先从当前文件收集
	m.collectTypeSpecsFromFile(m.file)

	// 然后扫描同目录下的其他Go文件
	iterator := NewGoFileIterator(m.filePath)
	_ = iterator.Iterate(func(fullPath string) bool {
		otherFile, err := parser.ParseFile(m.fset, fullPath, nil, parser.ParseComments)
		if err != nil {
			if DebugMode {
				fmt.Printf("[DEBUG] Failed to parse file %s: %v\n", fullPath, err)
			}
			return true // 继续遍历
		}
		m.collectTypeSpecsFromFile(otherFile)
		return true
	})
}

// collectTypeSpecsFromFile 从单个文件收集类型定义和方法定义
func (m *Mapper) collectTypeSpecsFromFile(file *ast.File) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			// 收集类型定义
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					m.typeSpecs[typeSpec.Name.Name] = typeSpec
				}
			}

		case *ast.FuncDecl:
			// 收集方法定义
			if d.Recv == nil || len(d.Recv.List) == 0 {
				continue
			}
			recvType := extractTypeName(d.Recv.List[0].Type)
			if recvType == "" {
				continue
			}
			if m.methodDecls[recvType] == nil {
				m.methodDecls[recvType] = make(map[string]*ast.FuncDecl)
			}
			m.methodDecls[recvType][d.Name.Name] = d
		}
	}
}

// collectVariableAssignments 收集局部变量赋值
func (m *Mapper) collectVariableAssignments(body *ast.BlockStmt) {
	for _, stmt := range body.List {
		switch s := stmt.(type) {
		case *ast.AssignStmt:
			// 处理赋值语句: name := d.Name 或 name = d.Name
			m.processAssignStmt(s)

		case *ast.DeclStmt:
			// 处理变量声明: var name = d.Name
			if genDecl, ok := s.Decl.(*ast.GenDecl); ok && genDecl.Tok == token.VAR {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						m.processValueSpec(valueSpec)
					}
				}
			}

		case *ast.IfStmt:
			// 递归处理 if 语句块（只收集变量，不进入条件分支的覆盖）
			// 注意：我们只收集初始赋值，忽略条件分支中的重新赋值
			if s.Init != nil {
				if assignStmt, ok := s.Init.(*ast.AssignStmt); ok {
					m.processAssignStmt(assignStmt)
				}
			}

		case *ast.ForStmt:
			// 递归处理 for 语句的初始化部分
			if s.Init != nil {
				if assignStmt, ok := s.Init.(*ast.AssignStmt); ok {
					m.processAssignStmt(assignStmt)
				}
			}

		case *ast.RangeStmt:
			// 忽略 range 语句的迭代变量
		}
	}
}

// processAssignStmt 处理赋值语句
func (m *Mapper) processAssignStmt(s *ast.AssignStmt) {
	// 只处理简单赋值（:= 或 =）
	if len(s.Lhs) != len(s.Rhs) {
		return
	}

	for i, lhs := range s.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok {
			continue
		}

		varName := ident.Name
		rhs := s.Rhs[i]

		// 检查是否是方法调用：d.MethodName()
		if methodInfo := m.extractMethodCallInfo(rhs); methodInfo != nil {
			if _, exists := m.methodCallMap[varName]; !exists {
				m.methodCallMap[varName] = *methodInfo
			}
			continue
		}

		// 提取右侧的源路径
		sourcePath := m.extractSourcePathFromExpr(rhs)
		if sourcePath != "" {
			// 只记录第一次赋值（初始值）
			if _, exists := m.varMap[varName]; !exists {
				m.varMap[varName] = sourcePath
			}
		} else {
			// 检查是否是从另一个局部变量赋值
			if rhsIdent, ok := rhs.(*ast.Ident); ok {
				if existingPath, exists := m.varMap[rhsIdent.Name]; exists {
					if _, exists := m.varMap[varName]; !exists {
						m.varMap[varName] = existingPath
					}
				}
				// 检查是否是从另一个方法调用变量赋值
				if existingMethod, exists := m.methodCallMap[rhsIdent.Name]; exists {
					if _, exists := m.methodCallMap[varName]; !exists {
						m.methodCallMap[varName] = existingMethod
					}
				}
			}
		}
	}
}

// processValueSpec 处理变量声明
func (m *Mapper) processValueSpec(spec *ast.ValueSpec) {
	if len(spec.Names) != len(spec.Values) {
		return
	}

	for i, name := range spec.Names {
		varName := name.Name
		value := spec.Values[i]

		sourcePath := m.extractSourcePathFromExpr(value)
		if sourcePath != "" {
			m.varMap[varName] = sourcePath
		}
	}
}

// collectTargetColumns 收集目标类型（PO）的所有数据库列名
func (m *Mapper) collectTargetColumns() {
	// 使用 structparse 解析目标类型，它能正确处理外部包的嵌入类型
	// 首先尝试在当前文件中查找
	structInfo, err := structparse.ParseStruct(m.filePath, m.receiverType)
	if err != nil {
		if DebugMode {
			fmt.Printf("[DEBUG] structparse.ParseStruct failed for %s in %s: %v\n", m.receiverType, m.filePath, err)
			fmt.Println("[DEBUG] Trying to find struct in same directory...")
		}
		// 在同目录下的其他文件中查找
		structInfo, err = m.findStructInSameDirectory()
		if err != nil {
			if DebugMode {
				fmt.Printf("[DEBUG] Failed to find struct %s in same directory: %v\n", m.receiverType, err)
				fmt.Println("[DEBUG] Falling back to collectTargetColumnsSimple")
			}
			// 解析失败时回退到简单方式
			m.collectTargetColumnsSimple()
			return
		}
	}

	if DebugMode {
		fmt.Printf("[DEBUG] structparse.ParseStruct succeeded for %s, found %d fields:\n", m.receiverType, len(structInfo.Fields))
		for i, field := range structInfo.Fields {
			fmt.Printf("[DEBUG]   %d. Name=%s, Type=%s, Tag=%s, SourceType=%s, EmbeddedPrefix=%s\n",
				i+1, field.Name, field.Type, field.Tag, field.SourceType, field.EmbeddedPrefix)
		}
	}

	// 从解析结果中提取列名，同时记录字段位置
	for i, field := range structInfo.Fields {
		columnName := gormparse.ExtractColumnNameWithPrefix(field.Name, field.Tag, field.EmbeddedPrefix)
		m.result.TargetColumns = append(m.result.TargetColumns, columnName)
		m.result.TargetFieldPositions[columnName] = i
		if DebugMode {
			fmt.Printf("[DEBUG] Column: %s (from field %s, position %d)\n", columnName, field.Name, i)
		}
	}

	if DebugMode {
		fmt.Printf("[DEBUG] Total target columns: %d\n", len(m.result.TargetColumns))
	}
}

// findStructInSameDirectory 在同目录下的其他Go文件中查找结构体
func (m *Mapper) findStructInSameDirectory() (*structparse.StructInfo, error) {
	var result *structparse.StructInfo
	var lastErr error

	iterator := NewGoFileIterator(m.filePath)
	_ = iterator.Iterate(func(filePath string) bool {
		structInfo, err := structparse.ParseStruct(filePath, m.receiverType)
		if err == nil {
			if DebugMode {
				fmt.Printf("[DEBUG] Found struct %s in file: %s\n", m.receiverType, filePath)
			}
			result = structInfo
			return false // 找到后停止遍历
		}
		lastErr = err
		return true
	})

	if result != nil {
		return result, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("在目录中未找到结构体 %s: %w", m.receiverType, lastErr)
	}
	return nil, fmt.Errorf("在目录中未找到结构体 %s", m.receiverType)
}

// collectTargetColumnsSimple 简单方式收集列名（仅处理当前文件中的类型）
func (m *Mapper) collectTargetColumnsSimple() {
	typeSpec := m.typeSpecs[m.receiverType]
	if typeSpec == nil {
		return
	}

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return
	}

	columns := m.collectStructColumns(structType, "")
	m.result.TargetColumns = columns
	// 同时填充位置映射
	for i, col := range columns {
		m.result.TargetFieldPositions[col] = i
	}
}

// collectStructColumns 递归收集结构体的所有列名
func (m *Mapper) collectStructColumns(structType *ast.StructType, prefix string) []string {
	var columns []string

	for _, field := range structType.Fields.List {
		// 处理嵌入字段
		if len(field.Names) == 0 {
			embeddedTypeName := extractTypeName(field.Type)
			embeddedTypeSpec := m.typeSpecs[embeddedTypeName]

			// 获取嵌入字段的前缀
			embeddedPrefix := ""
			if field.Tag != nil {
				tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
				gormTag := tag.Get("gorm")
				for _, part := range strings.Split(gormTag, ";") {
					if strings.HasPrefix(part, "embeddedPrefix:") {
						embeddedPrefix = strings.TrimPrefix(part, "embeddedPrefix:")
					}
				}
			}

			if embeddedTypeSpec != nil {
				if embeddedStructType, ok := embeddedTypeSpec.Type.(*ast.StructType); ok {
					subColumns := m.collectStructColumns(embeddedStructType, prefix+embeddedPrefix)
					columns = append(columns, subColumns...)
				}
			}
			continue
		}

		// 处理命名字段
		for _, name := range field.Names {
			fieldName := name.Name
			// 跳过非导出字段
			if !isExported(fieldName) {
				continue
			}

			// 获取列名
			columnName := toSnakeCase(fieldName)
			if field.Tag != nil {
				tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
				gormTag := tag.Get("gorm")
				for _, part := range strings.Split(gormTag, ";") {
					if strings.HasPrefix(part, "column:") {
						columnName = strings.TrimPrefix(part, "column:")
					}
				}
			}

			columns = append(columns, prefix+columnName)
		}
	}

	return columns
}

// extractUsedFieldsFromMethod 从方法体中提取使用的字段
func (m *Mapper) extractUsedFieldsFromMethod(funcDecl *ast.FuncDecl) []string {
	if funcDecl.Body == nil {
		return nil
	}

	// 获取接收者名称
	var recvName string
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		recv := funcDecl.Recv.List[0]
		if len(recv.Names) > 0 {
			recvName = recv.Names[0].Name
		}
	}
	if recvName == "" {
		return nil
	}

	// 收集使用的字段
	usedFields := make(map[string]bool)
	ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// 检查是否是 recv.Field 形式
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == recvName {
			usedFields[sel.Sel.Name] = true
		}
		return true
	})

	// 转换为有序列表
	result := make([]string, 0, len(usedFields))
	for field := range usedFields {
		result = append(result, field)
	}

	// 按字母顺序排序以保证稳定性
	slices.Sort(result)

	return result
}
