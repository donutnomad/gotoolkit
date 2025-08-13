package generator

import (
	"github.com/donutnomad/gotoolkit/approveGen/types"
	"go/ast"
)

// GenMethodCallApprovalData 模板数据结构
type GenMethodCallApprovalData struct {
	GenMethodName          string
	AddUnmarshalMethodArgs bool
	Methods                []MyMethod
	EveryMethodSuffix      string
	DefaultSuccess         bool
	GetType                func(typ ast.Expr, method MyMethod) string
	HookRejectedMap        map[string]bool // v2版本使用，标记哪些方法支持 HookRejected
}

type MyMethod = types.MyMethod
