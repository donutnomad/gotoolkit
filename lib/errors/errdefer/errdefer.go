// Package errdefer 提供用于 defer 场景的错误处理函数。
// 这些函数操作 *error 指针，适合在 defer 中修改返回的错误。
package errdefer

import (
	"strings"

	"github.com/donutnomad/gotoolkit/lib/errors"
)

// Mark 用于在 defer 中给错误打标。
// 注意：必须传入 err 的指针 (&err)。
//
// 用法: defer errdefer.Mark(&err, ErrRetryable)
func Mark(errPtr *error, marker error) {
	MarkIf(errPtr, func(err error) bool {
		return true
	}, marker)
}

// MarkIf 只有当 predicate 返回 true 时，才给错误打标。
// 这是最灵活的方法，可用于判断 gRPC code、mysql error code 等。
//
// 用法: defer errdefer.MarkIf(&err, isGrpcUnavailable, ErrRetryable)
func MarkIf(errPtr *error, predicate func(error) bool, marker error, msg ...string) {
	if errPtr == nil || *errPtr == nil {
		return
	}
	if predicate(*errPtr) {
		var message string
		if len(msg) > 0 {
			message = strings.Join(msg, " ")
		}
		bd := errors.From(*errPtr).Mark(marker)
		if len(message) > 0 {
			bd = bd.WithMessage(message)
		}
		*errPtr = bd.Err()
	}
}

// MarkIfIs 只有当 errors.Is(err, target) 为 true 时，才打标。
// 适用于：把一种具体的错误（如 io.EOF）提升为一种业务错误（如 ErrUserNotFound）。
//
// 用法: defer errdefer.MarkIfIs(&err, os.ErrNotExist, ErrConfigFileMissing)
func MarkIfIs(errPtr *error, target error, marker error, msg ...string) {
	MarkIf(errPtr, func(err error) bool {
		return errors.Is(err, target)
	}, marker, msg...)
}
