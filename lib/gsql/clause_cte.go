package gsql

import (
	"github.com/donutnomad/gotoolkit/lib/gsql/field"
	"gorm.io/gorm/clause"
)

// CTEDefinition 表示单个 CTE 定义
type CTEDefinition struct {
	Name    string
	Columns []string // 可选的列名列表
	Query   field.Expression
}

// CTEClause 表示 WITH 子句（可包含多个 CTE）
type CTEClause struct {
	CTEs      []CTEDefinition
	Recursive bool
}

// Name implements clause.Interface
func (c CTEClause) Name() string {
	return "CTE"
}

// Build implements clause.Expression
func (c CTEClause) Build(builder clause.Builder) {
	if len(c.CTEs) == 0 {
		return
	}

	writer := &safeWriter{builder}

	// WITH [RECURSIVE]
	writer.WriteString("WITH ")
	if c.Recursive {
		writer.WriteString("RECURSIVE ")
	}

	// 构建每个 CTE
	for idx, cte := range c.CTEs {
		if idx > 0 {
			writer.WriteString(", ")
		}

		// CTE 名称
		writer.WriteQuoted(cte.Name)

		// 可选的列名列表
		if len(cte.Columns) > 0 {
			writer.WriteByte('(')
			for i, col := range cte.Columns {
				if i > 0 {
					writer.WriteString(", ")
				}
				writer.WriteQuoted(col)
			}
			writer.WriteByte(')')
		}

		// AS (query)
		writer.WriteString(" AS (")
		writer.AddVar(writer, cte.Query)
		writer.WriteByte(')')
	}

	writer.WriteByte(' ')
}

// MergeClause implements clause.Interface
func (c CTEClause) MergeClause(cl *clause.Clause) {
	if existing, ok := cl.Expression.(CTEClause); ok {
		// 合并 CTE 定义
		c.CTEs = append(existing.CTEs, c.CTEs...)
		c.Recursive = c.Recursive || existing.Recursive
	}
	cl.Expression = c
}

// cteBuilder 是私有的 CTE 构建器
type cteBuilder struct {
	ctes      []CTEDefinition
	recursive bool
}

// With 创建一个非递归 CTE
func With(name string, query *QueryBuilder, columns ...string) *cteBuilder {
	return &cteBuilder{
		ctes: []CTEDefinition{
			{
				Name:    name,
				Columns: columns,
				Query:   query.ToExpr(),
			},
		},
		recursive: false,
	}
}

// WithRecursive 创建一个递归 CTE
func WithRecursive(name string, query *QueryBuilder, columns ...string) *cteBuilder {
	return &cteBuilder{
		ctes: []CTEDefinition{
			{
				Name:    name,
				Columns: columns,
				Query:   query.ToExpr(),
			},
		},
		recursive: true,
	}
}

// And 添加另一个 CTE（链式调用）
func (b *cteBuilder) And(name string, query *QueryBuilder, columns ...string) *cteBuilder {
	b.ctes = append(b.ctes, CTEDefinition{
		Name:    name,
		Columns: columns,
		Query:   query.ToExpr(),
	})
	return b
}

// Select 开始主查询，返回 baseQueryBuilder
func (b *cteBuilder) Select(fields ...field.IField) *baseQueryBuilder {
	result := &baseQueryBuilder{
		selects: nil,
		cte: &CTEClause{
			CTEs:      b.ctes,
			Recursive: b.recursive,
		},
	}
	return result.Select(fields...)
}

// cteBuilderG 是泛型版本的私有 CTE 构建器
type cteBuilderG[T any] struct {
	ctes      []CTEDefinition
	recursive bool
}

// WithG 创建一个非递归 CTE（泛型版本）
func WithG[T any](name string, query *QueryBuilder, columns ...string) *cteBuilderG[T] {
	return &cteBuilderG[T]{
		ctes: []CTEDefinition{
			{
				Name:    name,
				Columns: columns,
				Query:   query.ToExpr(),
			},
		},
		recursive: false,
	}
}

// WithRecursiveG 创建一个递归 CTE（泛型版本）
func WithRecursiveG[T any](name string, query *QueryBuilder, columns ...string) *cteBuilderG[T] {
	return &cteBuilderG[T]{
		ctes: []CTEDefinition{
			{
				Name:    name,
				Columns: columns,
				Query:   query.ToExpr(),
			},
		},
		recursive: true,
	}
}

// And 添加另一个 CTE（链式调用）
func (b *cteBuilderG[T]) And(name string, query *QueryBuilder, columns ...string) *cteBuilderG[T] {
	b.ctes = append(b.ctes, CTEDefinition{
		Name:    name,
		Columns: columns,
		Query:   query.ToExpr(),
	})
	return b
}

// Select 开始主查询，返回 baseQueryBuilderG
func (b *cteBuilderG[T]) Select(fields ...field.IField) *baseQueryBuilderG[T] {
	result := &baseQueryBuilderG[T]{
		selects: nil,
		cte: &CTEClause{
			CTEs:      b.ctes,
			Recursive: b.recursive,
		},
	}
	return result.Select(fields...)
}
